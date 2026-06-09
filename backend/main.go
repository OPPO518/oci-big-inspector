package main

import (
	"bytes"
	"crypto/tls"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:embed dist/*
var frontendFS embed.FS

// 引入全局锁与控制通道，实现免重启热更新 Bot
var (
	tgBotCancel chan struct{}
	tgMu        sync.Mutex
	activeToken string
)

func main() {
	InitializeDB()

	// 异步拉起 Telegram 守护协程
	go func() {
		var token string
		_ = DB.QueryRow("SELECT value FROM system_config WHERE key = 'tg_bot_token'").Scan(&token)
		if token != "" {
			StartTgBotEngine(token)
		}
	}()

	publicFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		log.Fatalf("❌ 无法加载内嵌前端: %v", err)
	}
	frontendHandler := http.FileServer(http.FS(publicFS))

	// 统一路由入口
	http.HandleFunc("/", basicAuthWrapper(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/api/status":
				fmt.Fprintf(w, `{"status":"running","need_init":%v}`, checkInitNeeded())
			case "/api/accounts/add":
				HandleAddAccount(w, r)
			case "/api/system/init":
				HandleSystemInit(w, r)
			case "/api/accounts/list":
				HandleListAccounts(w, r)
			case "/api/accounts/test":
				HandleTestConnection(w, r)
			case "/api/system/config/get":
				HandleGetSystemConfig(w, r)
			case "/api/system/config/save":
				HandleSaveSystemConfig(w, r)
			default:
				http.Error(w, `{"error":"Not Found"}`, http.StatusNotFound)
			}
			return
		}
		frontendHandler.ServeHTTP(w, r)
	}))

	// 启动 HTTPS 逻辑
	target := getPublicIP()
	if target != "" {
		certDir := "/app/data/certs"
		_ = os.MkdirAll(certDir, 0755)
		certFile := filepath.Join(certDir, target+".cer")
		keyFile := filepath.Join(certDir, target+".key")

		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			runAcmeSubprocess(target, false)
		}
		startCertCheckTimer(target)

		if _, err := os.Stat(certFile); err == nil {
			log.Printf("🚀 大探长已启动，安全运行在 :443")
			server := &http.Server{
				Addr: ":443",
				TLSConfig: &tls.Config{
					GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
						cert, err := tls.LoadX509KeyPair(certFile, keyFile)
						if err != nil {
							return nil, err
						}
						return &cert, nil
					},
				},
			}
			log.Fatal(server.ListenAndServeTLS("", ""))
		}
	}
	log.Fatal(http.ListenAndServe(":443", nil))
}

// --- TELEGRAM 核心热启动引擎 ---

func StartTgBotEngine(token string) {
	tgMu.Lock()
	if tgBotCancel != nil {
		close(tgBotCancel) // 停掉老机器人的轮询
	}
	tgBotCancel = make(chan struct{})
	activeToken = token
	ch := tgBotCancel
	tgMu.Unlock()

	log.Printf("🤖 Telegram Bot 核心开始建立监听通道...")
	
	// 启动长轮询，用来接收发给 Bot 的指令 (如 /status, /2fa 等)
	go func() {
		offset := 0
		client := &http.Client{Timeout: 25 * time.Second}
		for {
			select {
			case <-ch:
				log.Println("🤖 旧 Telegram Bot 监听已安全注销。")
				return
			default:
				url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=20", activeToken, offset)
				resp, err := client.Get(url)
				if err != nil {
					time.Sleep(5 * time.Second)
					continue
				}
				
				var updateData struct {
					Ok     bool `json:"ok"`
					Result []struct {
						UpdateID int `json:"update_id"`
						Message  struct {
							Chat struct {
								ID int64 `json:"id"`
							} `json:"chat"`
							Text string `json:"text"`
						} `json:"message"`
					} `json:"result"`
				}
				
				if err := json.NewDecoder(resp.Body).Decode(&updateData); err == nil && updateData.Ok {
					for _, upd := range updateData.Result {
						offset = upd.UpdateID + 1
						
						// 预留点：在这里拦截处理来自管理员发来的交互指令
						cmd := strings.TrimSpace(upd.Message.Text)
						if cmd == "/status" {
							SendMessageToTG("📊 <b>当前开机任务状态：</b>\n暂无处于激活的刷机队列。")
						} else if cmd == "/2fa" {
							SendMessageToTG("🔑 <b>当前双因素验证码：</b>\n<code>749102</code> (示例，后期联动)")
						}
					}
				}
				resp.Body.Close()
			}
		}
	}()
}

// SendMessageToTG 全局推流核心接口
func SendMessageToTG(text string) {
	var token, chatID, enabled string
	_ = DB.QueryRow("SELECT value FROM system_config WHERE key = 'tg_bot_token'").Scan(&token)
	_ = DB.QueryRow("SELECT value FROM system_config WHERE key = 'tg_chat_id'").Scan(&chatID)
	_ = DB.QueryRow("SELECT value FROM system_config WHERE key = 'tg_notify_enabled'").Scan(&enabled)

	if enabled != "1" || token == "" || chatID == "" {
		return
	}

	go func() {
		url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
		payload, _ := json.Marshal(map[string]interface{}{
			"chat_id":    chatID,
			"text":       text,
			"parse_mode": "HTML",
		})
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
}

// --- 系统辅助功能 ---

func checkInitNeeded() bool {
	var count int
	_ = DB.QueryRow("SELECT COUNT(*) FROM system_config WHERE key = 'username'").Scan(&count)
	return count == 0
}

func getPublicIP() string {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.ipify.org")
	if err == nil { defer resp.Body.Close(); ip, _ := io.ReadAll(resp.Body); return strings.TrimSpace(string(ip)) }
	return ""
}

func runAcmeSubprocess(target string, isCron bool) {
	acmePath := "/root/.acme.sh/acme.sh"
	if _, err := os.Stat(acmePath); os.IsNotExist(err) { acmePath = "acme.sh" }
	args := []string{"--issue", "-d", target, "--standalone", "--server", "letsencrypt", "--insecure", "--certificate-profile", "shortlived", "--fullchain-file", "/app/data/certs/" + target + ".cer", "--key-file", "/app/data/certs/" + target + ".key"}
	if isCron { args = []string{"--cron"} }
	cmd := exec.Command(acmePath, args...)
	_ = cmd.Run()
}

func startCertCheckTimer(target string) {
	go func() {
		for range time.Tick(24 * time.Hour) { runAcmeSubprocess(target, true) }
	}()
}

func basicAuthWrapper(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/system/init") || r.URL.Path == "/" || strings.Contains(r.URL.Path, ".") {
			next(w, r); return
		}
		next(w, r)
	}
}

func HandleGetSystemConfig(w http.ResponseWriter, r *http.Request) {
	res := make(map[string]string)
	rows, err := DB.Query("SELECT key, value FROM system_config WHERE key IN ('tg_bot_token', 'tg_chat_id', 'tg_notify_enabled')")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var k, v string
			if rows.Scan(&k, &v) == nil { res[k] = v }
		}
	}
	json.NewEncoder(w).Encode(res)
}

func HandleSaveSystemConfig(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	_ = json.NewDecoder(r.Body).Decode(&req)
	
	var newToken string
	for k, v := range req { // 👈 核心修复：在这里加上了 :=
		_, _ = DB.Exec("INSERT OR REPLACE INTO system_config (key, value) VALUES (?, ?)", k, v)
		if k == "tg_bot_token" {
			newToken = v
		}
	}

	// 核心热启动触发：如果用户修改或填入了有效的 Bot Token，内存当场刷新并通知 TG
	if newToken != "" {
		StartTgBotEngine(newToken)
		// 如果开启了通知，直接推送测试喜报
		if req["tg_notify_enabled"] == "1" {
			go func() {
				time.Sleep(1 * time.Second) // 留出一秒等数据库事务写完
				SendMessageToTG("🤖 <b>大探长 OCI 控制台喜报</b>\n\n🎉 您的系统配置与 Telegram Bot 已成功测通！\n📡 后端常驻协程开始实时监听指令。\n\n💡 <b>目前可用指令：</b>\n<code>/status</code> - 查看实时开机状态\n<code>/2fa</code> - 获取即时登录码")
			}()
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success"}`))
}

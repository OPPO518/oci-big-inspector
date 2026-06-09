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
	"time"
)

//go:embed dist/*
var frontendFS embed.FS

func main() {
	InitializeDB()

	// 异步拉起 Telegram 常驻守护协程，负责在后台监听指令
	go InitTelegramBot()

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

// --- TELEGRAM NOTIFICATION ROUTER (TG 通知总线) ---

// InitTelegramBot 负责长期驻留后台，处理外部发来的指令
func InitTelegramBot() {
	log.Println("🤖 Telegram 守护协程已就位，等待配置激活...")
	for {
		// 此处预留长轮询监听 /2fa 或 /status 指令
		time.Sleep(30 * time.Second)
	}
}

// SendMessageToTG 提供给全局任何模块调用的主动推流接口
func SendMessageToTG(text string) {
	var token, chatID, enabled string
	_ = DB.QueryRow("SELECT value FROM system_config WHERE key = 'tg_bot_token'").Scan(&token)
	_ = DB.QueryRow("SELECT value FROM system_config WHERE key = 'tg_chat_id'").Scan(&chatID)
	_ = DB.QueryRow("SELECT value FROM system_config WHERE key = 'tg_notify_enabled'").Scan(&enabled)

	if enabled != "1" || token == "" || chatID == "" {
		return
	}

	// 异步发送，防止阻塞主业务引擎
	go func() {
		url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
		payload, _ := json.Marshal(map[string]string{
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

// --- 支撑逻辑 ---

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

// HandleGetSystemConfig 获取安全与 TG 配置
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

// HandleSaveSystemConfig 保存系统配置
func HandleSaveSystemConfig(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	_ = json.NewDecoder(r.Body).Decode(&req)
	for k, v := range req {
		_, _ = DB.Exec("INSERT OR REPLACE INTO system_config (key, value) VALUES (?, ?)", k, v)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success"}`))
}

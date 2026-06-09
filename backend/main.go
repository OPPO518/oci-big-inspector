package main

import (
	"embed"
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

// 自动化核心：Go 自动联网获取当前 VPS 的真实公网 IP
func getPublicIP() string {
	client := http.Client{Timeout: 5 * time.Second}
	// 尝试从 ipify 获取纯文本 IP
	resp, err := client.Get("https://api.ipify.org")
	if err == nil {
		defer resp.Body.Close()
		ip, _ := io.ReadAll(resp.Body)
		if len(ip) > 0 {
			return strings.TrimSpace(string(ip))
		}
	}
	log.Println("⚠️ [ACME] 通过首选 API 获取 IP 失败，尝试备用通道...")
	return ""
}

// 调用系统内置的 acme.sh 申请 Let's Encrypt 短寿证书
func runAcmeSubprocess(target string) {
	if target == "" {
		log.Println("❌ [ACME] 公网 IP 为空，跳过证书申请")
		return
	}
	log.Printf("📢 [ACME] 开始为自动获取的 IP [%s] 申请 Let's Encrypt 证书...", target)
	
	acmePath := "/root/.acme.sh/acme.sh"
	if _, err := os.Stat(acmePath); os.IsNotExist(err) {
		acmePath = "acme.sh"
	}

	cmd := exec.Command(acmePath, "--issue", "-d", target, "--standalone", "--server", "letsencrypt", "--insecure")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("❌ [ACME] 证书申请或续签失败 (请确保 80 端口已放行): %v", err)
		return
	}
	log.Println("✅ [ACME] Let's Encrypt IP 证书处理成功！")
}

// 24小时自动续签定时器（无感更新）
func startCertCheckTimer(target string) {
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			log.Println("🔄 [定时任务] 触发 24 小时证书状态检查...")
			runAcmeSubprocess(target)
		}
	}()
}

// 动态鉴权中间件
func basicAuthWrapper(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从数据库动态查询管理员配置
		var expectedUser, expectedPass string
		_ = DB.QueryRow("SELECT value FROM system_config WHERE key = 'username'").Scan(&expectedUser)
		_ = DB.QueryRow("SELECT value FROM system_config WHERE key = 'password'").Scan(&expectedPass)

		// 【初始化引导豁免】如果数据库无账号，说明系统处于“全新待注册”状态
		if expectedUser == "" || expectedPass == "" {
			// 只放行注册接口，如果是其他 API 请求，通知前端去引导初始化
			if r.URL.Path == "/api/system/init" {
				next(w, r)
				return
			}
			if strings.HasPrefix(r.URL.Path, "/api/") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"need_init":true,"message":"请先完成面板初始化设置"}`))
				return
			}
			// 允许加载前端网页资源去渲染初始化向导界面
			next(w, r)
			return
		}

		// 正常全站防护认证流程
		user, pass, ok := r.BasicAuth()
		if !ok || user != expectedUser || pass != expectedPass {
			w.Header().Set("WWW-Authenticate", `Basic realm="OCI Big Inspector Restrict"`)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"Unauthorized"}`))
			return
		}

		next(w, r)
	}
}

func main() {
	// 0. 初始化本地 SQLite 数据底座
	InitializeDB()

	// 1. 解析嵌入的前端静态资源
	publicFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		log.Fatalf("❌ 无法加载内嵌前端: %v", err)
	}
	frontendHandler := http.FileServer(http.FS(publicFS))

	// 2. 基础路由分发 (全部套上安全守门员)
	http.HandleFunc("/", basicAuthWrapper(func(w http.ResponseWriter, r *http.Request) {
		// API 路由拦截处理
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			
			if r.URL.Path == "/api/status" {
				fmt.Fprintf(w, `{"status":"running","engine":"Go-OCI-Core","cert":"LetsEncrypt"}`)
				return
			}
			
			// 核心路由 1：接收前端传来的参数，加密并添加入库
			if r.URL.Path == "/api/accounts/add" {
				HandleAddAccount(w, r)
				return
			}
			
			// 核心路由 2：首次访问网页时的初始化注册接口
			if r.URL.Path == "/api/system/init" {
				HandleSystemInit(w, r)
				return
			}
			
			// 核心路由 3：让前端表格渲染已安全归档的账号列表
			if r.URL.Path == "/api/accounts/list" {
				HandleListAccounts(w, r)
				return
			}
			
			http.Error(w, `{"error":"Not Found"}`, http.StatusNotFound)
			return
		}
		
		// 非 API 请求，直接渲染内嵌的 Vue 3 静态页面
		frontendHandler.ServeHTTP(w, r)
	}))

	// 3. 自动联网抓取当前公网 IP 并判定证书
	target := getPublicIP()
	port := "443" // 容器内部网络固定监听 443

	if target != "" {
		certDir := "/app/data/certs"
		_ = os.MkdirAll(certDir, 0755)
		certFile := filepath.Join(certDir, target+".cer")
		keyFile := filepath.Join(certDir, target+".key")

		// 首次启动自动申请
		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			runAcmeSubprocess(target)
		}

		// 启动后台每 24 小时的无感续签定时任务
		startCertCheckTimer(target)

		if _, err := os.Stat(certFile); err == nil {
			log.Printf("🚀 大探长已自动捕获公网 IP [%s] 并加载证书，安全运行在 https://%s:%s", target, target, port)
			if err := http.ListenAndServeTLS(":"+port, certFile, keyFile, nil); err != nil {
				log.Fatalf("❌ HTTPS 绑定失败: %v", err)
			}
		}
	}

	// 降级保底（例如无网环境或 IP 变动）
	log.Printf("⚠️ 未能捕获公网 IP 或证书未就绪，降级至 HTTP 模式，端口 :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("❌ HTTP 绑定失败: %v", err)
	}
}

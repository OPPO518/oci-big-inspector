package main

import (
	"embed"
	"fmt"
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

// 调用系统内置的 acme.sh 申请 Let's Encrypt 6天短寿 IP 证书
func runAcmeSubprocess(target string) {
	log.Printf("📢 [ACME] 开始为目标 IP/域名 [%s] 申请 Let's Encrypt 证书...", target)
	
	acmePath := "/root/.acme.sh/acme.sh"
	if _, err := os.Stat(acmePath); os.IsNotExist(err) {
		acmePath = "acme.sh"
	}

	cmd := exec.Command(acmePath, "--issue", "-d", target, "--standalone", "--server", "letsencrypt", "--insecure")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("❌ [ACME] 证书申请或续签失败 (请确保宿主机 80 端口已映射并放行): %v", err)
		return
	}
	log.Println("✅ [ACME] Let's Encrypt IP/域名证书处理成功！")
}

// 定时任务：每 24 小时检查并自动续签证书
func startCertCheckTimer(target string) {
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			log.Println("🔄 [定时任务] 触发 24 小时证书状态检查...")
			runAcmeSubprocess(target)
		}
	}()
}

// 【新增核心：安全守门员】HTTP 基础认证中间件
func basicAuthWrapper(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从环境变量读取设定的管理员账号密码
		expectedUser := os.Getenv("WEB_USERNAME")
		expectedPass := os.Getenv("WEB_PASSWORD")

		// 兜底防御：如果部署时忘了写环境变量，强制阻断访问，并在日志中警告
		if expectedUser == "" || expectedPass == "" {
			log.Println("⚠️ [安全警告] 未配置 WEB_USERNAME 或 WEB_PASSWORD 环境变量，系统已自动锁定拒绝所有访问！")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"服务器未配置安全凭证，拒绝访问"}`))
			return
		}

		// 获取浏览器传来的认证信息
		user, pass, ok := r.BasicAuth()
		if !ok || user != expectedUser || pass != expectedPass {
			// 认证失败，向浏览器发送弹窗指令
			w.Header().Set("WWW-Authenticate", `Basic realm="OCI Big Inspector Restrict"`)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"Unauthorized"}`))
			return
		}

		// 认证成功，继续执行后续的网页或 API 逻辑
		next(w, r)
	}
}

func main() {
	// 0. 优先初始化本地安全数据库
	InitializeDB()

	// 1. 解析嵌入的前端静态资源
	publicFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		log.Fatalf("❌ 无法加载内嵌前端: %v", err)
	}
	frontendHandler := http.FileServer(http.FS(publicFS))

	// 2. 基础路由分发 (全部套上安全守门员 basicAuthWrapper)
	http.HandleFunc("/", basicAuthWrapper(func(w http.ResponseWriter, r *http.Request) {
		// API 路由分支
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			
			if r.URL.Path == "/api/status" {
				fmt.Fprintf(w, `{"status":"running","engine":"Go-OCI-Core","cert":"LetsEncrypt"}`)
				return
			}
			if r.URL.Path == "/api/accounts/add" {
				HandleAddAccount(w, r)
				return
			}
			
			http.Error(w, `{"error":"Not Found"}`, http.StatusNotFound)
			return
		}
		
		// 静态前端资源页面
		frontendHandler.ServeHTTP(w, r)
	}))

	// 3. 读取核心环境变量
	target := os.Getenv("SERVER_TARGET") // VPS 纯公网 IP 或域名
	port := "443"                        // 容器内部始终固定监听标准的 443 HTTPS 端口

	// 4. 安全启动判定
	if target != "" {
		certDir := "/app/data/certs"
		_ = os.MkdirAll(certDir, 0755)
		
		certFile := filepath.Join(certDir, target+".cer")
		keyFile := filepath.Join(certDir, target+".key")

		// 首次启动若无证书则去申请
		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			runAcmeSubprocess(target)
		}

		// 启动后台高频续签轮询
		startCertCheckTimer(target)

		if _, err := os.Stat(certFile); err == nil {
			log.Printf("🚀 大探长已成功加载 Let's Encrypt 证书，安全运行在容器内 :%s 端口", port)
			if err := http.ListenAndServeTLS(":"+port, certFile, keyFile, nil); err != nil {
				log.Fatalf("❌ HTTPS 绑定失败: %v", err)
			}
		}
	}

	// 保底降级
	log.Printf("⚠️ 未提供 SERVER_TARGET 或证书未就绪，降级至 HTTP 模式，端口 :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("❌ HTTP 绑定失败: %v", err)
	}
}

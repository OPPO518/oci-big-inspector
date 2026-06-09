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

	// standalone 独立模式占用 80 端口验证，向 Let's Encrypt 申请短寿证书
	cmd := exec.Command(acmePath, "--issue", "-d", target, "--standalone", "--server", "letsencrypt", "--insecure")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("❌ [ACME] 证书申请或续签失败 (请确保宿主机 80 端口已映射并放行): %v", err)
		return
	}
	log.Println("✅ [ACME] Let's Encrypt IP/域名证书处理成功！")
}

// 后端守护进程：每 24 小时检查一次证书并自动续签
func startCertCheckTimer(target string) {
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			log.Println("🔄 [定时任务] 触发 24 小时证书状态检查...")
			runAcmeSubprocess(target)
		}
	}()
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

	// 2. 基础路由分发
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 预留的 API 接口前缀
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/api/status" {
				fmt.Fprintf(w, `{"status":"running","engine":"Go-OCI-Core","cert":"LetsEncrypt-ShortLived"}`)
				return
			}
			http.Error(w, `{"error":"Not Found"}`, http.StatusNotFound)
			return
		}
		// 静态前端资源
		frontendHandler.ServeHTTP(w, r)
	})

	// 3. 读取核心环境变量
	target := os.Getenv("SERVER_TARGET") // VPS 纯公网 IP 或域名
	port := os.Getenv("WEB_PORT")
	if port == "" {
		port = "443"
	}

	// 4. 安全启动判定
	if target != "" {
		certDir := "/app/data/certs"
		_ = os.MkdirAll(certDir, 0755)
		
		certFile := filepath.Join(certDir, target+".cer")
		keyFile := filepath.Join(certDir, target+".key")

		// 首次启动：如果没有证书，立刻执行一次申请
		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			runAcmeSubprocess(target)
		}

		// 启动后台高频续签轮询
		startCertCheckTimer(target)

		if _, err := os.Stat(certFile); err == nil {
			log.Printf("🚀 大探长已成功加载 Let's Encrypt 证书，安全运行在 https://%s:%s", target, port)
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

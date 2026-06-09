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
	// 尝试从 ipify 获取
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

// 调用系统内置的 acme.sh 申请 Let's Encrypt 证书
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
		log.Printf("❌ [ACME] 证书申请或续签失败: %v", err)
		return
	}
	log.Println("✅ [ACME] Let's Encrypt IP 证书处理成功！")
}

// 24小时自动续签定时器
func startCertCheckTimer(target string) {
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			runAcmeSubprocess(target)
		}
	}()
}

// 【动态鉴权】从 SQLite 数据库动态读取用户在网页上设置的用户名密码
func basicAuthWrapper(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从数据库查询配置
		var expectedUser, expectedPass string
		_ = DB.QueryRow("SELECT value FROM system_config WHERE key = 'username'").Scan(&expectedUser)
		_ = DB.QueryRow("SELECT value FROM system_config WHERE key = 'password'").Scan(&expectedPass)

		// 【Setup Wizard 豁免】如果数据库里还没有账号密码，说明系统处于“待初始化”状态，直接放行，让前端显示初始化注册页面
		if expectedUser == "" || expectedPass == "" {
			// 如果前端请求的是注册接口，允许放行，否则全部交给前端渲染初始化引导
			if r.URL.Path == "/api/system/init" {
				next(w, r)
				return
			}
			// 允许加载前端资源去显示初始化表单
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				next(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"need_init":true,"message":"请先完成面板初始化设置"}`))
			return
		}

		// 正常认证流程
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
	InitializeDB()

	publicFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		log.Fatalf("❌ 无法加载内嵌前端: %v", err)
	}
	frontendHandler := http.FileServer(http.FS(publicFS))

	// 统一包裹安全中间件
	http.HandleFunc("/", basicAuthWrapper(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			
			if r.URL.Path == "/api/status" {
				fmt.Fprintf(w, `{"status":"running","engine":"Go-OCI-Core"}`)
				return
			}
			if r.URL.Path == "/api/accounts/add" {
				HandleAddAccount(w, r)
				return
			}
			// 预留给前端初始化提交账号密码的接口
			if r.URL.Path == "/api/system/init" {
				// 后续在这里实现将前端传来的管理员用户名密码写入 system_config 表即可
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"success"}`))
				return
			}
			
			http.Error(w, `{"error":"Not Found"}`, http.StatusNotFound)
			return
		}
		frontendHandler.ServeHTTP(w, r)
	}))

	// 自动获取公网 IP 并申请证书
	target := getPublicIP()
	port := "443"

	if target != "" {
		certDir := "/app/data/certs"
		_ = os.MkdirAll(certDir, 0755)
		certFile := filepath.Join(certDir, target+".cer")
		keyFile := filepath.Join(certDir, target+".key")

		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			runAcmeSubprocess(target)
		}

		startCertCheckTimer(target)

		if _, err := os.Stat(certFile); err == nil {
			log.Printf("🚀 大探长已自动捕获公网 IP [%s] 并加载证书，安全运行在 https://%s:%s", target, target, port)
			if err := http.ListenAndServeTLS(":"+port, certFile, keyFile, nil); err != nil {
				log.Fatalf("❌ HTTPS 绑定失败: %v", err)
			}
		}
	}

	log.Printf("⚠️ 未能捕获公网 IP 或证书未就绪，降级至 HTTP 模式，端口 :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("❌ HTTP 绑定失败: %v", err)
	}
}

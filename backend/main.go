package main

import (
	"crypto/tls"
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

func getPublicIP() string {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.ipify.org")
	if err == nil {
		defer resp.Body.Close()
		ip, _ := io.ReadAll(resp.Body)
		if len(ip) > 0 {
			return strings.TrimSpace(string(ip))
		}
	}
	return ""
}

func runAcmeSubprocess(target string, isCron bool) {
	if target == "" {
		return
	}
	
	acmePath := "/root/.acme.sh/acme.sh"
	if _, err := os.Stat(acmePath); os.IsNotExist(err) {
		acmePath = "acme.sh"
	}

	var cmd *exec.Cmd
	if isCron {
		log.Println("🔄 [ACME] 触发定时巡检，acme.sh 智能检测短寿证书是否需要续签...")
		// 使用 acme.sh 原生的 cron 指令，安全且智能
		cmd = exec.Command(acmePath, "--cron")
	} else {
		log.Printf("📢 [ACME] 开始为 IP [%s] 申请 Let's Encrypt 6天短寿证书...", target)
		certDir := "/app/data/certs"
		_ = os.MkdirAll(certDir, 0755)
		certFile := filepath.Join(certDir, target+".cer")
		keyFile := filepath.Join(certDir, target+".key")

		// 【核心修复】增加 --certificate-profile shortlived 满足 Let's Encrypt 强制要求
		// 增加 --fullchain-file 和 --key-file 将生成的证书物理导出到我们的挂载目录
		cmd = exec.Command(acmePath, "--issue", "-d", target, "--standalone", "--server", "letsencrypt", "--insecure", 
			"--certificate-profile", "shortlived",
			"--fullchain-file", certFile,
			"--key-file", keyFile)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("❌ [ACME] 进程执行异常: %v", err)
		return
	}
	if !isCron {
		log.Println("✅ [ACME] Let's Encrypt IP 短寿证书签发并导出成功！")
	}
}

func startCertCheckTimer(target string) {
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			runAcmeSubprocess(target, true)
		}
	}()
}

func basicAuthWrapper(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var expectedUser, expectedPass string
		_ = DB.QueryRow("SELECT value FROM system_config WHERE key = 'username'").Scan(&expectedUser)
		_ = DB.QueryRow("SELECT value FROM system_config WHERE key = 'password'").Scan(&expectedPass)

		if expectedUser == "" || expectedPass == "" {
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
			next(w, r)
			return
		}

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

	http.HandleFunc("/", basicAuthWrapper(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			
			if r.URL.Path == "/api/status" {
				fmt.Fprintf(w, `{"status":"running"}`)
				return
			}
			if r.URL.Path == "/api/accounts/add" {
				HandleAddAccount(w, r)
				return
			}
			if r.URL.Path == "/api/system/init" {
				HandleSystemInit(w, r)
				return
			}
			if r.URL.Path == "/api/accounts/list" {
				HandleListAccounts(w, r)
				return
			}
			if r.URL.Path == "/api/accounts/test" {
				HandleTestConnection(w, r)
				return
			}
			
			http.Error(w, `{"error":"Not Found"}`, http.StatusNotFound)
			return
		}
		frontendHandler.ServeHTTP(w, r)
	}))

	target := getPublicIP()
	port := "443"

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
			log.Printf("🚀 大探长已自动捕获公网 IP [%s] 并加载证书，安全运行在 :%s 端口", target, port)
			
			server := &http.Server{
				Addr: ":" + port,
				TLSConfig: &tls.Config{
					// 【黑科技】每次有新浏览器连接时，动态从磁盘读取闭包，实现 6 天证书无感热重载
					GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
						cert, err := tls.LoadX509KeyPair(certFile, keyFile)
						if err != nil {
							return nil, err
						}
						return &cert, nil
					},
				},
			}
			
			if err := server.ListenAndServeTLS("", ""); err != nil {
				log.Fatalf("❌ HTTPS 绑定失败: %v", err)
			}
		}
	}

	log.Printf("⚠️ 未能捕获有效公网证书，降级至 HTTP 模式，端口 :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("❌ HTTP 绑定失败: %v", err)
	}
}

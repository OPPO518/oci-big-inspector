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

func runAcmeSubprocess(target string) {
	if target == "" {
		return
	}
	acmePath := "/root/.acme.sh/acme.sh"
	if _, err := os.Stat(acmePath); os.IsNotExist(err) {
		acmePath = "acme.sh"
	}
	cmd := exec.Command(acmePath, "--issue", "-d", target, "--standalone", "--server", "letsencrypt", "--insecure")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

func startCertCheckTimer(target string) {
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			runAcmeSubprocess(target)
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
			// 【新增加的路由】指向刚才写好的测试联通函数
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
			runAcmeSubprocess(target)
		}
		startCertCheckTimer(target)

		if _, err := os.Stat(certFile); err == nil {
			log.Printf("🚀 大探长已自动捕获公网 IP [%s] 并加载证书，安全运行在 :%s 端口", target, port)
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

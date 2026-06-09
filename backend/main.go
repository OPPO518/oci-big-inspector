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

func main() {
	InitializeDB()

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
						return tls.LoadX509KeyPair(certFile, keyFile)
					},
				},
			}
			log.Fatal(server.ListenAndServeTLS("", ""))
		}
	}
	log.Fatal(http.ListenAndServe(":443", nil))
}

// --- 辅助逻辑 ---

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
		// 校验 Auth 的逻辑 ...
		next(w, r)
	}
}

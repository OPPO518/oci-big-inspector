package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// --- 数据模型 ---

type AddAccountRequest struct {
	Alias       string `json:"alias"`
	TenancyID   string `json:"tenancy_id"`
	UserID      string `json:"user_id"`
	Fingerprint string `json:"fingerprint"`
	Region      string `json:"region"`
	PrivateKey  string `json:"private_key"`
	RawConfig   string `json:"raw_config"` // 用于粘贴解析
}

type SysInitRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AccountConfigResponse struct {
	ID          int    `json:"id"`
	Alias       string `json:"alias"`
	TenancyID   string `json:"tenancy_id"`
	Region      string `json:"region"`
	Fingerprint string `json:"fingerprint"`
}

type TestAccountRequest struct {
	ID int `json:"id"`
}

// --- 辅助函数 ---

// parseOCIConfig 解析类似 OCI config 文件的字符串
func parseOCIConfig(content string) (map[string]string, error) {
	result := make(map[string]string)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		// 跳过注释
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result, nil
}

// --- 处理器 ---

// HandleSystemInit 处理系统初始化
func HandleSystemInit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var count int
	_ = DB.QueryRow("SELECT COUNT(*) FROM system_config WHERE key = 'username'").Scan(&count)
	if count > 0 {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"系统已经初始化过"}`))
		return
	}

	var req SysInitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"非法的输入"}`))
		return
	}

	_, err := DB.Exec("INSERT OR REPLACE INTO system_config (key, value) VALUES ('username', ?), ('password', ?)", req.Username, req.Password)
	if err != nil {
		log.Printf("系统初始化失败: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success"}`))
}

// HandleAddAccount 处理添加账户，支持 RawConfig 解析
func HandleAddAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req AddAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 如果粘贴了配置，进行解析覆盖
	if req.RawConfig != "" {
		conf, _ := parseOCIConfig(req.RawConfig)
		if v, ok := conf["tenancy"]; ok { req.TenancyID = v }
		if v, ok := conf["user"]; ok { req.UserID = v }
		if v, ok := conf["fingerprint"]; ok { req.Fingerprint = v }
		if v, ok := conf["region"]; ok { req.Region = v }
	}

	if req.TenancyID == "" || req.UserID == "" || req.Fingerprint == "" || req.PrivateKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"参数不完整，请检查配置或手动填写"}`))
		return
	}

	encryptedKey, err := EncryptText(req.PrivateKey)
	if err != nil {
		log.Printf("加密模块异常: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	query := `INSERT INTO oci_accounts (alias, tenancy_id, user_id, fingerprint, region, encrypted_key) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = DB.Exec(query, req.Alias, req.TenancyID, req.UserID, req.Fingerprint, req.Region, encryptedKey)
	if err != nil {
		log.Printf("数据库写入异常: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success"}`))
}

// HandleListAccounts 获取账户列表
func HandleListAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	rows, err := DB.Query("SELECT id, alias, tenancy_id, region, fingerprint FROM oci_accounts")
	if err != nil {
		log.Printf("查询数据库异常: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var list []AccountConfigResponse
	for rows.Next() {
		var acc AccountConfigResponse
		if err := rows.Scan(&acc.ID, &acc.Alias, &acc.TenancyID, &acc.Region, &acc.Fingerprint); err == nil {
			list = append(list, acc)
		}
	}
	json.NewEncoder(w).Encode(list)
}

// HandleTestConnection 测试连接
func HandleTestConnection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req TestAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tenantName, err := TestOCILink(req.ID)
	if err != nil {
		log.Printf("OCI 连接测试失败: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "连接失败: " + err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":      "success",
		"tenant_name": tenantName,
	})
}

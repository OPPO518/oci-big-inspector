package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// --- 数据模型 ---

type AddAccountRequest struct {
	Alias       string `json:"alias"`
	TenancyID   string `json:"tenancy_id"`
	UserID      string `json:"user_id"`
	Fingerprint string `json:"fingerprint"`
	Region      string `json:"region"`
	PrivateKey  string `json:"private_key"`
	RawConfig   string `json:"raw_config"` 
	Proxy       string `json:"proxy"`      
}

type SysInitRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AccountConfigResponse struct {
	ID            int    `json:"id"`
	Alias         string `json:"alias"` 
	TenancyID     string `json:"tenancy_id"`
	TenantName    string `json:"tenant_name"` 
	Region        string `json:"region"`
	Fingerprint   string `json:"fingerprint"`
	AccountType   string `json:"account_type"`
	IsMultiRegion bool   `json:"is_multi_region"`
	CreatedAt     string `json:"created_at"` 
	AliveDays     int    `json:"alive_days"`
	HasBootTask   bool   `json:"has_boot_task"`
	Status        string `json:"status"`
	Proxy         string `json:"proxy"`
}

type TestAccountRequest struct {
	ID int `json:"id"`
}

// --- 内部解析器 ---

func parseOCIConfig(content string) (map[string]string, error) {
	result := make(map[string]string)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" || strings.HasPrefix(line, "[") { continue }
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 { result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1]) }
	}
	return result, nil
}

// --- 处理器集合 ---

func HandleSystemInit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var count int
	_ = DB.QueryRow("SELECT COUNT(*) FROM system_config WHERE key = 'username'").Scan(&count)
	if count > 0 { w.WriteHeader(http.StatusForbidden); return }
	var req SysInitRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	_, _ = DB.Exec("INSERT OR REPLACE INTO system_config (key, value) VALUES ('username', ?), ('password', ?)", req.Username, req.Password)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

func HandleAddAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req AddAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { w.WriteHeader(http.StatusBadRequest); return }

	if req.RawConfig != "" {
		conf, _ := parseOCIConfig(req.RawConfig)
		if v, ok := conf["tenancy"]; ok { req.TenancyID = v }
		if v, ok := conf["user"]; ok { req.UserID = v }
		if v, ok := conf["fingerprint"]; ok { req.Fingerprint = v }
		if v, ok := conf["region"]; ok { req.Region = v }
	}

	if req.TenancyID == "" || req.UserID == "" || req.Fingerprint == "" || req.PrivateKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"配置基本要素不全"}`))
		return
	}

	if req.Alias == "" { req.Alias = "未命名租户" }
	if req.Proxy == "" { req.Proxy = "直连" }

	encryptedKey, _ := EncryptText(req.PrivateKey)
	query := `INSERT INTO oci_accounts (alias, tenancy_id, user_id, fingerprint, region, encrypted_key, account_type, is_multi_region, proxy, created_at, status, tenant_name) 
	          VALUES (?, ?, ?, ?, ?, ?, '个人免费账号', 0, ?, datetime('now','localtime'), 'active', '获取中...')`
	_, _ = DB.Exec(query, req.Alias, req.TenancyID, req.UserID, req.Fingerprint, req.Region, encryptedKey, req.Proxy)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

func HandleListAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	queryStr := `SELECT id, alias, tenancy_id, COALESCE(tenant_name, '获取中...'), region, fingerprint, COALESCE(account_type, '个人免费账号'), COALESCE(is_multi_region, 0), created_at, COALESCE(proxy, '直连'), COALESCE(status, 'active') FROM oci_accounts`
	rows, err := DB.Query(queryStr)
	if err != nil { w.WriteHeader(http.StatusInternalServerError); return }
	defer rows.Close()

	var list []AccountConfigResponse
	for rows.Next() {
		var acc AccountConfigResponse
		var isMulti int
		err := rows.Scan(&acc.ID, &acc.Alias, &acc.TenancyID, &acc.TenantName, &acc.Region, &acc.Fingerprint, &acc.AccountType, &isMulti, &acc.CreatedAt, &acc.Proxy, &acc.Status)
		if err == nil {
			acc.IsMultiRegion = (isMulti == 1)
			if acc.CreatedAt != "" && acc.CreatedAt != "获取中" {
				t, err := time.Parse("2006-01-02 15:04:05", acc.CreatedAt)
				if err == nil { acc.AliveDays = int(time.Since(t).Hours() / 24) }
				if acc.AliveDays <= 0 { acc.AliveDays = 1 }
			}
			list = append(list, acc)
		}
	}
	json.NewEncoder(w).Encode(list)
}

func HandleTestConnection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req TestAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { w.WriteHeader(http.StatusBadRequest); return }

	tenantName, err := TestOCILink(req.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	officialCreatedAt := "2025-07-26 14:22:03" 
	accountType := "个人免费账号"               
	isMultiRegion := 0                          

	if strings.Contains(strings.ToLower(tenantName), "payg") || strings.Contains(strings.ToLower(tenantName), "upgrade") {
		accountType = "升级版账号"
		isMultiRegion = 1
	}

	updateQuery := `UPDATE oci_accounts SET tenant_name = ?, created_at = ?, account_type = ?, is_multi_region = ? WHERE id = ?`
	_, _ = DB.Exec(updateQuery, tenantName, officialCreatedAt, accountType, isMultiRegion, req.ID)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "success",
		"tenant_name":     tenantName,
		"created_at":      officialCreatedAt,
		"account_type":    accountType,
		"is_multi_region": isMultiRegion == 1,
	})
}

func HandleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req TestAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { w.WriteHeader(http.StatusBadRequest); return }

	_, err := DB.Exec("DELETE FROM oci_accounts WHERE id = ?", req.ID)
	if err != nil { w.WriteHeader(http.StatusInternalServerError); return }
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

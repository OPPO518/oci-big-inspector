package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type AddAccountRequest struct {
	Alias       string `json:"alias"`
	TenancyID   string `json:"tenancy_id"`
	UserID      string `json:"user_id"`
	Fingerprint string `json:"fingerprint"`
	Region      string `json:"region"`
	PrivateKey  string `json:"private_key"`
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
		// 修复点：用上了 log 打印错误日志
		log.Printf("系统初始化失败: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success"}`))
}

func HandleAddAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req AddAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PrivateKey == "" {
		w.WriteHeader(http.StatusBadRequest)
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

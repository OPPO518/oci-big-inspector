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

// 定义测试连接的请求体
type TestAccountRequest struct {
	ID int `json:"id"`
}

// 处理系统首次初始化 (POST /api/system/init)
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
		_, _ = w.Write([]byte(`{"error":"系统已经初始化过，拒绝重复操作"}`))
		return
	}

	var req SysInitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"非法的账号或密码输入"}`))
		return
	}

	_, err := DB.Exec("INSERT OR REPLACE INTO system_config (key, value) VALUES ('username', ?), ('password', ?)", req.Username, req.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"初始化写入失败"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success","message":"面板初始化成功"}`))
}

// 处理添加账户 (POST /api/accounts/add)
func HandleAddAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req AddAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PrivateKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"请求参数错误"}`))
		return
	}

	encryptedKey, err := EncryptText(req.PrivateKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	query := `INSERT INTO oci_accounts (alias, tenancy_id, user_id, fingerprint, region, encrypted_key) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = DB.Exec(query, req.Alias, req.TenancyID, req.UserID, req.Fingerprint, req.Region, encryptedKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success","message":"账户添加成功"}`))
}

// 查询列表 (GET /api/accounts/list)
func HandleListAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	rows, err := DB.Query("SELECT id, alias, tenancy_id, region, fingerprint FROM oci_accounts")
	if err != nil {
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

// 【新增模块】处理测试连通性请求 (POST /api/accounts/test)
func HandleTestConnection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req TestAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"非法的请求参数"}`))
		return
	}

	// 呼叫底层的 OCI SDK 桥接模块
	tenantName, err := TestOCILink(req.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		// 将错误信息转化为 JSON 返回给前端弹窗
		json.NewEncoder(w).Encode(map[string]string{"error": "连接失败: " + err.Error()})
		return
	}

	// 如果成功，返回甲骨文那边的真实租户名
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":      "success",
		"tenant_name": tenantName,
	})
}

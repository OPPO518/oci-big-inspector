package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// AddAccountRequest 定义前端传过来的添加账户结构
type AddAccountRequest struct {
	Alias       string `json:"alias"`
	TenancyID   string `json:"tenancy_id"`
	UserID      string `json:"user_id"`
	Fingerprint string `json:"fingerprint"`
	Region      string `json:"region"`
	PrivateKey  string `json:"private_key"`
}

// SysInitRequest 定义系统首次初始化请求体
type SysInitRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AccountConfigResponse 用于安全地返回给前端账号列表（绝不暴露加密后的私钥字段）
type AccountConfigResponse struct {
	ID          int    `json:"id"`
	Alias       string `json:"alias"`
	TenancyID   string `json:"tenancy_id"`
	Region      string `json:"region"`
	Fingerprint string `json:"fingerprint"`
}

// HandleSystemInit 处理系统首次在网页上输入账号密码初始化 (POST /api/system/init)
func HandleSystemInit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 安全检查：如果数据库里已经有账号了，直接拒绝，防止被二次恶意初始化
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

	// 将管理员账号密码写入系统配置表
	_, err := DB.Exec("INSERT OR REPLACE INTO system_config (key, value) VALUES ('username', ?), ('password', ?)", req.Username, req.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"初始化写入失败"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success","message":"面板初始化成功，请刷新页面登录"}`))
}

// HandleAddAccount 处理添加账户的 API 请求 (POST /api/accounts/add)
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

	// 调用 crypto.go 高强度加密私钥
	encryptedKey, err := EncryptText(req.PrivateKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	query := `INSERT INTO oci_accounts (alias, tenancy_id, user_id, fingerprint, region, encrypted_key) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = DB.Exec(query, req.Alias, req.TenancyID, req.UserID, req.Fingerprint, req.Region, encryptedKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("❌ 数据库写入失败: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success","message":"账户添加成功"}`))
}

// HandleListAccounts 让前端在表格里渲染已绑定的甲骨文账号 (GET /api/accounts/list)
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

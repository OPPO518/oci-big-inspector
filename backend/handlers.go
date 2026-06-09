package main

import (
	"encoding/json"
	"net/http"
)

// AddAccountRequest 定义前端传过来的 JSON 数据结构
type AddAccountRequest struct {
	Alias       string `json:"alias"`
	TenancyID   string `json:"tenancy_id"`
	UserID      string `json:"user_id"`
	Fingerprint string `json:"fingerprint"`
	Region      string `json:"region"`
	PrivateKey  string `json:"private_key"` // 接收明文私钥，准备落盘加密
}

// HandleAddAccount 处理添加账户的 API 请求 (POST /api/accounts/add)
func HandleAddAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 1. 严格限制只能使用 POST 请求
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte(`{"error":"仅支持 POST 请求"}`))
		return
	}

	// 2. 解析前端传来的 JSON
	var req AddAccountRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"非法的 JSON 请求体"}`))
		return
	}

	// 3. 基础参数非空校验
	if req.Alias == "" || req.TenancyID == "" || req.UserID == "" || req.Fingerprint == "" || req.Region == "" || req.PrivateKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"所有配置项均为必填，请检查输入"}`))
		return
	}

	// 4. 【核心安全逻辑】调用 crypto.go 将明文私钥高强度加密
	encryptedKey, err := EncryptText(req.PrivateKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"系统加密模块异常"}`))
		return
	}

	// 5. 将密文及其他配置存入 SQLite 数据库 (调用 db.go 中的全局变量 DB)
	query := `INSERT INTO oci_accounts (alias, tenancy_id, user_id, fingerprint, region, encrypted_key) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = DB.Exec(query, req.Alias, req.TenancyID, req.UserID, req.Fingerprint, req.Region, encryptedKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("❌ 数据库写入失败: %v", err)
		_, _ = w.Write([]byte(`{"error":"数据库写入失败"}`))
		return
	}

	// 6. 返回成功响应
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success","message":"甲骨文 API 凭证已安全加密归档"}`))
}

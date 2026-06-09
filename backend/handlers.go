package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// --- 数据模型 ---

type AddAccountRequest struct {
	Alias          string `json:"alias"`
	TenancyID      string `json:"tenancy_id"`
	UserID         string `json:"user_id"`
	Fingerprint    string `json:"fingerprint"`
	Region         string `json:"region"`
	PrivateKey     string `json:"private_key"`
	RawConfig      string `json:"raw_config"`       // 用于快速粘贴解析
	AccountType    string `json:"account_type"`     // 账号类型：免费版/升级版
	IsMultiRegion  bool   `json:"is_multi_region"`  // 是否多区
	Proxy          string `json:"proxy"`            // 专属代理
}

type SysInitRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AccountConfigResponse struct {
	ID             int    `json:"id"`
	Alias          string `json:"alias"`
	TenancyID      string `json:"tenancy_id"`      // OCID
	TenantName     string `json:"tenant_name"`     // 物理租户名（例如：compta91）
	Region         string `json:"region"`
	Fingerprint    string `json:"fingerprint"`
	AccountType    string `json:"account_type"`
	IsMultiRegion  bool   `json:"is_multi_region"`
	CreatedAt      string `json:"created_at"`
	AliveDays      int    `json:"alive_days"`
	HasBootTask    bool   `json:"has_boot_task"`
	Status         string `json:"status"`
	Proxy          string `json:"proxy"`
}

type TestAccountRequest struct {
	ID int `json:"id"`
}

// --- 辅助函数 ---

// parseOCIConfig 解析 OCI 原始配置文段
func parseOCIConfig(content string) (map[string]string, error) {
	result := make(map[string]string)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" || strings.HasPrefix(line, "[") {
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

func HandleSystemInit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost { w.WriteHeader(http.StatusMethodNotAllowed); return }
	var count int
	_ = DB.QueryRow("SELECT COUNT(*) FROM system_config WHERE key = 'username'").Scan(&count)
	if count > 0 {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"系统已初始化"}`))
		return
	}
	var req SysInitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, _ = DB.Exec("INSERT OR REPLACE INTO system_config (key, value) VALUES ('username', ?), ('password', ?)", req.Username, req.Password)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success"}`))
}

func HandleAddAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req AddAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.RawConfig != "" {
		conf, _ := parseOCIConfig(req.RawConfig)
		if v, ok := conf["tenancy"]; ok { req.TenancyID = v }
		if v, ok := conf["user"]; ok { req.UserID = v }
		if v, ok := conf["fingerprint"]; ok { req.Fingerprint = v }
		if v, ok := conf["region"]; ok { req.Region = v }
	}

	if req.TenancyID == "" || req.UserID == "" || req.Fingerprint == "" || req.PrivateKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"参数不完整，请检查配置内容"}`))
		return
	}

	if req.Alias == "" { req.Alias = "DEFAULT" }
	if req.AccountType == "" { req.AccountType = "个人免费账号" }
	if req.Proxy == "" { req.Proxy = "直连" }

	encryptedKey, err := EncryptText(req.PrivateKey)
	if err != nil { w.WriteHeader(http.StatusInternalServerError); return }

	// 动态插入扩展字段
	query := `INSERT INTO oci_accounts (alias, tenancy_id, user_id, fingerprint, region, encrypted_key, account_type, is_multi_region, proxy, created_at, status) 
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now','localtime'), 'active')`
	_, err = DB.Exec(query, req.Alias, req.TenancyID, req.UserID, req.Fingerprint, req.Region, encryptedKey, req.AccountType, req.IsMultiRegion, req.Proxy)
	if err != nil {
		log.Printf("数据库写入错误: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success"}`))
}

func HandleListAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// 注意：此处若数据库表尚未执行 ALTER ADD COLUMN，请确保底层表结构已更新
	rows, err := DB.Query("SELECT id, alias, tenancy_id, region, fingerprint, account_type, is_multi_region, created_at, proxy, status FROM oci_accounts")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var list []AccountConfigResponse
	for rows.Next() {
		var acc AccountConfigResponse
		var isMulti int
		err := rows.Scan(&acc.ID, &acc.Alias, &acc.TenancyID, &acc.Region, &acc.Fingerprint, &acc.AccountType, &isMulti, &acc.CreatedAt, &acc.Proxy, &acc.Status)
		if err == nil {
			acc.IsMultiRegion = (isMulti == 1)
			
			// 计算存活天数逻辑
			if acc.CreatedAt != "" {
				t, err := time.Parse("2006-01-02 15:04:05", acc.CreatedAt)
				if err == nil {
					acc.AliveDays = int(time.Since(t).Hours() / 24)
				}
			}
			if acc.AliveDays <= 0 { acc.AliveDays = 1 } // 默认初始 1 天
			
			// 核心修正：tenant_name 理论上需要通过 TestOCILink(acc.ID) 在后台查询并缓存，这里暂用别名或处理后的字符串占位，待测通后下发
			acc.TenantName = "获取中..." 
			
			list = append(list, acc)
		}
	}
	json.NewEncoder(w).Encode(list)
}

func HandleTestConnection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req TestAccountRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	tenantName, err := TestOCILink(req.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// 测通后同步把真实租户名更新到表中
	_, _ = DB.Exec("UPDATE oci_accounts SET tenant_name = ? WHERE id = ?", tenantName, req.ID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "tenant_name": tenantName})
}

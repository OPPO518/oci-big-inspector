package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/glebarez/go-sqlite" // 注册纯 Go 版本的 sqlite 驱动
)

var DB *sql.DB

// InitializeDB 初始化 SQLite 数据库及数据表
func InitializeDB() {
	dbDir := "/app/data"
	// 如果是本地调试发现没有这个目录，自动降级创建当前目录下的 data
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		dbDir = "./data"
	}
	_ = os.MkdirAll(dbDir, 0755)

	dbPath := filepath.Join(dbDir, "inspector.db")
	log.Printf("📦 [数据库] 正在连接/创建本地 SQLite 数据库: %s", dbPath)

	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("❌ 数据库打开失败: %v", err)
	}

	// 创建账户凭证表 (存储加密后的私钥)
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS oci_accounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		alias TEXT NOT NULL,              -- 账号别名 (如: 东平主号)
		tenancy_id TEXT NOT NULL,         -- 租户 OCID
		user_id TEXT NOT NULL,            -- 用户 OCID
		fingerprint TEXT NOT NULL,        -- API 密钥指纹
		region TEXT NOT NULL,             -- 默认主区域
		encrypted_key TEXT NOT NULL,      -- AES 加密后的 Private Key 全文
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = DB.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("❌ 数据表创建失败: %v", err)
	}
	log.Println("✅ [数据库] 密文数据表初始化/检查完成。")
}

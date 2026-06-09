package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/glebarez/go-sqlite"
)

var DB *sql.DB

func InitializeDB() {
	dbDir := "/app/data"
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

	// 1. 创建系统配置表（存放 Web 界面初始化的账号密码）
	createConfigTable := `
	CREATE TABLE IF NOT EXISTS system_config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`
	_, _ = DB.Exec(createConfigTable)

	// 2. 创建账户凭证表保持不变
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS oci_accounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		alias TEXT NOT NULL,
		tenancy_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		fingerprint TEXT NOT NULL,
		region TEXT NOT NULL,
		encrypted_key TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = DB.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("❌ 数据表创建失败: %v", err)
	}
	log.Println("✅ [数据库] 所有数据表初始化检查完成。")
}

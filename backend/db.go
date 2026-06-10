package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

// InitializeDB 负责在系统初载时，创建完美对齐的工业级数据库表结构
func InitializeDB() {
	dbDir := "/app/data"
	_ = os.MkdirAll(dbDir, 0755)
	dbPath := filepath.Join(dbDir, "inspector.db")

	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("❌ 无法打开 SQLite 数据库: %v", err)
	}

	// 1. 创建系统基础配置表
	configTableQuery := `
	CREATE TABLE IF NOT EXISTS system_config (
		key TEXT PRIMARY KEY,
		value TEXT
	);`
	_, err = DB.Exec(configTableQuery)
	if err != nil {
		log.Fatalf("❌ 创建 system_config 表失败: %v", err)
	}

	// 2. 🚀【核心修正】：创建满血完全体的 OCI 租户凭证表，把所有缺少的列全部锁死进初始化中
	accountTableQuery := `
	CREATE TABLE IF NOT EXISTS oci_accounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		alias TEXT,
		tenancy_id TEXT,
		user_id TEXT,
		fingerprint TEXT,
		region TEXT,
		encrypted_key TEXT,
		account_type TEXT DEFAULT '个人免费账号',
		is_multi_region INTEGER DEFAULT 0,
		proxy TEXT DEFAULT '直连',
		created_at TEXT,
		status TEXT DEFAULT 'active',
		tenant_name TEXT DEFAULT '获取中...'
	);`
	_, err = DB.Exec(accountTableQuery)
	if err != nil {
		log.Fatalf("❌ 创建 oci_accounts 表失败: %v", err)
	}
}

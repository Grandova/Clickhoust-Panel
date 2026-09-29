package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"clickhouse-manager/internal/api"
	"clickhouse-manager/internal/auth"
	"clickhouse-manager/internal/clickhouse"
	"clickhouse-manager/internal/database"
	"clickhouse-manager/web"
)

func main() {
	port := flag.Int("port", 8080, "Web panel server port")
	dbPath := flag.String("db", "data/clickhouse-manager.db", "SQLite database path")
	chHost := flag.String("ch-host", "127.0.0.1", "ClickHouse host")
	chPort := flag.Int("ch-port", 9000, "ClickHouse native TCP port")
	chUser := flag.String("ch-user", "default", "ClickHouse user")
	chPassword := flag.String("ch-password", "", "ClickHouse password")
	flag.Parse()

	// Environment variable overrides
	if envPort := os.Getenv("PORT"); envPort != "" {
		_, _ = fmt.Sscanf(envPort, "%d", port)
	}

	log.Printf("[Init] Starting ClickHouse Manager...")

	// 1. Initialize SQLite
	if err := database.InitDB(*dbPath); err != nil {
		log.Fatalf("[Database] Failed to initialize SQLite database at %s: %v", *dbPath, err)
	}
	log.Printf("[Database] SQLite database initialized at %s", *dbPath)
	if err := auth.InitJWTSecret(filepath.Join(filepath.Dir(*dbPath), "jwt-secret")); err != nil {
		log.Fatalf("[Auth] Failed to initialize JWT secret: %v", err)
	}

	// 2. Initialize ClickHouse client configuration (fallback to persisted DB setting)
	appliedCfg := clickhouse.ConnectionConfig{
		Host:     *chHost,
		Port:     *chPort,
		User:     *chUser,
		Password: *chPassword,
		Database: "default",
		Protocol: "native",
	}

	var setting database.PanelSetting
	if database.DB != nil && database.DB.Where("key = ?", "ch_connection").First(&setting).Error == nil {
		var persistedCfg clickhouse.ConnectionConfig
		if err := json.Unmarshal([]byte(setting.Value), &persistedCfg); err == nil && persistedCfg.Host != "" {
			appliedCfg = persistedCfg
			log.Printf("[ClickHouse] Loaded persisted connection configuration: %s:%d (user=%s)", appliedCfg.Host, appliedCfg.Port, appliedCfg.User)
		}
	}
	clickhouse.DefaultService.SetConfig(appliedCfg)

	// 3. Setup Gin router
	router := api.SetupRouter()
	web.RegisterStaticRoutes(router)

	banner := fmt.Sprintf(`
============================================================
   ______ _ _      _    _    _                         
  / ____/| (_)____| |  | |  | |                        
 | |     | |_  ___| | _| |__| | ___  _   _ ___  ___    
 | |     | | |/ __| |/ /  __  |/ _ \| | | / __|/ _ \   
 | |____ | | | (__|   <| |  | | (_) | |_| \__ \  __/   
  \_____||_|_|\___|_|\_\_|  |_|\___/ \__,_|___/\___|   
                                                       
                ClickHouse Manager v1.2.0
============================================================
  Web Panel URL:     http://0.0.0.0:%d
  ClickHouse Target: %s:%d
  Panel Database:    %s
  Default Username:  admin
  Default Password:  admin123456 (Must change on first login)
============================================================
`, *port, *chHost, *chPort, *dbPath)

	fmt.Println(banner)

	addr := fmt.Sprintf(":%d", *port)
	if err := router.Run(addr); err != nil {
		log.Fatalf("[Server] Failed to run server on %s: %v", addr, err)
	}
}

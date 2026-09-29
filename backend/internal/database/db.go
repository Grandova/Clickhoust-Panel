package database

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

type AdminUser struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	Username           string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash       string    `gorm:"not null" json:"-"`
	Role               string    `gorm:"size:32;default:'admin'" json:"role"`
	MustChangePassword bool      `gorm:"default:true" json:"must_change_password"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"size:64;index" json:"username"`
	IP        string    `gorm:"size:64" json:"ip"`
	Action    string    `gorm:"size:64;index" json:"action"`
	Target    string    `gorm:"size:255" json:"target"`
	Result    string    `gorm:"size:32" json:"result"` // SUCCESS / FAILED
	Error     string    `gorm:"type:text" json:"error,omitempty"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

type ConfigBackup struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	FilePath  string    `gorm:"size:255;index" json:"file_path"`
	Content   string    `gorm:"type:text" json:"content"`
	Reason    string    `gorm:"size:255" json:"reason"`
	CreatedBy string    `gorm:"size:64" json:"created_by"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

type QueryFavorite struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"size:128" json:"title"`
	SQLText   string    `gorm:"type:text" json:"sql_text"`
	Database  string    `gorm:"size:64" json:"database"`
	CreatedBy string    `gorm:"size:64" json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type BackupRecord struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	BackupID     string     `gorm:"size:128;uniqueIndex" json:"backup_id"`
	DatabaseName string     `gorm:"size:64" json:"database_name"`
	TableName    string     `gorm:"size:64" json:"table_name"`
	Disk         string     `gorm:"size:64;default:'default'" json:"disk"`
	Size         int64      `json:"size"`
	Status       string     `gorm:"size:32" json:"status"` // RUNNING, SUCCESS, FAILED
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Error        string     `gorm:"type:text" json:"error,omitempty"`
}

type PanelSetting struct {
	Key       string    `gorm:"primaryKey;size:64" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

func InitDB(dbPath string) error {
	if dbPath == "" {
		dbPath = "data/clickhouse-manager.db"
	}

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(1)
		_, _ = sqlDB.Exec("PRAGMA journal_mode = WAL;")
		_, _ = sqlDB.Exec("PRAGMA busy_timeout = 5000;")
		_, _ = sqlDB.Exec("PRAGMA synchronous = NORMAL;")
	}

	if err := db.AutoMigrate(
		&AdminUser{},
		&AuditLog{},
		&ConfigBackup{},
		&QueryFavorite{},
		&BackupRecord{},
		&PanelSetting{},
	); err != nil {
		return err
	}

	DB = db

	// Ensure default admin user exists
	var count int64
	db.Model(&AdminUser{}).Count(&count)
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin123456"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		defaultAdmin := AdminUser{
			Username:           "admin",
			PasswordHash:       string(hash),
			Role:               "admin",
			MustChangePassword: true,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}
		if err := db.Create(&defaultAdmin).Error; err != nil {
			return err
		}
		log.Println("[Database] Initialized default admin user: admin (password: admin123456, first login requires change)")
	}

	return nil
}

package clickhouse

import (
	"context"
	"fmt"
	"time"

	"clickhouse-manager/internal/database"
)

type BackupRequest struct {
	Database string `json:"database"`
	Table    string `json:"table"` // empty for all tables in database
	Disk     string `json:"disk"`  // default / backups
	BackupID string `json:"backup_id"`
}

type RestoreRequest struct {
	BackupID string `json:"backup_id"`
	Disk     string `json:"disk"`
}

func (s *ClickHouseService) CreateBackup(ctx context.Context, req BackupRequest) (string, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return "", err
	}

	backupID := req.BackupID
	if backupID == "" {
		backupID = fmt.Sprintf("backup_%s_%s", req.Database, time.Now().Format("20060102_150405"))
	}

	disk := req.Disk
	if disk == "" {
		disk = "default"
	}

	dest := fmt.Sprintf("Disk('%s', '%s')", escapeString(disk), escapeString(backupID))

	var backupSQL string
	if req.Table != "" {
		backupSQL = fmt.Sprintf("BACKUP TABLE `%s`.`%s` TO %s", escapeIdent(req.Database), escapeIdent(req.Table), dest)
	} else if req.Database != "" {
		backupSQL = fmt.Sprintf("BACKUP DATABASE `%s` TO %s", escapeIdent(req.Database), dest)
	} else {
		backupSQL = fmt.Sprintf("BACKUP ALL TO %s", dest)
	}

	// Record in panel database
	record := database.BackupRecord{
		BackupID:     backupID,
		DatabaseName: req.Database,
		TableName:    req.Table,
		Disk:         disk,
		Status:       "RUNNING",
		StartedAt:    time.Now(),
	}
	if database.DB != nil {
		database.DB.Create(&record)
	}

	// Execute backup in background or with timeout
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		execErr := conn.Exec(bgCtx, backupSQL)
		now := time.Now()
		if database.DB != nil {
			if execErr != nil {
				database.DB.Model(&database.BackupRecord{}).Where("backup_id = ?", backupID).Updates(map[string]interface{}{
					"status":       "FAILED",
					"completed_at": &now,
					"error":        execErr.Error(),
				})
			} else {
				database.DB.Model(&database.BackupRecord{}).Where("backup_id = ?", backupID).Updates(map[string]interface{}{
					"status":       "SUCCESS",
					"completed_at": &now,
				})
			}
		}
	}()

	return backupID, nil
}

func (s *ClickHouseService) RestoreBackup(ctx context.Context, req RestoreRequest) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}

	disk := req.Disk
	if disk == "" {
		disk = "default"
	}

	src := fmt.Sprintf("Disk('%s', '%s')", escapeString(disk), escapeString(req.BackupID))
	restoreSQL := fmt.Sprintf("RESTORE ALL FROM %s", src)

	return conn.Exec(ctx, restoreSQL)
}

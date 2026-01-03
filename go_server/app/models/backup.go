package models

import (
	"time"
)

type BackupStatus string

const (
	BackupInProgress BackupStatus = "IN_PROGRESS"
	BackupCanceled   BackupStatus = "CANCELED"
	BackupFailed     BackupStatus = "FAILED"
	BackupDone       BackupStatus = "DONE"
)

// BackupSession tracks an ongoing backup process
type BackupSession struct {
	ID             uint         `gorm:"primaryKey" json:"id"`
	TransferID     string       `gorm:"uniqueIndex;size:64" json:"transfer_id"`
	DeviceID       string       `gorm:"index;size:64" json:"device_id"`
	FileUUID       string       `gorm:"index;size:64" json:"file_uuid"`
	FileName       string       `json:"file_name"`
	Version        int          `json:"version"`
	CurrentOffset  int64        `json:"current_offset"`
	TotalSize      int64        `json:"total_size"`
	Status         BackupStatus `gorm:"size:20" json:"status"`
	LastUpdateTime time.Time    `json:"last_update_time"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

// BackupSnapshot stores info about a completed backup version
type BackupSnapshot struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	DeviceID   string    `gorm:"index;size:64" json:"device_id"`
	FileUUID   string    `gorm:"index;size:64" json:"file_uuid"`
	Version    int       `json:"version"`
	ServerPath string    `json:"server_path"` // Path on server storage
	FileSize   int64     `json:"file_size"`
	FileHash   string    `gorm:"size:64" json:"file_hash"` // SHA256
	CreatedAt  time.Time `json:"created_at"`
}

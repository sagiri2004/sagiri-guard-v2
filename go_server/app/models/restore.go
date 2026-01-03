package models

import (
	"time"
)

type RestoreStatus string

const (
	RestoreInProgress RestoreStatus = "IN_PROGRESS"
	RestoreDone       RestoreStatus = "DONE"
	RestoreFailed     RestoreStatus = "FAILED"
)

type RestoreSession struct {
	ID         uint          `gorm:"primaryKey" json:"id"`
	TransferID string        `gorm:"uniqueIndex;size:64" json:"transfer_id"`
	DeviceID   string        `gorm:"index;size:64" json:"device_id"`
	FileUUID   string        `gorm:"index;size:64" json:"file_uuid"`
	FileName   string        `json:"file_name"`
	Version    int           `json:"version"`
	ServerPath string        `json:"server_path"`
	TotalSize  int64         `json:"total_size"`
	FileHash   string        `gorm:"size:64" json:"file_hash"`
	Status     RestoreStatus `gorm:"size:20" json:"status"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

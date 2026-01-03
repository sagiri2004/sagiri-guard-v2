package db

import (
	"time"
)

// MonitoredFile tracks the current state of a file or folder
type MonitoredFile struct {
	UUID         string    `gorm:"primaryKey" json:"uuid"` // user.sagiri_id for files, deterministic for folders
	CurrentPath  string    `gorm:"index" json:"current_path"`
	ItemType     string    `json:"item_type"` // "file" or "folder"
	LastAction   string    `json:"last_action"`
	LastEventAt  time.Time `json:"last_event_at"`
	LastBackupAt time.Time `json:"last_backup_at"`
}

// FileChangeEvent tracks the history of file events
type FileChangeEvent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	FileUUID  string    `gorm:"index" json:"file_uuid"`
	ItemType  string    `json:"item_type"` // "file" or "folder"
	Action    string    `json:"action"`    // CREATE, MODIFY, RENAME, DELETE
	FromPath  string    `json:"from_path"`
	ToPath    string    `json:"to_path"`
	Timestamp time.Time `gorm:"index" json:"timestamp"`
}

// LocalRestoreSession tracks an ongoing restore process on the client
type LocalRestoreSession struct {
	TransferID    string    `gorm:"primaryKey" json:"transfer_id"`
	FileUUID      string    `gorm:"index" json:"file_uuid"`
	Version       int       `json:"version"`
	LocalPath     string    `json:"local_path"`
	CurrentOffset int64     `json:"current_offset"`
	TotalSize     int64     `json:"total_size"`
	FileHash      string    `json:"file_hash"`
	Status        string    `json:"status"` // IN_PROGRESS, DONE, FAILED
	UpdatedAt     time.Time `json:"updated_at"`
}

package models

import (
	"time"

	"gorm.io/gorm"
)

type DeviceFileHistory struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	DeviceID  string         `gorm:"index" json:"device_id"`
	FileUUID  string         `gorm:"index" json:"file_uuid"`
	Action    string         `json:"action"` // create, modify, delete, rename, move_out
	Path      string         `json:"path"`
	OldPath   string         `json:"old_path"` // For rename/move
	EventTime time.Time      `json:"event_time"`
	SyncedAt  time.Time      `json:"synced_at"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

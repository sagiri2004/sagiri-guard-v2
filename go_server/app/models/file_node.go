package models

import (
	"time"

	"gorm.io/gorm"
)

type FileNode struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	DeviceID  string         `gorm:"index;size:64" json:"device_id"`
	ParentID  *uint          `gorm:"index" json:"parent_id"` // Recursive relationship
	UUID      string         `gorm:"uniqueIndex;size:64" json:"uuid"`
	Name      string         `gorm:"size:255" json:"name"`
	Path      string         `gorm:"size:1024" json:"path"`
	Type      string         `gorm:"size:10" json:"type"` // "file" or "folder"
	IsDeleted bool           `gorm:"default:false" json:"is_deleted"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`

	// Relations
	Children []FileNode `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

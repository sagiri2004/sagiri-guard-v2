package models

import (
	"time"
)

type FirewallCategory struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"unique;not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FirewallDomain struct {
	ID         uint             `gorm:"primaryKey" json:"id"`
	CategoryID uint             `gorm:"not null;index" json:"category_id"`
	Domain     string           `gorm:"not null" json:"domain"`
	Category   FirewallCategory `gorm:"foreignKey:CategoryID" json:"-"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

type DeviceConfig struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	DeviceID           string    `gorm:"unique;not null" json:"device_id"`
	FirewallEnabled    bool      `gorm:"default:false" json:"firewall_enabled"`
	SelectedCategories string    `gorm:"type:text" json:"selected_categories"` // JSON array of Category IDs e.g. "[1, 2]"
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

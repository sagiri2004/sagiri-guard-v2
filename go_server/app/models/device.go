package models

import "gorm.io/gorm"

type Device struct {
	gorm.Model
	UserID    uint   `gorm:"not null"`
	DeviceID  string `gorm:"size:255;uniqueIndex;not null"` // UUID from client
	Name      string
	OSName    string
	OSVersion string
	Hostname  string
	Arch      string
}

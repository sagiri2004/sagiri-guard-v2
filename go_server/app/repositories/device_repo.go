package repositories

import (
	"demo/network/go_server/app/models"

	"gorm.io/gorm"
)

type DeviceRepository struct {
	DB *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{DB: db}
}

func (r *DeviceRepository) GetByDeviceID(deviceID string) (*models.DeviceConfig, error) {
	var config models.DeviceConfig
	result := r.DB.Where("device_id = ?", deviceID).First(&config)
	if result.Error != nil {
		return nil, result.Error
	}
	return &config, nil
}

func (r *DeviceRepository) Upsert(config *models.DeviceConfig) error {
	return r.DB.Save(config).Error
}

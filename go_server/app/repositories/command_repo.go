package repositories

import (
	"demo/network/go_server/app/models"

	"gorm.io/gorm"
)

type CommandRepository struct {
	DB *gorm.DB
}

func NewCommandRepository(db *gorm.DB) *CommandRepository {
	return &CommandRepository{DB: db}
}

func (r *CommandRepository) Create(cmd *models.Command) error {
	return r.DB.Create(cmd).Error
}

func (r *CommandRepository) Update(cmd *models.Command) error {
	return r.DB.Save(cmd).Error
}

func (r *CommandRepository) GetPendingByDevice(deviceID string) ([]models.Command, error) {
	var commands []models.Command
	result := r.DB.Where("device_id = ? AND status = ?", deviceID, models.StatusPending).Find(&commands)
	return commands, result.Error
}

func (r *CommandRepository) GetHistory(deviceID string, limit, offset int) ([]models.Command, error) {
	var commands []models.Command
	query := r.DB.Model(&models.Command{}).Order("created_at desc")
	if deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}
	result := query.Limit(limit).Offset(offset).Find(&commands)
	return commands, result.Error
}

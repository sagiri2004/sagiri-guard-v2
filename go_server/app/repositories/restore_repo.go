package repositories

import (
	"demo/network/go_server/app/models"

	"gorm.io/gorm"
)

type RestoreRepository struct {
	db *gorm.DB
}

func NewRestoreRepository(db *gorm.DB) *RestoreRepository {
	return &RestoreRepository{db: db}
}

func (r *RestoreRepository) CreateSession(session *models.RestoreSession) error {
	return r.db.Create(session).Error
}

func (r *RestoreRepository) GetSessionByTransferID(transferID string) (*models.RestoreSession, error) {
	var session models.RestoreSession
	err := r.db.Where("transfer_id = ?", transferID).First(&session).Error
	return &session, err
}

func (r *RestoreRepository) UpdateSession(session *models.RestoreSession) error {
	return r.db.Save(session).Error
}

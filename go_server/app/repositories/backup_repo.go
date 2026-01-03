package repositories

import (
	"demo/network/go_server/app/models"
	"time"

	"gorm.io/gorm"
)

type BackupRepository struct {
	db *gorm.DB
}

func NewBackupRepository(db *gorm.DB) *BackupRepository {
	return &BackupRepository{db: db}
}

func (r *BackupRepository) CreateSession(session *models.BackupSession) error {
	return r.db.Create(session).Error
}

func (r *BackupRepository) GetSessionByTransferID(transferID string) (*models.BackupSession, error) {
	var session models.BackupSession
	err := r.db.Where("transfer_id = ?", transferID).First(&session).Error
	return &session, err
}

func (r *BackupRepository) UpdateSession(session *models.BackupSession) error {
	session.LastUpdateTime = time.Now()
	return r.db.Save(session).Error
}

func (r *BackupRepository) GetLatestVersion(deviceID, fileUUID string) (int, error) {
	var snapshot models.BackupSnapshot
	err := r.db.Where("device_id = ? AND file_uuid = ?", deviceID, fileUUID).
		Order("version desc").First(&snapshot).Error
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	return snapshot.Version, err
}

func (r *BackupRepository) CreateSnapshot(snapshot *models.BackupSnapshot) error {
	return r.db.Create(snapshot).Error
}

func (r *BackupRepository) GetSnapshots(deviceID, fileUUID string) ([]models.BackupSnapshot, error) {
	var snapshots []models.BackupSnapshot
	err := r.db.Where("device_id = ? AND file_uuid = ?", deviceID, fileUUID).
		Order("version desc").Find(&snapshots).Error
	return snapshots, err
}

func (r *BackupRepository) GetActiveSession(deviceID, fileUUID string) (*models.BackupSession, error) {
	var session models.BackupSession
	err := r.db.Where("device_id = ? AND file_uuid = ? AND status = ?", deviceID, fileUUID, models.BackupInProgress).
		Order("created_at desc").First(&session).Error
	return &session, err
}

func (r *BackupRepository) GetLatestSnapshot(deviceID, fileUUID string, snapshot *models.BackupSnapshot) error {
	return r.db.Where("device_id = ? AND file_uuid = ?", deviceID, fileUUID).
		Order("version desc").First(snapshot).Error
}

func (r *BackupRepository) GetSnapshotByVersion(deviceID, fileUUID string, version int, snapshot *models.BackupSnapshot) error {
	return r.db.Where("device_id = ? AND file_uuid = ? AND version = ?", deviceID, fileUUID, version).
		First(snapshot).Error
}

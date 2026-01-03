package services

import (
	"demo/network/go_server/app/models"
	"demo/network/go_server/app/repositories"
	"errors"
	"io"
	"os"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RestoreService struct {
	repo         *repositories.BackupRepository // Reuse BackupRepository for snapshots
	restoreRepo  *repositories.RestoreRepository
	fileNodeRepo *repositories.FileNodeRepository
}

func NewRestoreService(repo *repositories.BackupRepository, restoreRepo *repositories.RestoreRepository, fileNodeRepo *repositories.FileNodeRepository) *RestoreService {
	return &RestoreService{
		repo:         repo,
		restoreRepo:  restoreRepo,
		fileNodeRepo: fileNodeRepo,
	}
}

func (s *RestoreService) InitSession(deviceID, fileUUID string, version int) (*models.RestoreSession, error) {
	// 1. Find the snapshot
	var snapshot models.BackupSnapshot
	var err error

	if version == 0 {
		// Latest
		err = s.repo.GetLatestSnapshot(deviceID, fileUUID, &snapshot)
	} else {
		err = s.repo.GetSnapshotByVersion(deviceID, fileUUID, version, &snapshot)
	}

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("snapshot not found")
		}
		return nil, err
	}

	// 2. Get FileName
	fileName := "restored_file"
	if node, err := s.fileNodeRepo.FindByUUID(fileUUID); err == nil {
		fileName = node.Name
	}

	// 3. Create Restore Session
	session := &models.RestoreSession{
		TransferID: uuid.New().String(),
		DeviceID:   deviceID,
		FileUUID:   fileUUID,
		FileName:   fileName,
		Version:    snapshot.Version,
		ServerPath: snapshot.ServerPath,
		TotalSize:  snapshot.FileSize,
		FileHash:   snapshot.FileHash,
		Status:     models.RestoreInProgress,
	}

	if err := s.restoreRepo.CreateSession(session); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *RestoreService) GetChunk(transferID string, offset int64, size int) ([]byte, error) {
	session, err := s.restoreRepo.GetSessionByTransferID(transferID)
	if err != nil {
		return nil, err
	}

	if session.Status != models.RestoreInProgress {
		return nil, errors.New("session not in progress")
	}

	file, err := os.Open(session.ServerPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buffer := make([]byte, size)
	n, err := file.ReadAt(buffer, offset)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return buffer[:n], nil
}

func (s *RestoreService) FinishSession(transferID string) error {
	session, err := s.restoreRepo.GetSessionByTransferID(transferID)
	if err != nil {
		return err
	}

	session.Status = models.RestoreDone
	return s.restoreRepo.UpdateSession(session)
}

func (s *RestoreService) ResumeSession(transferID string) (*models.RestoreSession, error) {
	session, err := s.restoreRepo.GetSessionByTransferID(transferID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("session not found")
		}
		return nil, err
	}

	if session.Status != models.RestoreInProgress {
		return nil, errors.New("session not in progress")
	}

	return session, nil
}

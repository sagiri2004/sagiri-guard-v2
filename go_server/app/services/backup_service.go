package services

import (
	"demo/network/go_server/app/models"
	"demo/network/go_server/app/repositories"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type BackupService struct {
	repo *repositories.BackupRepository
}

func NewBackupService(repo *repositories.BackupRepository) *BackupService {
	return &BackupService{repo: repo}
}

func (s *BackupService) InitSession(deviceID, fileUUID, fileName string, totalSize int64) (*models.BackupSession, error) {
	// 1. Determine next version
	currentVersion, err := s.repo.GetLatestVersion(deviceID, fileUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest version: %v", err)
	}
	nextVersion := currentVersion + 1

	// 2. Create session
	session := &models.BackupSession{
		TransferID:     uuid.New().String(),
		DeviceID:       deviceID,
		FileUUID:       fileUUID,
		FileName:       fileName,
		Version:        nextVersion,
		CurrentOffset:  0,
		TotalSize:      totalSize,
		Status:         models.BackupInProgress,
		LastUpdateTime: time.Now(),
	}

	if err := s.repo.CreateSession(session); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *BackupService) UpdateChunk(transferID string, offset int64, dataLen int64) error {
	session, err := s.repo.GetSessionByTransferID(transferID)
	if err != nil {
		return err
	}

	if session.Status != models.BackupInProgress {
		return fmt.Errorf("session is not in progress: %s", session.Status)
	}

	// Update offset
	session.CurrentOffset = offset + dataLen
	return s.repo.UpdateSession(session)
}

func (s *BackupService) FinishSession(transferID, serverPath, fileHash string) error {
	session, err := s.repo.GetSessionByTransferID(transferID)
	if err != nil {
		return err
	}

	if session.Status != models.BackupInProgress {
		return fmt.Errorf("session is not in progress: %s", session.Status)
	}

	// 1. Create Snapshot
	snapshot := &models.BackupSnapshot{
		DeviceID:   session.DeviceID,
		FileUUID:   session.FileUUID,
		Version:    session.Version,
		ServerPath: serverPath,
		FileSize:   session.TotalSize,
		FileHash:   fileHash,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.CreateSnapshot(snapshot); err != nil {
		return err
	}

	// 2. Mark Session as DONE
	session.Status = models.BackupDone
	return s.repo.UpdateSession(session)
}

func (s *BackupService) CancelSession(transferID string) error {
	session, err := s.repo.GetSessionByTransferID(transferID)
	if err != nil {
		return err
	}

	session.Status = models.BackupCanceled
	return s.repo.UpdateSession(session)
}

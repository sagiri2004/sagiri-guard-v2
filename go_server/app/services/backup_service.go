package services

import (
	"demo/network/go_server/app/models"
	"demo/network/go_server/app/repositories"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type BackupService struct {
	repo        *repositories.BackupRepository
	storagePath string
}

func NewBackupService(repo *repositories.BackupRepository, storagePath string) *BackupService {
	return &BackupService{repo: repo, storagePath: storagePath}
}

func (s *BackupService) InitSession(deviceID, fileUUID, fileName string, totalSize int64, headHash string) (*models.BackupSession, error) {
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
		FileHeadHash:   headHash,
		Status:         models.BackupInProgress,
		LastUpdateTime: time.Now(),
	}

	if err := s.repo.CreateSession(session); err != nil {
		return nil, err
	}

	// 3. Ensure Storage Directory Exists
	dir := filepath.Join(s.storagePath, deviceID, fileUUID, fmt.Sprintf("v%d", nextVersion))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %v", err)
	}

	return session, nil
}

func (s *BackupService) UpdateChunk(transferID string, offset int64, dataLen int64, hexData string) error {
	session, err := s.repo.GetSessionByTransferID(transferID)
	if err != nil {
		return err
	}

	if session.Status != models.BackupInProgress {
		return fmt.Errorf("session is not in progress: %s", session.Status)
	}

	// 1. Decode Data
	data, err := hex.DecodeString(hexData)
	if err != nil {
		return fmt.Errorf("failed to decode chunk data: %v", err)
	}

	if int64(len(data)) != dataLen {
		return fmt.Errorf("data length mismatch: expected %d, got %d", dataLen, len(data))
	}

	// 2. Write to File
	path := filepath.Join(s.storagePath, session.DeviceID, session.FileUUID, fmt.Sprintf("v%d", session.Version), session.FileName)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file for writing: %v", err)
	}
	defer f.Close()

	if _, err := f.WriteAt(data, offset); err != nil {
		return fmt.Errorf("failed to write data chunk: %v", err)
	}

	// 3. Update offset in DB (optional since we have offset in request, but good for progress)
	if offset+dataLen > session.CurrentOffset {
		session.CurrentOffset = offset + dataLen
		return s.repo.UpdateSession(session)
	}
	return nil
}

func (s *BackupService) FinishSession(transferID, serverPath, fileHash string) error {
	session, err := s.repo.GetSessionByTransferID(transferID)
	if err != nil {
		return err
	}

	if session.Status != models.BackupInProgress {
		return fmt.Errorf("session is not in progress: %s", session.Status)
	}

	// 1. Generate serverPath if not provided by client
	// Format: storagePath/deviceID/fileUUID/version/fileName (or just the directory)
	finalPath := serverPath
	if finalPath == "" {
		finalPath = filepath.Join(s.storagePath, session.DeviceID, session.FileUUID, fmt.Sprintf("v%d", session.Version), session.FileName)
	}

	// 2. Create Snapshot
	snapshot := &models.BackupSnapshot{
		DeviceID:   session.DeviceID,
		FileUUID:   session.FileUUID,
		Version:    session.Version,
		ServerPath: finalPath,
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

func (s *BackupService) GetActiveSession(deviceID, fileUUID string) (*models.BackupSession, error) {
	return s.repo.GetActiveSession(deviceID, fileUUID)
}

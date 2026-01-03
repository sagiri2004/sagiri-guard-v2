package services

import (
	"demo/network/go_server/app/models"
	"demo/network/go_server/app/repositories"
	"path/filepath"
	"time"
)

type FileHistoryService struct {
	repo     *repositories.FileHistoryRepository
	nodeRepo *repositories.FileNodeRepository
}

func NewFileHistoryService(repo *repositories.FileHistoryRepository, nodeRepo *repositories.FileNodeRepository) *FileHistoryService {
	return &FileHistoryService{repo: repo, nodeRepo: nodeRepo}
}

type SyncEvent struct {
	UUID      string `json:"uuid"`
	Action    string `json:"action"`
	Path      string `json:"path"`
	OldPath   string `json:"old_path"`
	Type      string `json:"type"` // "file" or "folder"
	Timestamp int64  `json:"ts"`   // Unix timestamp
}

func (s *FileHistoryService) SyncEvents(deviceID string, events []SyncEvent) error {
	var histories []models.DeviceFileHistory
	now := time.Now()

	for _, evt := range events {
		h := models.DeviceFileHistory{
			DeviceID:  deviceID,
			FileUUID:  evt.UUID,
			Action:    evt.Action,
			Path:      evt.Path,
			OldPath:   evt.OldPath,
			EventTime: time.Unix(evt.Timestamp, 0),
			SyncedAt:  now,
			CreatedAt: now,
		}
		histories = append(histories, h)

		// Update Directory Tree
		s.processFileNode(deviceID, evt)
	}

	if len(histories) > 0 {
		return s.repo.BulkCreate(histories)
	}
	return nil
}

func (s *FileHistoryService) processFileNode(deviceID string, evt SyncEvent) {
	// Handle Create/Modify/Rename/Delete
	switch evt.Action {
	case "create", "modify", "rename":
		// Resolve Parent
		parentID, _ := s.nodeRepo.EnsureParent(deviceID, evt.Path)

		// Find or Create Node
		node, err := s.nodeRepo.FindByUUID(evt.UUID)
		if err != nil {
			// New Node
			node = &models.FileNode{
				DeviceID:  deviceID,
				ParentID:  parentID,
				UUID:      evt.UUID,
				Name:      filepath.Base(evt.Path),
				Path:      evt.Path,
				Type:      evt.Type,
				IsDeleted: false,
			}
		} else {
			// Update Existing
			if evt.Action == "rename" && node.Path != evt.Path {
				oldPath := node.Path
				node.Path = evt.Path
				node.Name = filepath.Base(evt.Path)
				node.ParentID = parentID
				// If it's a folder, update children paths
				if node.Type == "folder" {
					_ = s.nodeRepo.UpdatePaths(deviceID, oldPath, evt.Path)
				}
			} else {
				node.Path = evt.Path
				node.Name = filepath.Base(evt.Path)
				node.ParentID = parentID
			}
			node.IsDeleted = false
		}
		_ = s.nodeRepo.UpsertNode(node)

	case "delete", "move_out":
		_ = s.nodeRepo.MarkDeleted(evt.UUID)
	}
}

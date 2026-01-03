package services

import (
	"demo/network/go_server/app/models"
	"demo/network/go_server/app/repositories"
)

type DirectoryTreeService struct {
	nodeRepo *repositories.FileNodeRepository
}

func NewDirectoryTreeService(nodeRepo *repositories.FileNodeRepository) *DirectoryTreeService {
	return &DirectoryTreeService{nodeRepo: nodeRepo}
}

type TreeQuery struct {
	DeviceID    string `json:"device_id"`
	ParentID    *uint  `json:"parent_id"`
	Page        int    `json:"page"`
	Size        int    `json:"size"`
	ShowDeleted bool   `json:"show_deleted"`
}

type TreeResponse struct {
	Nodes []models.FileNode `json:"nodes"`
	Total int64             `json:"total"`
	Page  int               `json:"page"`
	Size  int               `json:"size"`
}

func (s *DirectoryTreeService) GetTree(query TreeQuery) (*TreeResponse, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Size <= 0 {
		query.Size = 20
	}

	nodes, total, err := s.nodeRepo.QueryTree(query.DeviceID, query.ParentID, query.Page, query.Size, query.ShowDeleted)
	if err != nil {
		return nil, err
	}

	return &TreeResponse{
		Nodes: nodes,
		Total: total,
		Page:  query.Page,
		Size:  query.Size,
	}, nil
}

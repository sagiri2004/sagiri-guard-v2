package repositories

import (
	"demo/network/go_server/app/models"
	"path/filepath"

	"gorm.io/gorm"
)

type FileNodeRepository struct {
	db *gorm.DB
}

func NewFileNodeRepository(db *gorm.DB) *FileNodeRepository {
	return &FileNodeRepository{db: db}
}

func (r *FileNodeRepository) UpsertNode(node *models.FileNode) error {
	// 1. Resolve ParentID by Parent Path if not root?
	// Actually, client sends full path. We should probably identify parent by path.
	// But it's easier to use the Client's hierarchy if provided.
	// For now, let's just stick to UUID based upsert and we'll fix ParentID during sync processing.
	return r.db.Save(node).Error
}

func (r *FileNodeRepository) FindByUUID(uuid string) (*models.FileNode, error) {
	var node models.FileNode
	if err := r.db.Where("uuid = ?", uuid).First(&node).Error; err != nil {
		return nil, err
	}
	return &node, nil
}

func (r *FileNodeRepository) FindByPath(deviceID, path string) (*models.FileNode, error) {
	var node models.FileNode
	if err := r.db.Where("device_id = ? AND path = ? AND is_deleted = false", deviceID, path).First(&node).Error; err != nil {
		return nil, err
	}
	return &node, nil
}

func (r *FileNodeRepository) MarkDeleted(uuid string) error {
	// Soft delete the node itself
	if err := r.db.Model(&models.FileNode{}).Where("uuid = ?", uuid).Updates(map[string]interface{}{
		"is_deleted": true,
	}).Error; err != nil {
		return err
	}

	// If it's a folder, we need to recursively mark children?
	// Actually, if we use `LIKE path/%`, it's faster.
	var node models.FileNode
	if err := r.db.Where("uuid = ?", uuid).First(&node).Error; err == nil {
		if node.Type == "folder" {
			return r.db.Model(&models.FileNode{}).
				Where("device_id = ? AND path LIKE ?", node.DeviceID, node.Path+"/%").
				Updates(map[string]interface{}{
					"is_deleted": true,
				}).Error
		}
	}
	return nil
}

func (r *FileNodeRepository) UpdatePaths(deviceID, oldPath, newPath string) error {
	// 1. Update the node itself (handled by caller usually via UUID)
	// 2. Update all children prefix
	// This is for rename/move operations on folders.

	// Example: rename /a/b to /a/c
	// Child /a/b/d becomes /a/c/d

	// Update path = REPLACE(path, oldPath, newPath) where path LIKE oldPath/%
	return r.db.Exec("UPDATE file_nodes SET path = REPLACE(path, ?, ?) WHERE device_id = ? AND path LIKE ?",
		oldPath+"/", newPath+"/", deviceID, oldPath+"/%").Error
}

func (r *FileNodeRepository) QueryTree(deviceID string, parentID *uint, page, size int, showDeleted bool) ([]models.FileNode, int64, error) {
	var nodes []models.FileNode
	var total int64

	query := r.db.Model(&models.FileNode{}).Where("device_id = ?", deviceID)

	if parentID != nil {
		query = query.Where("parent_id = ?", *parentID)
	} else {
		// Roots (usually paths with only 1 or 2 parts depending on mounting,
		// but let's assume we use ParentID = nil for roots)
		query = query.Where("parent_id IS NULL")
	}

	if !showDeleted {
		query = query.Where("is_deleted = false")
	}

	query.Count(&total)

	err := query.Offset((page - 1) * size).Limit(size).Order("type desc, name asc").Find(&nodes).Error
	return nodes, total, err
}

func (r *FileNodeRepository) EnsureParent(deviceID, path string) (*uint, error) {
	if path == "." || path == "/" || path == "" {
		return nil, nil
	}

	parentPath := filepath.Dir(path)
	if parentPath == "." || parentPath == "/" || parentPath == "" {
		return nil, nil
	}

	// Try find parent
	var parent models.FileNode
	err := r.db.Where("device_id = ? AND path = ? AND is_deleted = false", deviceID, parentPath).First(&parent).Error
	if err == nil {
		return &parent.ID, nil
	}

	if err == gorm.ErrRecordNotFound {
		// Recursive create parent folder?
		// Note: Client should ideally send events for folders too.
		// If not, we create a placeholder folder.
		grandParentID, _ := r.EnsureParent(deviceID, parentPath)

		parent = models.FileNode{
			DeviceID:  deviceID,
			ParentID:  grandParentID,
			UUID:      "placeholder-" + parentPath, // We don't have UUID yet
			Name:      filepath.Base(parentPath),
			Path:      parentPath,
			Type:      "folder",
			IsDeleted: false,
		}
		if err := r.db.Create(&parent).Error; err != nil {
			return nil, err
		}
		return &parent.ID, nil
	}

	return nil, err
}

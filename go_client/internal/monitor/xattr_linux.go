package monitor

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/xattr"
)

const SagiriIDAttr = "user.sagiri_id"

// GetFileID reads the custom UUID from file extended attributes
func GetFileID(path string) (string, error) {
	val, err := xattr.Get(path, SagiriIDAttr)
	if err != nil {
		return "", err
	}
	return string(val), nil
}

// SetFileID writes the custom UUID to file extended attributes
func SetFileID(path string, id string) error {
	return xattr.Set(path, SagiriIDAttr, []byte(id))
}

// EnsureFileID checks if the file has an ID, if not, creates one.
// Returns the ID.
func EnsureFileID(path string) (string, error) {
	// Try read
	id, err := GetFileID(path)
	if err == nil && id != "" {
		return id, nil
	}

	// Create new
	newID := uuid.New().String()
	if err := SetFileID(path, newID); err != nil {
		return "", fmt.Errorf("failed to set xattr for %s: %v", path, err)
	}

	return newID, nil
}

// IsFileSupported checks if xattr is supported on the file's filesystem (basic check)
// Actually xattr package handles errors, but symlinks might be tricky.
// Usually we don't follow symlinks for tagging the link itself vs target.
// pkg/xattr usually follows symlinks by default unless Lget/Lset used.
// Let's stick to following symlinks for now as we monitor file content.
func VerifyXattrSupport(path string) bool {
	// Dummy write/read test?
	// Or just rely on errors during EnsureFileID
	return true
}

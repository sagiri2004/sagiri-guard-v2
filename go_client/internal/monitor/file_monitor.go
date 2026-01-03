//go:build linux

package monitor

import (
	"crypto/sha1"
	"demo/network/go_client/internal/config"
	dbpkg "demo/network/go_client/internal/db"
	"demo/network/go_client/internal/logger"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"gorm.io/gorm/clause"
)

type ActionType string

const (
	ActionCreate  ActionType = "create"
	ActionModify  ActionType = "modify"
	ActionDelete  ActionType = "delete"
	ActionRename  ActionType = "rename"
	ActionMoveOut ActionType = "move_out"
)

type FileEvent struct {
	Action    ActionType
	Path      string
	OldPath   string
	Type      string // "file" or "folder"
	Timestamp time.Time
}

const (
	eventBufferSize = 10000 // Tăng buffer để chịu tải khi xóa folder lớn
	dbWorkerCount   = 5     // Số lượng worker ghi DB song song
)

type FileMonitor struct {
	watcher    *fsnotify.Watcher
	watchedDir map[string]struct{}
	mu         sync.RWMutex

	eventChan chan FileEvent
	dbChan    chan FileEvent
	stop      chan struct{}
	wg        sync.WaitGroup
	once      sync.Once
}

func NewFileMonitor(paths []string) (*FileMonitor, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	fm := &FileMonitor{
		watcher:    watcher,
		watchedDir: make(map[string]struct{}),
		eventChan:  make(chan FileEvent, eventBufferSize),
		dbChan:     make(chan FileEvent, eventBufferSize),
		stop:       make(chan struct{}),
	}

	// 1. Khởi động DB Workers trước
	for i := 0; i < dbWorkerCount; i++ {
		fm.wg.Add(1)
		go fm.dbWorker()
	}

	// 2. Auto Migrate ID
	db := dbpkg.Get()
	if db != nil {
		db.AutoMigrate(&dbpkg.MonitoredFile{}, &dbpkg.FileChangeEvent{})
	}

	// 3. Add các đường dẫn ban đầu
	for _, raw := range paths {
		abs, _ := filepath.Abs(raw)
		info, err := os.Stat(abs)
		if err != nil {
			continue
		}

		target := abs
		if !info.IsDir() {
			target = filepath.Dir(abs)
		}

		if err := fm.watchRecursive(target); err != nil {
			logger.Errorf("Initial watch failed for %s: %v", target, err)
		}

		// Initial Tagging for existing files
		if info.IsDir() {
			filepath.WalkDir(target, func(path string, d os.DirEntry, err error) error {
				if err == nil && !d.IsDir() {
					_, _ = EnsureFileID(path)
				}
				return nil
			})
		} else {
			_, _ = EnsureFileID(abs)
		}
	}

	return fm, nil
}

// MonitorFiles trả về channel để layer trên tiêu thụ (nếu cần)
func (f *FileMonitor) MonitorFiles() <-chan FileEvent {
	f.wg.Add(1)
	go f.mainLoop()
	return f.eventChan
}

func (f *FileMonitor) mainLoop() {
	defer f.wg.Done()
	for {
		select {
		case <-f.stop:
			return
		case err, ok := <-f.watcher.Errors:
			if !ok {
				return
			}
			logger.Errorf("Inotify error: %v", err)
		case evt, ok := <-f.watcher.Events:
			if !ok {
				return
			}
			f.handleFsnotifyEvent(evt)
		}
	}
}

func (f *FileMonitor) handleFsnotifyEvent(evt fsnotify.Event) {
	path := filepath.Clean(evt.Name)
	now := time.Now()

	// Ignore hidden files (optional)
	if filepath.Base(path)[0] == '.' {
		return
	}

	isDir := f.checkIsDir(path)

	// Logic Xattr & Action
	var action ActionType
	// var fileID string // Removed unused variable
	var fromPath string // For Rename

	// Note: fsnotify events are bitmask
	if evt.Op&fsnotify.Create != 0 {
		action = ActionCreate
		if isDir {
			// New Directory / Moved Directory
			// We must recursively scan it to find moved files or new files
			_ = f.watchRecursive(path) // This only adds watch

			// Manually scan contents
			filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}

				// Process File
				id, err := GetFileID(p)
				var subAction ActionType

				if err == nil && id != "" {
					subAction = ActionRename // Moved here with parent
				} else {
					subAction = ActionCreate // New file
					id, _ = EnsureFileID(p)
				}

				// Emit event for this file
				f.emitEvent(subAction, p, "", "file", now)
				return nil
			})

		} else {
			// Check if it has an ID (Move/Rename) OR New File
			id, err := GetFileID(path)
			if err == nil && id != "" {
				// Has ID -> It was Moved/Renamed here
				// fileID = id
				// Determine Old Path?
				// SQLite query to find last known path for this ID?
				action = ActionRename
			} else {
				// New File -> Tag it
				// newID, _ := EnsureFileID(path)
				_, _ = EnsureFileID(path)
				// fileID = newID
			}
		}
	} else if evt.Op&fsnotify.Write != 0 {
		action = ActionModify
		// Ensure ID exists (in case it was stripped or created externally without tag)
		if !isDir {
			_, _ = EnsureFileID(path) // fileID = ...
		}
	} else if evt.Op&fsnotify.Remove != 0 {
		action = ActionDelete
		// Can't read Xattr of deleted file.
		// Need lookup in DB by Path to find ID?
		// We can try to look up in DB inside dbWorker.
		if isDir {
			f.unwatchInternal(path)
		}
	} else if evt.Op&fsnotify.Rename != 0 {
		// This is usually the "Old Path" event in Rename (MOVED_FROM).
		// followed by Create (MOVED_TO).
		action = ActionMoveOut
		if isDir {
			f.unwatchInternal(path)
		}
	}

	if action != "" {
		itemType := "file"
		if isDir {
			itemType = "folder"
		}
		f.emitEvent(action, path, fromPath, itemType, now)
	}
}

func (f *FileMonitor) emitEvent(action ActionType, path, oldPath, itemType string, timestamp time.Time) {
	event := FileEvent{
		Action:    action,
		Path:      path,
		OldPath:   oldPath,
		Type:      itemType,
		Timestamp: timestamp,
	}

	select {
	case f.dbChan <- event:
	default:
		logger.Warnf("DB Queue full, event dropped for %s", path)
	}

	select {
	case f.eventChan <- event:
	default:
	}
}

// dbWorker chạy song song để ghi dữ liệu, không làm nghẽn inotify loop
func (f *FileMonitor) dbWorker() {
	defer f.wg.Done()
	for {
		select {
		case <-f.stop:
			return
		case evt := <-f.dbChan:
			f.persistToDB(evt)
		}
	}
}

func (f *FileMonitor) persistToDB(evt FileEvent) {
	db := dbpkg.Get()
	if db == nil {
		return
	}

	// eventId := "" // Removed unused

	// 1. Resolve ID and Handle Directory Cascading
	// If it's a directory (how do we know? We can't Stat a deleted dir).
	// We infer it's a directory if we can't find a File entry for this exact path
	// BUT we find children with this prefix?
	// Or we rely on the fact that if it was a monitored directory, we tracked it?
	// Our DB currently only tracks "MonitoredFile" (lines). It does NOT track directories explicitly.
	// So if `evt.Path` is a directory, `GetFileID` fails (no xattr on dir usually/or we don't care).
	// But we need to update children.

	// Strategy: Always try to update children assuming it MIGHT be a directory.
	// Update all MonitoredFile where current_path LIKE 'evt.Path/%'

	if evt.Action == ActionDelete || evt.Action == ActionMoveOut {
		// Cascade Update Children (Implicit Delete/MoveOut)
		// We only update their status. We don't know their new path (for MoveOut) here.
		// For Delete, they are gone.
		var children []dbpkg.MonitoredFile
		db.Where("current_path LIKE ?", evt.Path+"/%").Find(&children)

		for _, child := range children {
			// Log History for Child
			history := dbpkg.FileChangeEvent{
				FileUUID:  child.UUID,
				ItemType:  child.ItemType,
				Action:    string(evt.Action),
				FromPath:  child.CurrentPath,
				ToPath:    "", // Unknown for MoveOut (until they reappear), Empty for Delete
				Timestamp: evt.Timestamp,
			}
			db.Create(&history)

			// Update Status
			db.Model(&dbpkg.MonitoredFile{}).Where("uuid = ?", child.UUID).Updates(map[string]interface{}{
				"last_action":   string(evt.Action),
				"last_event_at": evt.Timestamp,
			})
		}
	}

	// 2. Handle the entry itself (File or Folder)
	var fileID string
	itemType := evt.Type

	if evt.Action == ActionDelete || evt.Action == ActionMoveOut {
		var mf dbpkg.MonitoredFile
		if err := db.Where("current_path = ?", evt.Path).First(&mf).Error; err == nil {
			fileID = mf.UUID
			itemType = mf.ItemType
		}
	} else {
		if itemType == "folder" {
			// Deterministic ID for folders: sha1(deviceID + ":" + path)
			devCfg, _ := config.LoadDeviceConfig()
			deviceID := "unknown"
			if devCfg != nil {
				deviceID = devCfg.DeviceID
			}
			h := sha1.New()
			h.Write([]byte(deviceID + ":" + evt.Path))
			fileID = "folder-" + hex.EncodeToString(h.Sum(nil))
		} else {
			// Create/Modify/RenameIn: Read Xattr
			fileID, _ = GetFileID(evt.Path)
		}
	}

	if fileID != "" {
		// eventId = fileID // Removed unused assignment
		// Update/Create MonitoredFile

		var fromPath string

		// If it's a known file, get old path for history
		// If it's a known file, get old path for history
		var prev dbpkg.MonitoredFile
		// Use Find + Limit to avoid "record not found" error log
		if db.Where("uuid = ?", fileID).Limit(1).Find(&prev).RowsAffected > 0 {
			fromPath = prev.CurrentPath
		}

		if evt.Action == ActionDelete {
			db.Model(&dbpkg.MonitoredFile{}).Where("uuid = ?", fileID).Updates(map[string]interface{}{
				"last_action":   string(evt.Action),
				"last_event_at": evt.Timestamp,
			})
		} else if evt.Action == ActionMoveOut {
			db.Model(&dbpkg.MonitoredFile{}).Where("uuid = ?", fileID).Updates(map[string]interface{}{
				"last_action":   string(evt.Action),
				"last_event_at": evt.Timestamp,
			})
		} else {
			// Create/Modify/Rename(In)
			if fromPath == evt.Path {
				fromPath = ""
			}
			if evt.Action == ActionRename && fromPath == "" {
				fromPath = evt.OldPath // Use hint from monitor if available
			}

			mf := dbpkg.MonitoredFile{
				UUID:        fileID,
				CurrentPath: evt.Path,
				ItemType:    itemType,
				LastAction:  string(evt.Action),
				LastEventAt: evt.Timestamp,
			}
			db.Clauses(clause.OnConflict{
				UpdateAll: true,
			}).Create(&mf)
		}

		// Always Log History
		history := dbpkg.FileChangeEvent{
			FileUUID:  fileID,
			ItemType:  itemType,
			Action:    string(evt.Action),
			FromPath:  fromPath,
			ToPath:    evt.Path,
			Timestamp: evt.Timestamp,
		}
		if evt.Action == ActionDelete || evt.Action == ActionMoveOut {
			history.ToPath = ""
			history.FromPath = evt.Path
		}
		db.Create(&history)
	}
}

func (f *FileMonitor) watchRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}

		f.mu.Lock()
		if _, exists := f.watchedDir[path]; !exists {
			if err := f.watcher.Add(path); err == nil {
				f.watchedDir[path] = struct{}{}
				logger.Debugf("Watching: %s", path)
			}
		}
		f.mu.Unlock()
		return nil
	})
}

func (f *FileMonitor) unwatchInternal(path string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Khi xóa 1 folder, inotify tự động gỡ các watch con bên trong,
	// nhưng ta cần dọn dẹp map để tránh rò rỉ bộ nhớ.
	for p := range f.watchedDir {
		if p == path || filepath.HasPrefix(p, path+string(os.PathSeparator)) {
			delete(f.watchedDir, p)
			_ = f.watcher.Remove(p)
		}
	}
}

func (f *FileMonitor) checkIsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (f *FileMonitor) Close() error {
	f.once.Do(func() {
		close(f.stop)
		_ = f.watcher.Close()
	})
	f.wg.Wait()
	close(f.eventChan)
	close(f.dbChan)
	return nil
}

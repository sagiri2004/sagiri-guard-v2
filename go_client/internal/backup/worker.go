package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
	"unsafe"

	"demo/network/go_client/internal/config"
	dbpkg "demo/network/go_client/internal/db"
	"demo/network/go_client/internal/logger"
)

/*
#include <stdlib.h>
#include "../../../client/core.h"
*/
import "C"

var clientCtx *C.ClientContext

func SetClientContext(ctx unsafe.Pointer) {
	clientCtx = (*C.ClientContext)(ctx)
}

type BackupWorker struct {
	stopChan  chan struct{}
	jobsChan  chan dbpkg.MonitoredFile
	activeMap sync.Map // map[string]bool - Track file UUIDs currently being backed up
	wg        sync.WaitGroup
}

func NewBackupWorker() *BackupWorker {
	return &BackupWorker{
		stopChan: make(chan struct{}),
		jobsChan: make(chan dbpkg.MonitoredFile, 20),
	}
}

func (w *BackupWorker) Start() {
	logger.Infof("[Backup] Worker Started (Concurrent Pool: 3 workers)")
	w.wg.Add(1)
	go w.loop()

	// Start Workers
	for i := 0; i < 3; i++ {
		w.wg.Add(1)
		go w.worker(i)
	}
}

func (w *BackupWorker) Stop() {
	close(w.stopChan)
	w.wg.Wait()
	logger.Info("[Backup] Worker Stopped")
}

func (w *BackupWorker) loop() {
	defer w.wg.Done()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			close(w.jobsChan)
			return
		case <-ticker.C:
			w.dispatchJobs()
		}
	}
}

func (w *BackupWorker) worker(id int) {
	defer w.wg.Done()
	for f := range w.jobsChan {
		devCfg, _ := config.LoadDeviceConfig()
		if devCfg == nil || devCfg.DeviceID == "" {
			w.activeMap.Delete(f.UUID)
			continue
		}

		logger.Infof("[Backup] Worker %d handling %s", id, f.CurrentPath)
		if err := w.backupFile(f, devCfg.DeviceID); err != nil {
			logger.Errorf("[Backup] Worker %d error for %s: %v", id, f.CurrentPath, err)
		} else {
			db := dbpkg.Get()
			if db != nil {
				db.Model(&f).Update("last_backup_at", time.Now())
			}
			logger.Infof("[Backup] Worker %d completed %s", id, f.CurrentPath)
		}
		w.activeMap.Delete(f.UUID)
	}
}

func (w *BackupWorker) dispatchJobs() {
	db := dbpkg.Get()
	if db == nil || clientCtx == nil {
		return
	}

	// 1. Find candidates
	var files []dbpkg.MonitoredFile
	err := db.Where("item_type = ? AND (last_backup_at IS NULL OR last_backup_at < last_event_at)", "file").
		Order("last_event_at desc").Limit(10).Find(&files).Error
	if err != nil {
		return
	}

	for _, f := range files {
		// Only queue if not already active
		if _, loaded := w.activeMap.LoadOrStore(f.UUID, true); !loaded {
			select {
			case w.jobsChan <- f:
				// Queued
			default:
				// Channel full, skip for now
				w.activeMap.Delete(f.UUID)
			}
		}
	}
}

func (w *BackupWorker) backupFile(f dbpkg.MonitoredFile, deviceID string) error {
	file, err := os.Open(f.CurrentPath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, _ := file.Stat()
	totalSize := info.Size()

	// 1. Calculate Head Hash (64KB) for integrity check
	headBuf := make([]byte, 64*1024)
	nHead, _ := file.Read(headBuf)
	hHead := sha256.New()
	hHead.Write(headBuf[:nHead])
	headHash := hex.EncodeToString(hHead.Sum(nil))
	file.Seek(0, 0) // Reset to start

	var transferID string
	var offset int64 = 0
	var respBuf [4096]C.char

	// 2. Try Resume
	resumePayload := map[string]interface{}{
		"device_id":  deviceID,
		"file_uuid":  f.UUID,
		"head_hash":  headHash,
		"total_size": totalSize,
	}
	jResume, _ := json.Marshal(resumePayload)
	cResume := C.CString(string(jResume))
	respBuf[0] = 0
	res := C.client_backup_resume(clientCtx, cResume, &respBuf[0])
	C.free(unsafe.Pointer(cResume))

	if res != 0 {
		var resumeResp struct {
			TransferID string `json:"transfer_id"`
			Offset     int64  `json:"offset"`
			Status     string `json:"status"` // "found", "not_found", "mismatch"
		}
		json.Unmarshal([]byte(C.GoString(&respBuf[0])), &resumeResp)
		if resumeResp.Status == "found" {
			transferID = resumeResp.TransferID
			offset = resumeResp.Offset
			logger.Infof("[Backup] Resuming %s from offset %d", f.CurrentPath, offset)
		}
	}

	// 3. If not resumed, Init Session
	if transferID == "" {
		initPayload := map[string]interface{}{
			"device_id":  deviceID,
			"file_uuid":  f.UUID,
			"file_name":  info.Name(),
			"total_size": totalSize,
			"head_hash":  headHash,
		}
		jsonInit, _ := json.Marshal(initPayload)
		cInit := C.CString(string(jsonInit))
		respBuf[0] = 0
		res := C.client_backup_init(clientCtx, cInit, &respBuf[0])
		C.free(unsafe.Pointer(cInit))

		if res == 0 {
			return fmt.Errorf("backup_init failed: %s", C.GoString(&respBuf[0]))
		}

		var initResp struct {
			TransferID string `json:"transfer_id"`
		}
		json.Unmarshal([]byte(C.GoString(&respBuf[0])), &initResp)
		transferID = initResp.TransferID
		offset = 0
	}

	// 4. Upload Chunks from current offset
	hash := sha256.New()
	if offset > 0 {
		// If resuming, we can't easily recalculate partial hash without reading all previous data.
		// For now, let's assume we don't verify full hash at the end if resumed, OR we must read from start.
		// BETTER: Read from start to update hash, but skip upload until offset.
		buffer := make([]byte, 128*1024)
		var readAt int64 = 0
		for readAt < offset {
			toRead := offset - readAt
			if toRead > int64(len(buffer)) {
				toRead = int64(len(buffer))
			}
			n, _ := file.Read(buffer[:toRead])
			if n == 0 {
				break
			}
			hash.Write(buffer[:n])
			readAt += int64(n)
		}
	} else {
		file.Seek(0, 0)
	}

	buffer := make([]byte, 16*1024*1024) // 16MB Chunk
	for {
		n, err := file.Read(buffer)
		if n > 0 {
			hash.Write(buffer[:n])

			chunkPayload := map[string]interface{}{
				"transfer_id": transferID,
				"offset":      offset,
				"data_len":    int64(n),
				"data":        hex.EncodeToString(buffer[:n]),
			}
			jsonChunk, _ := json.Marshal(chunkPayload)
			cChunk := C.CString(string(jsonChunk))

			respBuf[0] = 0
			res := C.client_backup_chunk(clientCtx, cChunk, &respBuf[0])
			C.free(unsafe.Pointer(cChunk))

			if res == 0 {
				cancelPayload := map[string]interface{}{"transfer_id": transferID}
				jCancel, _ := json.Marshal(cancelPayload)
				cCancel := C.CString(string(jCancel))
				C.client_backup_cancel(clientCtx, cCancel, nil)
				C.free(unsafe.Pointer(cCancel))
				return fmt.Errorf("backup_chunk failed at offset %d: %s", offset, C.GoString(&respBuf[0]))
			}

			offset += int64(n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	// 5. Finish Session
	finishPayload := map[string]interface{}{
		"transfer_id": transferID,
		"server_path": "",
		"file_hash":   hex.EncodeToString(hash.Sum(nil)),
	}
	jsonFinish, _ := json.Marshal(finishPayload)
	cFinish := C.CString(string(jsonFinish))
	respBuf[0] = 0
	res = C.client_backup_finish(clientCtx, cFinish, &respBuf[0])
	C.free(unsafe.Pointer(cFinish))

	if res == 0 {
		return fmt.Errorf("backup_finish failed: %s", C.GoString(&respBuf[0]))
	}

	return nil
}

package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unsafe"

	"demo/network/go_client/internal/config"
	dbpkg "demo/network/go_client/internal/db"
	"demo/network/go_client/internal/logger"
)

/*
#include <stdlib.h>
#include <string.h>
#include "../../../client/core.h"
*/
import "C"

type RestoreJob struct {
	FileUUID   string
	Version    int
	TransferID string // Optional, for Resume
}

type RestoreWorker struct {
	ticker     *time.Ticker
	stop       chan bool
	numWorkers int
}

var (
	restoreQueue   = make(chan RestoreJob, 100)
	activeRestores sync.Map // Map[string]bool (FileUUID)
	restoreOnce    sync.Once
)

func NewRestoreWorker(numWorkers int) *RestoreWorker {
	return &RestoreWorker{
		ticker:     time.NewTicker(30 * time.Second),
		stop:       make(chan bool),
		numWorkers: numWorkers,
	}
}

func (w *RestoreWorker) Start() {
	restoreOnce.Do(func() {
		logger.Infof("[Init] Restore Worker Pool Started (%d workers)", w.numWorkers)
		for i := 0; i < w.numWorkers; i++ {
			go restoreWorkerLoop()
		}

		go func() {
			// Initial recovery
			RestoreRecovery()

			for {
				select {
				case <-w.ticker.C:
					RestoreRecovery()
				case <-w.stop:
					return
				}
			}
		}()
	})
}

func (w *RestoreWorker) Stop() {
	w.stop <- true
}

func RestoreRecovery() {
	db := dbpkg.Get()
	if db == nil {
		return
	}

	var sessions []dbpkg.LocalRestoreSession
	if err := db.Where("status = ?", "IN_PROGRESS").Find(&sessions).Error; err == nil {
		for _, s := range sessions {
			// Check if already active
			if _, loaded := activeRestores.Load(s.FileUUID); loaded {
				continue
			}

			logger.Infof("[Restore] Recovering/Queuing interrupted session: %s", s.TransferID)
			restoreQueue <- RestoreJob{
				FileUUID:   s.FileUUID,
				Version:    s.Version,
				TransferID: s.TransferID,
			}
		}
	}
}

func HandleRestoreCmd(payload string) {
	var req struct {
		FileUUID string `json:"file_uuid"`
		Version  int    `json:"version"`
	}
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		logger.Errorf("[Restore] Invalid Payload: %v", err)
		return
	}

	restoreQueue <- RestoreJob{
		FileUUID: req.FileUUID,
		Version:  req.Version,
	}
}

func restoreWorkerLoop() {
	for job := range restoreQueue {
		// Verify if we should process it
		if _, loaded := activeRestores.LoadOrStore(job.FileUUID, true); loaded {
			continue // Already being restored
		}
		performRestore(job)
		activeRestores.Delete(job.FileUUID)
	}
}

func performRestore(job RestoreJob) {
	devCfg, _ := config.LoadDeviceConfig()
	if devCfg == nil || devCfg.DeviceID == "" || clientCtx == nil {
		return
	}

	var session dbpkg.LocalRestoreSession
	db := dbpkg.Get()

	// 1. Init or Resume
	var initResp struct {
		TransferID string `json:"transfer_id"`
		FileName   string `json:"file_name"`
		Version    int    `json:"version"`
		TotalSize  int64  `json:"total_size"`
		FileHash   string `json:"file_hash"`
		Status     string `json:"status"`
	}

	var respBuf [4096]C.char
	if job.TransferID != "" {
		// RESUME
		logger.Infof("[Restore] Resuming session %s", job.TransferID)
		if db != nil {
			if err := db.Where("transfer_id = ?", job.TransferID).First(&session).Error; err != nil {
				logger.Errorf("[Restore] Session not found in DB: %s", job.TransferID)
				return
			}
		}

		resumePayload := map[string]interface{}{"transfer_id": job.TransferID}
		jResume, _ := json.Marshal(resumePayload)
		cResume := C.CString(string(jResume))
		res := C.client_restore_resume(clientCtx, cResume, &respBuf[0])
		C.free(unsafe.Pointer(cResume))

		if res == 0 {
			logger.Errorf("[Restore] Resume Failed on Server for %s", job.TransferID)
			return
		}
		json.Unmarshal([]byte(C.GoString(&respBuf[0])), &initResp)
	} else {
		// NEW INIT
		initPayload := map[string]interface{}{
			"device_id": devCfg.DeviceID,
			"file_uuid": job.FileUUID,
			"version":   job.Version,
		}
		jInit, _ := json.Marshal(initPayload)
		cInit := C.CString(string(jInit))
		res := C.client_restore_init(clientCtx, cInit, &respBuf[0])
		C.free(unsafe.Pointer(cInit))

		if res == 0 {
			logger.Errorf("[Restore] Init Failed: %s", C.GoString(&respBuf[0]))
			return
		}
		json.Unmarshal([]byte(C.GoString(&respBuf[0])), &initResp)

		if initResp.Status != "ok" {
			logger.Errorf("[Restore] Init Status: %s", initResp.Status)
			return
		}

		// Determine Local Path
		var originalPath string
		if db != nil {
			var mf dbpkg.MonitoredFile
			if err := db.Where("uuid = ?", job.FileUUID).First(&mf).Error; err == nil {
				originalPath = mf.CurrentPath
			}
		}

		destPath := originalPath
		if destPath == "" {
			restoreDir := "restored"
			os.MkdirAll(restoreDir, 0755)
			destPath = filepath.Join(restoreDir, initResp.FileName)
		}

		session = dbpkg.LocalRestoreSession{
			TransferID:    initResp.TransferID,
			FileUUID:      job.FileUUID,
			Version:       initResp.Version,
			LocalPath:     destPath + ".part",
			CurrentOffset: 0,
			TotalSize:     initResp.TotalSize,
			FileHash:      initResp.FileHash,
			Status:        "IN_PROGRESS",
			UpdatedAt:     time.Now(),
		}
		if db != nil {
			db.Create(&session)
		}
	}

	// 2. Pull Chunks
	logger.Infof("[Restore] Restoring to %s (Transfer: %s)", session.LocalPath, session.TransferID)

	file, err := os.OpenFile(session.LocalPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Errorf("[Restore] File Open Error: %v", err)
		return
	}
	defer file.Close()

	if session.CurrentOffset > 0 {
		file.Seek(session.CurrentOffset, 0)
	}

	chunkSize := 1024 * 1024 * 16 // 16MB
	hash := sha256.New()

	// If resuming, we need to hash the existing part to maintain SHA256 integrity
	if session.CurrentOffset > 0 {
		logger.Infof("[Restore] Re-hashing existing part (%d bytes)...", session.CurrentOffset)
		existingFile, _ := os.Open(session.LocalPath)
		buf := make([]byte, 1024*1024)
		var hashed int64 = 0
		for hashed < session.CurrentOffset {
			n, _ := existingFile.Read(buf)
			if n == 0 {
				break
			}
			hash.Write(buf[:n])
			hashed += int64(n)
		}
		existingFile.Close()
	}

	for session.CurrentOffset < session.TotalSize {
		toRead := int(session.TotalSize - session.CurrentOffset)
		if toRead > chunkSize {
			toRead = chunkSize
		}

		chunkReq := map[string]interface{}{
			"transfer_id": session.TransferID,
			"offset":      session.CurrentOffset,
			"size":        toRead,
		}
		jChunk, _ := json.Marshal(chunkReq)
		cChunk := C.CString(string(jChunk))

		chunkRespBuf := make([]byte, 64*1024*1024)
		res := C.client_restore_chunk(clientCtx, cChunk, (*C.char)(unsafe.Pointer(&chunkRespBuf[0])))
		C.free(unsafe.Pointer(cChunk))

		if res == 0 {
			logger.Errorf("[Restore] Chunk Pull Failed at %d", session.CurrentOffset)
			return
		}

		respStr := C.GoString((*C.char)(unsafe.Pointer(&chunkRespBuf[0])))
		var chunkResp struct {
			Data   string `json:"data"`
			Status string `json:"status"`
		}
		json.Unmarshal([]byte(respStr), &chunkResp)

		if chunkResp.Status != "ok" {
			logger.Errorf("[Restore] Chunk Status: %s at %d", chunkResp.Status, session.CurrentOffset)
			return
		}

		data, _ := hex.DecodeString(chunkResp.Data)
		file.Write(data)
		hash.Write(data)
		session.CurrentOffset += int64(len(data))
		session.UpdatedAt = time.Now()
		if db != nil {
			db.Save(&session)
		}
		logger.Infof("[Restore] [%s] Progress: %d/%d", session.TransferID, session.CurrentOffset, session.TotalSize)
	}

	// 3. Finish & Verify
	finishReq := map[string]interface{}{"transfer_id": session.TransferID}
	jFinish, _ := json.Marshal(finishReq)
	cFinish := C.CString(string(jFinish))
	var finishResp [1024]C.char
	C.client_restore_finish(clientCtx, cFinish, &finishResp[0])
	C.free(unsafe.Pointer(cFinish))

	file.Close()

	finalHash := hex.EncodeToString(hash.Sum(nil))
	if finalHash != session.FileHash {
		logger.Errorf("[Restore] Hash Mismatch! Expected %s, got %s", session.FileHash, finalHash)
		// We don't delete on mismatch if we want to resume, but here it's finished.
		// Maybe mark as failed?
		session.Status = "FAILED"
		if db != nil {
			db.Save(&session)
		}
	} else {
		// Success: Move .part to final
		destPath := session.LocalPath[:len(session.LocalPath)-5] // Strip .part
		os.Remove(destPath)
		if err := os.Rename(session.LocalPath, destPath); err != nil {
			logger.Errorf("[Restore] Rename Failed: %v", err)
		} else {
			logger.Infof("[Restore] Successfully restored to %s", destPath)
			session.Status = "DONE"
			if db != nil {
				db.Save(&session)
			}
		}
	}
}

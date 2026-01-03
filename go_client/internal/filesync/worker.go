package filesync

import (
	"encoding/json"
	"fmt"
	"strings"
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

// We need access to C context to send messages.
// We can use a global setter or pass it in Start.
var clientCtx *C.ClientContext

func SetClientContext(ctx unsafe.Pointer) {
	clientCtx = (*C.ClientContext)(ctx)
}

type SyncWorker struct {
	stopChan chan struct{}
	wg       sync.WaitGroup
}

func NewSyncWorker() *SyncWorker {
	return &SyncWorker{
		stopChan: make(chan struct{}),
	}
}

func (w *SyncWorker) Start() {
	logger.Info("[Sync] Worker Started")
	w.wg.Add(1)
	go w.metricsLoop()
}

func (w *SyncWorker) Stop() {
	close(w.stopChan)
	w.wg.Wait()
	logger.Info("[Sync] Worker Stopped")
}

func (w *SyncWorker) metricsLoop() {
	defer w.wg.Done()
	ticker := time.NewTicker(10 * time.Second) // 10s Interval
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.doSync()
		}
	}
}

func (w *SyncWorker) doSync() {
	db := dbpkg.Get()
	if db == nil {
		fmt.Println("[Sync] DB is nil")
		return
	}
	if clientCtx == nil {
		fmt.Println("[Sync] ClientCtx is nil (Wait for init?)")
		return
	}

	devCfg, err := config.LoadDeviceConfig()
	if err != nil || devCfg.DeviceID == "" {
		fmt.Println("[Sync] Error loading device config or ID empty")
		return
	}

	// 1. Get Unsynced Events
	// Watermark tracking using SyncState table
	type SyncState struct {
		Key   string `gorm:"primaryKey"`
		Value uint
	}
	db.AutoMigrate(&SyncState{})

	var state SyncState
	db.Where("key = ?", "file_history").FirstOrCreate(&state, SyncState{Key: "file_history", Value: 0})

	// fmt.Printf("[Sync] Checking events > %d\n", state.Value) // Verbose check

	var events []dbpkg.FileChangeEvent
	// Fetch batch of 50
	if err := db.Where("id > ?", state.Value).Limit(50).Order("id asc").Find(&events).Error; err != nil {
		logger.Errorf("[Sync] DB Error: %v", err)
		return
	}

	if len(events) == 0 {
		// fmt.Println("[Sync] No new events found.")
		return
	}

	// 2. Prepare Payload
	type PayloadEvent struct {
		UUID      string `json:"uuid"`
		Action    string `json:"action"`
		Path      string `json:"path"`
		OldPath   string `json:"old_path"`
		Type      string `json:"type"` // "file" or "folder"
		Timestamp int64  `json:"ts"`
	}

	var payloadEvents []PayloadEvent
	var maxID uint

	for _, e := range events {
		payloadEvents = append(payloadEvents, PayloadEvent{
			UUID:      e.FileUUID,
			Action:    e.Action,
			Path:      e.ToPath, // Current Path
			OldPath:   e.FromPath,
			Type:      e.ItemType,
			Timestamp: e.Timestamp.Unix(), // Seconds
		})
		if e.ID > maxID {
			maxID = e.ID
		}
	}

	payload := map[string]interface{}{
		"device_id": devCfg.DeviceID,
		"events":    payloadEvents,
	}

	jsonBytes, _ := json.Marshal(payload)
	cPayload := C.CString(string(jsonBytes))
	defer C.free(unsafe.Pointer(cPayload))

	// 3. Send to Server (Type 0xE5)
	var respBuf [1024]C.char

	// Ensure client_file_sync is available in core.h/c
	res := C.client_file_sync(clientCtx, cPayload, &respBuf[0])

	// 4. Handle Response
	if res == 1 {
		respStr := C.GoString(&respBuf[0])
		fmt.Printf("[Sync] Server Response: %s\n", respStr)

		if strings.Contains(respStr, "success") {
			msg := fmt.Sprintf("[Sync] Synced %d events (MaxID: %d)", len(events), maxID)
			logger.Info(msg)
			fmt.Println(msg)

			// Update Watermark
			state.Value = maxID
			if err := db.Save(&state).Error; err != nil {
				fmt.Printf("[Sync] Failed to save watermark: %v\n", err)
			} else {
				fmt.Printf("[Sync] Watermark updated to %d\n", maxID)
			}
		} else {
			logger.Warnf("[Sync] Failed response: %s", respStr)
			fmt.Printf("[Sync] Failed response: %s\n", respStr)
		}
	} else {
		// Network error? Skip update watermark
		logger.Warn("[Sync] Network unavailable or timeout (res == 0)")
		fmt.Println("[Sync] Network unavailable or timeout (res == 0)")
	}
}

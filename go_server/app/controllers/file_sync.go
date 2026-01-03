package controllers

import (
	"demo/network/go_server/app/services"
	"demo/network/go_server/server" // Assuming we use server.SendResponse
	"encoding/json"
	"fmt"
)

// In Init, we will need to store the svc instance
// In Init, we will need to store the svc instance
// Using global FileHistorySvc from init.go

type FileSyncRequest struct {
	DeviceID string               `json:"device_id"`
	Events   []services.SyncEvent `json:"events"`
}

func HandleClientFileSync(clientID int, payload string) {
	fmt.Printf("[Controller] File Sync Request from Client %d\n", clientID)

	var req FileSyncRequest
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		fmt.Printf("[Error] Failed to unmarshal file sync payload: %v\n", err)
		server.SendResponse(clientID, 0xE7, 0, `{"status": "error", "message": "Invalid Payload"}`)
		return
	}

	if req.DeviceID == "" {
		server.SendResponse(clientID, 0xE7, 0, `{"status": "error", "message": "Missing Device ID"}`)
		return
	}

	// Call Service
	err := FileHistorySvc.SyncEvents(req.DeviceID, req.Events)
	if err != nil {
		fmt.Printf("[Error] Failed to sync events: %v\n", err)
		server.SendResponse(clientID, 0xE7, 0, `{"status": "error", "message": "DB Error"}`)
		return
	}

	// Success Response
	resp := fmt.Sprintf(`{"status": "success", "synced_count": %d}`, len(req.Events))
	// 0xE7 = MSG_CLIENT_FILE_SYNC_RESP
	// STATUS MUST BE 200 for Client to accept it
	server.SendResponse(clientID, 0xE7, 200, resp)
}

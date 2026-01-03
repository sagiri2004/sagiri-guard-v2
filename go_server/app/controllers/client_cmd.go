package controllers

import (
	"demo/network/go_server/server"
	"encoding/json"
	"fmt"
)

func HandleClientLogUpload(sock int, payload string) {
	fmt.Println("[Server] Received Log Upload from Client (Saving to DB)...")

	// Payload is JSON now: {"device_id": "...", "content": "..."}
	var meta struct {
		DeviceID string `json:"device_id"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal([]byte(payload), &meta); err != nil {
		server.SendResponse(sock, 0xD5, 400, map[string]string{"error": "Invalid Payload"})
		return
	}

	if meta.DeviceID == "" {
		meta.DeviceID = "UNKNOWN"
	}

	// Use LogService
	err := LogSvc.StoreLog(meta.DeviceID, meta.Content)
	if err != nil {
		server.SendResponse(sock, 0xD5, 500, map[string]string{"error": "Failed to store logs"})
		return
	}

	server.SendResponse(sock, 0xD5, 200, map[string]string{"status": "Logs Saved"})
}

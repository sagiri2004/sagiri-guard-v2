package controllers

import (
	"demo/network/go_server/server"
	"encoding/json"
	"fmt"
)

type BackupInitReq struct {
	DeviceID  string `json:"device_id"`
	FileUUID  string `json:"file_uuid"`
	FileName  string `json:"file_name"`
	TotalSize int64  `json:"total_size"`
}

type BackupChunkReq struct {
	TransferID string `json:"transfer_id"`
	Offset     int64  `json:"offset"`
	DataLen    int64  `json:"data_len"`
	// Note: Actual data is usually handled separately or embedded in JSON for small chunks
	// The user specified: offset; data_len; data[];
}

type BackupFinishReq struct {
	TransferID string `json:"transfer_id"`
	ServerPath string `json:"server_path"`
	FileHash   string `json:"file_hash"`
}

func HandleBackupInit(clientID int, payload string) {
	var req BackupInitReq
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		server.SendResponse(clientID, 0xF2, 400, `{"error": "Invalid Payload"}`)
		return
	}

	session, err := BackupSvc.InitSession(req.DeviceID, req.FileUUID, req.FileName, req.TotalSize)
	if err != nil {
		server.SendResponse(clientID, 0xF2, 500, fmt.Sprintf(`{"error": "%v"}`, err))
		return
	}

	server.SendResponse(clientID, 0xF2, 200, session)
}

func HandleBackupChunk(clientID int, payload string) {
	var req BackupChunkReq
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		server.SendResponse(clientID, 0xF4, 400, `{"error": "Invalid Payload"}`)
		return
	}

	err := BackupSvc.UpdateChunk(req.TransferID, req.Offset, req.DataLen)
	if err != nil {
		server.SendResponse(clientID, 0xF4, 500, fmt.Sprintf(`{"error": "%v"}`, err))
		return
	}

	server.SendResponse(clientID, 0xF4, 200, `{"status": "chunk_received"}`)
}

func HandleBackupFinish(clientID int, payload string) {
	var req BackupFinishReq
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		server.SendResponse(clientID, 0xF6, 400, `{"error": "Invalid Payload"}`)
		return
	}

	err := BackupSvc.FinishSession(req.TransferID, req.ServerPath, req.FileHash)
	if err != nil {
		server.SendResponse(clientID, 0xF6, 500, fmt.Sprintf(`{"error": "%v"}`, err))
		return
	}

	server.SendResponse(clientID, 0xF6, 200, `{"status": "backup_done"}`)
}

func HandleBackupCancel(clientID int, payload string) {
	var req struct {
		TransferID string `json:"transfer_id"`
	}
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		return // Silent fail for cancel or send error?
	}

	_ = BackupSvc.CancelSession(req.TransferID)
	// No explicit response for cancel 0xF7 usually but can add if needed
}

package controllers

import (
	"demo/network/go_server/server"
	"encoding/hex"
	"encoding/json"
)

type AdminRestoreReq struct {
	DeviceID string `json:"device_id"`
	FileUUID string `json:"file_uuid"`
	Version  int    `json:"version"`
}

type AdminRestoreResp struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type RestoreInitReq struct {
	DeviceID string `json:"device_id"`
	FileUUID string `json:"file_uuid"`
	Version  int    `json:"version"`
}

type RestoreInitResp struct {
	TransferID string `json:"transfer_id"`
	FileName   string `json:"file_name"`
	TotalSize  int64  `json:"total_size"`
	FileHash   string `json:"file_hash"`
	Status     string `json:"status"`
}

type RestoreResumeReq struct {
	TransferID string `json:"transfer_id"`
}

type RestoreResumeResp struct {
	TransferID string `json:"transfer_id"`
	FileName   string `json:"file_name"`
	TotalSize  int64  `json:"total_size"`
	FileHash   string `json:"file_hash"`
	Status     string `json:"status"`
}

type RestoreChunkReq struct {
	TransferID string `json:"transfer_id"`
	Offset     int64  `json:"offset"`
	Size       int    `json:"size"`
}

type RestoreChunkResp struct {
	Data    string `json:"data"` // Hex encoded
	DataLen int    `json:"data_len"`
	Status  string `json:"status"`
}

type RestoreFinishReq struct {
	TransferID string `json:"transfer_id"`
}

func HandleAdminRestore(clientID int, payload string) {
	var req AdminRestoreReq
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		server.SendResponse(clientID, 0x71, 400, `{"status": "error", "message": "Invalid Payload"}`)
		return
	}

	// 1. Send Command to Device
	cmdPayload := map[string]interface{}{
		"op":        "RESTORE_CMD",
		"file_uuid": req.FileUUID,
		"version":   req.Version,
	}
	jCmd, _ := json.Marshal(cmdPayload)

	success := server.SendToDevice(req.DeviceID, 0x72, string(jCmd))
	if !success {
		server.SendResponse(clientID, 0x71, 200, AdminRestoreResp{Status: "error", Message: "Device Offline"})
		return
	}

	server.SendResponse(clientID, 0x71, 200, AdminRestoreResp{Status: "ok", Message: "Restore command sent to device"})
}

func HandleRestoreInit(clientID int, payload string) {
	var req RestoreInitReq
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		server.SendResponse(clientID, 0x74, 400, `{"status": "error"}`)
		return
	}

	session, err := RestoreSvc.InitSession(req.DeviceID, req.FileUUID, req.Version)
	if err != nil {
		server.SendResponse(clientID, 0x74, 200, RestoreInitResp{Status: "error"})
		return
	}

	server.SendResponse(clientID, 0x74, 200, RestoreInitResp{
		TransferID: session.TransferID,
		FileName:   session.FileName,
		TotalSize:  session.TotalSize,
		FileHash:   session.FileHash,
		Status:     "ok",
	})
}

func HandleRestoreResume(clientID int, payload string) {
	var req RestoreResumeReq
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		server.SendResponse(clientID, 0x7A, 400, `{"status": "error"}`)
		return
	}

	session, err := RestoreSvc.ResumeSession(req.TransferID)
	if err != nil {
		server.SendResponse(clientID, 0x7A, 200, RestoreResumeResp{Status: "error"})
		return
	}

	server.SendResponse(clientID, 0x7A, 200, RestoreResumeResp{
		TransferID: session.TransferID,
		FileName:   session.FileName,
		TotalSize:  session.TotalSize,
		FileHash:   session.FileHash,
		Status:     "ok",
	})
}

func HandleRestoreChunk(clientID int, payload string) {
	var req RestoreChunkReq
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		server.SendResponse(clientID, 0x76, 400, `{"status": "error"}`)
		return
	}

	data, err := RestoreSvc.GetChunk(req.TransferID, req.Offset, req.Size)
	if err != nil {
		server.SendResponse(clientID, 0x76, 200, RestoreChunkResp{Status: "error"})
		return
	}

	server.SendResponse(clientID, 0x76, 200, RestoreChunkResp{
		Data:    hex.EncodeToString(data),
		DataLen: len(data),
		Status:  "ok",
	})
}

func HandleRestoreFinish(clientID int, payload string) {
	var req RestoreFinishReq
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		server.SendResponse(clientID, 0x78, 400, `{"status": "error"}`)
		return
	}

	if err := RestoreSvc.FinishSession(req.TransferID); err != nil {
		server.SendResponse(clientID, 0x78, 200, `{"status": "error"}`)
		return
	}

	server.SendResponse(clientID, 0x78, 200, `{"status": "ok"}`)
}

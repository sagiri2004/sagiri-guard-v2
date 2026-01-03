package controllers

import (
	"demo/network/go_server/server"
	"encoding/json"
	"fmt"
)

type AdminGetLogsReq struct {
	TargetDeviceID string      `json:"target_device_id"`
	LineCount      interface{} `json:"line_count"`
}

func HandleAdminGetLogs(sock int, payload string) {
	var req AdminGetLogsReq
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		server.SendResponse(sock, 0xD2, 400, map[string]string{"error": "Invalid Payload"})
		return
	}

	lc := 50
	if val, ok := req.LineCount.(float64); ok {
		lc = int(val)
	} else if val, ok := req.LineCount.(int); ok {
		lc = val
	}

	// Use Service
	cmd, err := AdminSvc.QueueGetLogs(req.TargetDeviceID, lc)
	if err != nil {
		server.SendResponse(sock, 0xD2, 500, map[string]string{"error": "Failed to queue command"})
		return
	}

	fmt.Printf("[Admin] Queued Command %d for %s\n", cmd.ID, req.TargetDeviceID)

	server.SendResponse(sock, 0xD2, 200, map[string]string{
		"status": "Command Queued",
		"cmd_id": fmt.Sprintf("%d", cmd.ID),
	})
}

func HandleAdminGetStoredLogs(sock int, payload string) {
	var req struct {
		TargetDeviceID string `json:"target_device_id"`
	}
	json.Unmarshal([]byte(payload), &req)

	// Use LogService
	logs, _ := LogSvc.GetRecentLogs(req.TargetDeviceID, 5)

	respBytes, _ := json.Marshal(logs)
	server.SendResponse(sock, 0xD9, 200, map[string]string{"logs": string(respBytes)})
}

func HandleAdminGetCommandHistory(sock int, payload string) {
	var req struct {
		TargetDeviceID string `json:"target_device_id"`
		Page           int    `json:"page"`
		Size           int    `json:"size"`
	}
	json.Unmarshal([]byte(payload), &req)

	if req.Page < 1 {
		req.Page = 1
	}
	if req.Size < 1 {
		req.Size = 10
	}

	// Use AdminService
	cmds, _ := AdminSvc.GetCommandHistory(req.TargetDeviceID, req.Page, req.Size)

	respBytes, _ := json.Marshal(cmds)
	server.SendResponse(sock, 0xDB, 200, map[string]string{"history": string(respBytes)})
}

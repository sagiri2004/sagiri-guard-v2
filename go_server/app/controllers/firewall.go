package controllers

import (
	"demo/network/go_server/server"
	"encoding/json"
	"fmt"
)

// Admin Request for Control
type AdminFirewallControlReq struct {
	TargetDeviceID string `json:"target_device_id"`
	Enable         bool   `json:"enable"`
	Categories     []int  `json:"categories"` // List of Category IDs
}

func HandleAdminFirewallControl(sock int, payload string) {
	fmt.Printf("[Debug] Firewall Control Payload: '%s'\n", payload)
	var req AdminFirewallControlReq
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		server.SendResponse(sock, 0xE2, 400, map[string]string{"error": "Invalid Payload"})
		return
	}

	// Use FirewallService
	err := FirewallSvc.UpdateConfig(req.TargetDeviceID, req.Enable, req.Categories)
	if err != nil {
		server.SendResponse(sock, 0xE2, 500, map[string]string{"error": "Failed to update config"})
		return
	}

	server.SendResponse(sock, 0xE2, 200, map[string]string{"status": "Config Updated"})
}

func HandleClientGetFirewallConfig(sock int, payload string) {
	var req struct {
		DeviceID string `json:"device_id"`
	}
	json.Unmarshal([]byte(payload), &req)

	if req.DeviceID == "" {
		server.SendResponse(sock, 0xE5, 400, map[string]string{"error": "Missing DeviceID"})
		return
	}

	// Use FirewallService
	resp, err := FirewallSvc.GetConfig(req.DeviceID)
	if err != nil {
		server.SendResponse(sock, 0xE5, 500, map[string]string{"error": "Internal Error"})
		return
	}

	// Send Response
	server.SendResponse(sock, 0xE5, 200, resp)
}

package controllers

import (
	"demo/network/go_server/app/dto"
	"demo/network/go_server/app/models"
	"demo/network/go_server/global"
	"demo/network/go_server/server"
	"encoding/json"
	"fmt"
)

func HandleLogin(sock int, payload string) {
	var req dto.ProtocolLoginRequest
	err := json.Unmarshal([]byte(payload), &req)
	if err != nil {
		server.SendResponse(sock, 0xA2, 400, map[string]string{"error": "Invalid JSON"})
		return
	}

	fmt.Printf("[Controller] Login Attempt: %s (Device: %s)\n", req.Username, req.DeviceID)

	var user models.User
	if result := global.DB.Where("username = ?", req.Username).First(&user); result.Error != nil {
		server.SendResponse(sock, 0xA2, 401, map[string]string{"error": "User not found"})
		return
	}

	if user.Password != req.Password {
		server.SendResponse(sock, 0xA2, 401, map[string]string{"error": "Invalid Password"})
		return
	}

	// Check Device
	var device models.Device
	if result := global.DB.Where("user_id = ? AND device_id = ?", user.ID, req.DeviceID).First(&device); result.Error != nil {
		// Device not found
		server.SendResponse(sock, 0xA2, 403, map[string]string{"error": "Device not registered"})
		return
	}

	server.SendResponse(sock, 0xA2, 200, map[string]string{"message": "Login Successful", "user_id": fmt.Sprint(user.ID)})
}

func HandleAdminLogin(sock int, payload string) {
	var req dto.ProtocolLoginRequest
	err := json.Unmarshal([]byte(payload), &req)
	if err != nil {
		server.SendResponse(sock, 0xD7, 400, map[string]string{"error": "Invalid JSON"})
		return
	}

	fmt.Printf("[Controller] Admin Login Attempt: %s\n", req.Username)

	// Hardcoded Admin Check or DB Check
	// For simplicity, let's assume admin user must be in DB with username "admin"
	// Or even simpler as per request: "check username == admin && password == admin" ??
	// The user said: "check username == admin và mật khẩu"
	// Let's check DB for the user first.

	var user models.User
	if result := global.DB.Where("username = ?", req.Username).First(&user); result.Error != nil {
		server.SendResponse(sock, 0xD7, 401, map[string]string{"error": "User not found"})
		return
	}

	if user.Password != req.Password {
		server.SendResponse(sock, 0xD7, 401, map[string]string{"error": "Invalid Password"})
		return
	}

	// Check if username is "admin" strictly if requested, or just allow any valid user?
	// User said: "check username == admin".
	if user.Username != "admin" {
		server.SendResponse(sock, 0xD7, 403, map[string]string{"error": "Not an admin user"})
		return
	}

	// No Device Check
	server.SendResponse(sock, 0xD7, 200, map[string]string{"message": "Admin Login Successful"})
}

func HandleDeviceRegister(sock int, payload string) {
	var req dto.ProtocolDeviceRequest
	err := json.Unmarshal([]byte(payload), &req)
	if err != nil {
		server.SendResponse(sock, 0xC2, 400, map[string]string{"error": "Invalid JSON"})
		return
	}

	fmt.Printf("[Controller] Device Register Attempt: %s / %s\n", req.Username, req.DeviceID)

	// Validate User (Assuming user must exist to register device, or at least provided creds - for now just username check)
	// In a real scenario, we might want to auth the user first, but here we just trust the username for registration flow as per prompt implying "auto register".
	// However, we should probably check if user exists.
	var user models.User
	if result := global.DB.Where("username = ?", req.Username).First(&user); result.Error != nil {
		server.SendResponse(sock, 0xC2, 404, map[string]string{"error": "User not found"})
		return
	}

	// Check if device already exists
	var device models.Device
	if result := global.DB.Where("device_id = ?", req.DeviceID).First(&device); result.Error == nil {
		// Already registered
		server.SendResponse(sock, 0xC2, 200, map[string]string{"message": "Device already registered", "device_id": device.DeviceID})
		return
	}

	// Register New Device
	newDevice := models.Device{
		UserID:    user.ID,
		DeviceID:  req.DeviceID,
		Name:      req.Name,
		OSName:    req.OSName,
		OSVersion: req.OSVersion,
		Hostname:  req.Hostname,
		Arch:      req.Arch,
	}

	if err := global.DB.Create(&newDevice).Error; err != nil {
		server.SendResponse(sock, 0xC2, 500, map[string]string{"error": "Failed to register device"})
		return
	}

	server.SendResponse(sock, 0xC2, 200, map[string]string{"message": "Device registered successfully", "device_id": newDevice.DeviceID})
}

package server

import (
	"demo/network/go_server/app/models"
	"demo/network/go_server/global"
	"fmt"
	"time"
)

func ProcessCommandQueue(deviceID string) {
	fmt.Printf("[Queue] Checking pending commands for %s...\n", deviceID)

	var commands []models.Command
	// Find PENDING commands
	// Assuming 0xD3 is GET_LOGS.
	result := global.DB.Where("device_id = ? AND status = ?", deviceID, models.StatusPending).Find(&commands)
	if result.Error != nil {
		fmt.Printf("[Queue] Error querying DB: %v\n", result.Error)
		return
	}

	if len(commands) == 0 {
		fmt.Println("[Queue] No pending commands.")
		return
	}

	for _, cmd := range commands {
		fmt.Printf("[Queue] Process Command %d (Type %d)\n", cmd.ID, cmd.CommandType)

		// Attempt Send
		// Use wrapper's SendToDevice (which is in same package, so just SendToDevice)
		success := SendToDevice(deviceID, cmd.CommandType, cmd.Payload)

		if success {
			fmt.Printf("[Queue] Command %d sent successfully.\n", cmd.ID)
			cmd.Status = models.StatusSent
		} else {
			fmt.Printf("[Queue] Failed to send Command %d (Device offline?).\n", cmd.ID)
			// Keep PENDING or status FAILED?
			// If we are processing queue *on connect*, failure means something weird (socket lost immediately).
			// Let's keep PENDING or mark FAILED.
			// If we leave PENDING, it retries next connect.
			// But if SendToDevice returns false, it means device ID not found in C list.
			// But we effectively just got a callback saying "I am here".
			// Maybe race condition or ID mismatch.
			// Let's mark FAILED to prevent infinite loop if broken.
			// cmd.Status = models.StatusFailed
		}
		cmd.UpdatedAt = time.Now()
		global.DB.Save(&cmd)
	}
}

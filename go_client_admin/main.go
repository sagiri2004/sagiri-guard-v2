package main

/*
#cgo CFLAGS: -I../
#cgo LDFLAGS: ${SRCDIR}/../client_core.o ${SRCDIR}/../protocol.o
#include <stdlib.h>
#include "../client/core.h"
#include "../protocol/protocol.h"
*/
import "C"

import (
	"bufio"
	"demo/network/go_client_admin/config"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
	"unsafe"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("--- GO ADMIN CLIENT ---")

	// Load App Config
	if err := config.LoadAppConfig("config.yml"); err != nil {
		fmt.Printf("[Error] Failed to load config.yml: %v\n", err)
		return
	}
	appCfg := config.GlobalAppConfig

	// Init Client Context
	cHost := C.CString(appCfg.Admin.ServerHost)
	defer C.free(unsafe.Pointer(cHost))

	ctx := C.client_init(cHost, C.int(appCfg.Admin.ServerPort), C.int(appCfg.Admin.APIPort))
	defer C.client_close(ctx)

	// Login
	fmt.Print("Enter Admin Username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("Enter Admin Password: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	// Dummy device ID for admin (NOT used in new flow but kept for cleanup if needed)
	cUser := C.CString(username)
	cPass := C.CString(password)
	cDev := C.CString("ADMIN_CONSOLE")
	defer C.free(unsafe.Pointer(cUser))
	defer C.free(unsafe.Pointer(cPass))
	defer C.free(unsafe.Pointer(cDev))

	for {
		fmt.Println("[Info] Attempting Login...")
		success := C.client_admin_login(ctx, cUser, cPass)

		if success == 1 {
			fmt.Println("Login Successful!")
			break
		}
		fmt.Println("[Error] Login Failed! Retrying in 5s...")
		time.Sleep(5 * time.Second)
	}

	// Connect Notification (Optional for Admin but good for state)
	C.client_connect_notification(ctx, cDev)

	for {
		fmt.Println("\n1. List Online Users")
		fmt.Println("2. Trigger Get Logs (Async)")
		fmt.Println("3. View Stored Logs")
		fmt.Println("4. View Command History")
		fmt.Println("5. Firewall Control")
		fmt.Println("6. Browse File Tree")
		fmt.Println("7. Restore File")
		fmt.Println("8. Exit")
		fmt.Print("Choice: ")

		choiceStr, _ := reader.ReadString('\n')
		var choice int
		fmt.Sscanf(choiceStr, "%d", &choice)

		switch choice {
		case 1:
			var buffer [1024]C.char
			res := C.client_get_online_users(ctx, &buffer[0])
			if res == 1 {
				goStr := C.GoString(&buffer[0])
				fmt.Printf("Online Users: %s\n", goStr)
			} else {
				fmt.Println("Failed to fetch user list.")
			}

		case 2:
			fmt.Print("Enter Target Device ID: ")
			target, _ := reader.ReadString('\n')
			target = strings.TrimSpace(target)

			fmt.Print("Enter Line Count (default 50): ")
			lcStr, _ := reader.ReadString('\n')
			lcStr = strings.TrimSpace(lcStr)
			lineCount := 50
			if lcStr != "" {
				fmt.Sscanf(lcStr, "%d", &lineCount)
			}

			// Payload: {"target_device_id": "...", "line_count": 50}
			payload := map[string]interface{}{
				"target_device_id": target,
				"line_count":       lineCount,
			}
			jsonBytes, _ := json.Marshal(payload)
			cPayload := C.CString(string(jsonBytes))

			var buffer [1024]C.char
			res := C.client_admin_get_logs(ctx, cPayload, &buffer[0])
			C.free(unsafe.Pointer(cPayload))

			if res == 1 {
				fmt.Printf("Response: %s\n", C.GoString(&buffer[0]))
			} else {
				fmt.Println("Request Failed.")
			}

		case 3:
			// View Stored Logs
			fmt.Print("Enter Target Device ID (empty for all): ")
			target, _ := reader.ReadString('\n')
			target = strings.TrimSpace(target)

			payload := map[string]string{"target_device_id": target}
			jsonBytes, _ := json.Marshal(payload)
			cPayload := C.CString(string(jsonBytes))

			var buffer [65535]C.char // Larger buffer for logs
			res := C.client_admin_view_logs(ctx, cPayload, &buffer[0])
			C.free(unsafe.Pointer(cPayload))

			if res == 1 {
				fmt.Printf("Stored Logs:\n%s\n", C.GoString(&buffer[0]))
			} else {
				fmt.Println("Failed to fetch stored logs.")
			}

		case 4:
			// View Command History
			fmt.Print("Enter Target Device ID (empty for all): ")
			target, _ := reader.ReadString('\n')
			target = strings.TrimSpace(target)

			fmt.Print("Enter Page (default 1): ")
			pStr, _ := reader.ReadString('\n')
			pStr = strings.TrimSpace(pStr)
			page := 1
			if pStr != "" {
				fmt.Sscanf(pStr, "%d", &page)
			}

			fmt.Print("Enter Size (default 10): ")
			sStr, _ := reader.ReadString('\n')
			sStr = strings.TrimSpace(sStr)
			size := 10
			if sStr != "" {
				fmt.Sscanf(sStr, "%d", &size)
			}

			payload := map[string]interface{}{
				"target_device_id": target,
				"page":             page,
				"size":             size,
			}
			jsonBytes, _ := json.Marshal(payload)
			cPayload := C.CString(string(jsonBytes))

			var buffer [65535]C.char
			res := C.client_admin_get_history(ctx, cPayload, &buffer[0])
			C.free(unsafe.Pointer(cPayload))

			if res == 1 {
				fmt.Printf("Command History:\n%s\n", C.GoString(&buffer[0]))
			} else {
				fmt.Println("Failed to fetch command history.")
			}

		case 5:
			// Firewall Control
			fmt.Print("Enter Target Device ID: ")
			target, _ := reader.ReadString('\n')
			target = strings.TrimSpace(target)

			fmt.Print("Enable Firewall (y/n): ")
			enableStr, _ := reader.ReadString('\n')
			enableStr = strings.TrimSpace(enableStr)
			enable := (strings.ToLower(enableStr) == "y")

			var categories []int
			if enable {
				fmt.Println("Available Categories:")
				fmt.Println("1: Social Media")
				fmt.Println("2: AI")
				fmt.Println("3: Gaming")
				fmt.Println("4: Shopping")
				fmt.Println("5: News")
				fmt.Println("6: Entertainment")
				fmt.Println("7: Adult")
				fmt.Print("Enter Category IDs (comma separated, e.g. 1,3): ")
				catStr, _ := reader.ReadString('\n')
				catStr = strings.TrimSpace(catStr)
				parts := strings.Split(catStr, ",")
				for _, p := range parts {
					// Simple Atoi
					var cid int
					fmt.Sscanf(strings.TrimSpace(p), "%d", &cid)
					if cid > 0 {
						categories = append(categories, cid)
					}
				}
			}

			fmt.Printf("Updating Config... (Enable=%v, Cats=%v)\n", enable, categories)

			payload := map[string]interface{}{
				"target_device_id": target,
				"enable":           enable,
				"categories":       categories,
			}
			jsonBytes, _ := json.Marshal(payload)

			cPayload := C.CString(string(jsonBytes))
			var buffer [1024]C.char
			res := C.client_admin_firewall_control(ctx, cPayload, &buffer[0])
			C.free(unsafe.Pointer(cPayload))

			if res == 1 {
				fmt.Println("Success: Config Updated.")
			} else {
				fmt.Println("Request Failed.")
				fmt.Printf("Error Details: %s\n", C.GoString(&buffer[0]))
			}

		case 6:
			// Browse File Tree
			fmt.Print("Enter Target Device ID: ")
			target, _ := reader.ReadString('\n')
			target = strings.TrimSpace(target)

			fmt.Print("Enter Parent ID (empty for root): ")
			parentStr, _ := reader.ReadString('\n')
			parentStr = strings.TrimSpace(parentStr)
			var parentID *uint
			if parentStr != "" {
				var pid uint
				fmt.Sscanf(parentStr, "%d", &pid)
				parentID = &pid
			}

			fmt.Print("Show Deleted Files? (y/n): ")
			delStr, _ := reader.ReadString('\n')
			delStr = strings.TrimSpace(delStr)
			showDeleted := (strings.ToLower(delStr) == "y")

			payload := map[string]interface{}{
				"device_id":    target,
				"parent_id":    parentID,
				"show_deleted": showDeleted,
				"page":         1,
				"size":         50,
			}
			jsonBytes, _ := json.Marshal(payload)
			cPayload := C.CString(string(jsonBytes))

			var buffer [65535]C.char
			res := C.client_admin_get_file_tree(ctx, cPayload, &buffer[0])
			C.free(unsafe.Pointer(cPayload))

			if res == 1 {
				fmt.Printf("File Tree:\n%s\n", C.GoString(&buffer[0]))
			} else {
				fmt.Println("Failed to fetch file tree.")
			}

		case 7:
			// Restore
			fmt.Print("Enter Target Device ID: ")
			target, _ := reader.ReadString('\n')
			target = strings.TrimSpace(target)

			fmt.Print("Enter File UUID: ")
			fileUUID, _ := reader.ReadString('\n')
			fileUUID = strings.TrimSpace(fileUUID)

			fmt.Print("Enter Version (0 for latest): ")
			vStr, _ := reader.ReadString('\n')
			vStr = strings.TrimSpace(vStr)
			version := 0
			if vStr != "" {
				fmt.Sscanf(vStr, "%d", &version)
			}

			payload := map[string]interface{}{
				"device_id": target,
				"file_uuid": fileUUID,
				"version":   version,
			}
			jsonBytes, _ := json.Marshal(payload)
			cPayload := C.CString(string(jsonBytes))

			var buffer [1024]C.char
			res := C.client_admin_restore(ctx, cPayload, &buffer[0])
			C.free(unsafe.Pointer(cPayload))

			if res == 1 {
				fmt.Printf("Response: %s\n", C.GoString(&buffer[0]))
			} else {
				fmt.Println("Restore Trigger Failed.")
			}

		case 8:
			return
		}
	}
}

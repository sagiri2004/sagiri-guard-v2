package main

/*
#cgo CFLAGS: -I../
#cgo LDFLAGS: ${SRCDIR}/../client_core.o ${SRCDIR}/../protocol.o
#include <stdlib.h>
#include <string.h>
#include "../client/core.h"
#include "../protocol/protocol.h"

// Shim function to pass to C
extern void on_message_shim(const char *msg);
*/
import "C"

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unsafe"

	"demo/network/go_client/internal/auth"
	"demo/network/go_client/internal/config"
	dbpkg "demo/network/go_client/internal/db"
	"demo/network/go_client/internal/device"
	"demo/network/go_client/internal/filesync"
	"demo/network/go_client/internal/firewall"
	"demo/network/go_client/internal/logger"
	"demo/network/go_client/internal/monitor"
)

var GlobalClientCtx *C.ClientContext

//export goOnMessage
func goOnMessage(msg *C.char) {
	goStr := C.GoString(msg)
	fmt.Printf("\n[Notification] Received: %s\n", goStr)

	// Check for "GET_LOGS" command
	// Parsing JSON payload from server command?
	// The server sends: `msgType` + `payload`.
	// For now, let's assume the payload IS the JSON string like `{"line_count": 50}` or `GET_LOGS {"line_count": 50}`?
	// Protocol says msg is char*.
	// If the server sends `MSG_SERVER_COMMAND_GETLOG` (0xD3) with payload `{"line_count": 50}`,
	// The C layer might pass the raw payload string to this callback if configured.
	// Assume `goStr` is the payload.

	if strings.Contains(goStr, "GET_LOGS") || strings.Contains(goStr, "line_count") {
		fmt.Println("[Auto] Triggering Log Upload...")

		// Parse line count
		lineCount := 50 // Default
		var cmdPayload map[string]interface{}
		// Try to find JSON in the string
		start := strings.Index(goStr, "{")
		if start != -1 {
			json.Unmarshal([]byte(goStr[start:]), &cmdPayload)
			if lc, ok := cmdPayload["line_count"].(float64); ok {
				lineCount = int(lc)
			}
		}

		// Read Logs
		lines, err := logger.Tail(lineCount)
		var content string
		if err != nil {
			content = fmt.Sprintf("Error reading logs: %v", err)
		} else {
			content = strings.Join(lines, "\n")
		}

		// Wrap in JSON with Device ID (We need to load it)
		devCfg, _ := config.LoadDeviceConfig()

		payloadMap := map[string]string{
			"device_id": devCfg.DeviceID,
			"content":   content,
		}
		jsonBytes, _ := json.Marshal(payloadMap)
		logs := string(jsonBytes)

		cLogs := C.CString(logs)
		var resp [1024]C.char
		// Use GlobalClientCtx
		C.client_upload_logs(GlobalClientCtx, cLogs, &resp[0])
		C.free(unsafe.Pointer(cLogs))

		fmt.Printf("[Auto] Upload Response: %s\n", C.GoString(&resp[0]))
		fmt.Print("Choice: ") // Prompt restore
	}
	// 2. FIREWALL_UPDATE
	if strings.Contains(goStr, "FIREWALL_UPDATE") {
		fmt.Println("[Auto] Triggering Firewall Config Refresh...")
		// We need Context and DeviceID.
		// GlobalClientCtx is available.
		devCfg, _ := config.LoadDeviceConfig()
		if devCfg.DeviceID == "" {
			fmt.Println("[Error] Cannot refresh firewall: No Device ID.")
			return
		}

		cDevID := C.CString(devCfg.DeviceID)
		var resp [4096]C.char // Buffer for config
		// Call C function to fetch config
		// Wait.. `client_get_firewall_config` needs to be exported or available?
		// It IS available in `core.c` but `main.go` needs to call it via C.
		// `client_get_firewall_config` signature: (ctx, json_payload, resp_buffer)
		// Payload: `{"device_id": "..."}`

		payload := fmt.Sprintf(`{"device_id": "%s"}`, devCfg.DeviceID)
		cPayload := C.CString(payload)

		fmt.Println("[Firewall] Fetching new config...")
		res := C.client_get_firewall_config(GlobalClientCtx, cPayload, &resp[0])

		C.free(unsafe.Pointer(cDevID))
		C.free(unsafe.Pointer(cPayload))

		if res == 1 {
			respStr := C.GoString(&resp[0])
			fmt.Printf("[Firewall] Config Received: %s\n", respStr)

			// Parse JSON
			var fwResp struct {
				Enabled bool     `json:"enabled"`
				Domains []string `json:"domains"`
			}
			if err := json.Unmarshal([]byte(respStr), &fwResp); err != nil {
				fmt.Printf("[Firewall] Failed to parse config: %v\n", err)
			} else {
				// Update Global State
				config.UpdateFirewallConfig(fwResp.Enabled, fwResp.Domains)

				// Apply to Hosts
				status := config.GlobalFirewall
				hm := firewall.GetHostsManager()

				if status.Enabled {
					hm.SetDomains(status.Domains)
					hm.SetEnabled(true)
					fmt.Printf("[Firewall] Applied: Enabled (%d domains)\n", len(status.Domains))
				} else {
					hm.SetEnabled(false)
					fmt.Println("[Firewall] Applied: Disabled")
				}
			}
		} else {
			fmt.Println("[Firewall] Failed to fetch config.")
		}
		fmt.Print("Choice: ")
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("--- GO CLIENT START ---")

	// Load App Config
	if err := config.LoadAppConfig("config.yml"); err != nil {
		fmt.Printf("[Error] Failed to load config.yml: %v\n", err)
		return
	}
	appCfg := config.GlobalAppConfig

	// Init Logger
	if err := logger.Init(appCfg.Client.LogDir); err != nil {
		fmt.Printf("[Warning] Failed to init logger: %v\n", err)
	}
	logger.Info("Client Started")

	// Init Client Context with Config
	cHost := C.CString(appCfg.Client.ServerHost)
	defer C.free(unsafe.Pointer(cHost))

	ctx := C.client_init(cHost, C.int(appCfg.Client.ServerPort), C.int(appCfg.Client.APIPort))
	defer C.client_close(ctx)
	GlobalClientCtx = ctx

	// Init File Sync Context
	filesync.SetClientContext(unsafe.Pointer(ctx))

	// Start File Sync Worker
	syncWorker := filesync.NewSyncWorker()
	syncWorker.Start()
	fmt.Println("[Init] Sync Worker Started (10s interval)")
	defer syncWorker.Stop()

	// Register Callback
	C.client_set_on_message(ctx, C.MessageCallback(C.on_message_shim))

	// Load Device Config
	devCfg, err := config.LoadDeviceConfig()
	if err != nil {
		devCfg = &config.DeviceConfig{}
	}

	// Inputs
	fmt.Print("Enter Username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("Enter Password: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	// Device Registration Flow
	if devCfg.DeviceID == "" {
		fmt.Println("[Info] Device not registered. Gathering System Info...")

		sysInfo, err := device.GetSystemInfo()
		if err != nil {
			fmt.Printf("[Error] Failed to get system info: %v\n", err)
		}

		regPayload := auth.Credentials{
			Username:  username,
			DeviceID:  sysInfo.UUID,
			Name:      sysInfo.Hostname,
			OSName:    sysInfo.OSName,
			OSVersion: sysInfo.OSVersion,
			Hostname:  sysInfo.Hostname,
			Arch:      sysInfo.Arch,
		}

		jsonData, _ := json.Marshal(regPayload)
		cJSON := C.CString(string(jsonData))
		var respBuffer [1024]C.char

		fmt.Println("[Info] Sending Device Registration Request...")
		// Now passing ctx
		success := C.client_register_device(ctx, cJSON, &respBuffer[0])
		C.free(unsafe.Pointer(cJSON))

		if success == 1 {
			respStr := C.GoString(&respBuffer[0])
			fmt.Printf("[Success] Registration Response: %s\n", respStr)

			var respMap map[string]interface{}
			json.Unmarshal([]byte(respStr), &respMap)
			if did, ok := respMap["device_id"].(string); ok {
				devCfg.DeviceID = did
				config.SaveDeviceConfig(devCfg)
				fmt.Println("[Info] Device ID saved.")
			}
		} else {
			fmt.Println("[Error] Device Registration Failed! Exiting.")
			return
		}
	} else {
		fmt.Printf("[Info] Found registered Device ID: %s\n", devCfg.DeviceID)
	}

	cUser := C.CString(username)
	cPass := C.CString(password)
	cDev := C.CString(devCfg.DeviceID)
	defer C.free(unsafe.Pointer(cUser))
	defer C.free(unsafe.Pointer(cPass))
	defer C.free(unsafe.Pointer(cDev))

	fmt.Println("[Info] Attempting Login...")
	success := C.client_login(ctx, cUser, cPass, cDev)

	if success == 0 {
		fmt.Println("Login Failed! Exiting.")
		return
	}
	fmt.Println("Login Successful!")

	// --- FETCH FIREWALL CONFIG ON STARTUP ---
	fmt.Println("[Init] Fetching Firewall Config...")
	cDevID := C.CString(devCfg.DeviceID)
	payload := fmt.Sprintf(`{"device_id": "%s"}`, devCfg.DeviceID)
	cPayload := C.CString(payload)
	var resp [4096]C.char

	res := C.client_get_firewall_config(ctx, cPayload, &resp[0])
	C.free(unsafe.Pointer(cDevID))
	C.free(unsafe.Pointer(cPayload))

	if res == 1 {
		respStr := C.GoString(&resp[0])
		fmt.Printf("[Init] Firewall Config: %s\n", respStr)

		var fwResp struct {
			Enabled bool     `json:"enabled"`
			Domains []string `json:"domains"`
		}
		if err := json.Unmarshal([]byte(respStr), &fwResp); err == nil {
			config.UpdateFirewallConfig(fwResp.Enabled, fwResp.Domains)
			hm := firewall.GetHostsManager()
			if fwResp.Enabled {
				hm.SetDomains(fwResp.Domains)
				hm.SetEnabled(true)
				fmt.Printf("[Init] Firewall Applied: Enabled (%d domains)\n", len(fwResp.Domains))
			} else {
				hm.SetEnabled(false)
				fmt.Println("[Init] Firewall Applied: Disabled")
			}
		}
	} else {
		fmt.Println("[Init] Failed to fetch firewall config.")
	}
	// ----------------------------------------

	// --- FILE MONITOR & LOCAL DB ---
	// 1. Init Local DB
	dbPath := appCfg.Client.LogDir + "/client.db"
	if _, err := dbpkg.Init(dbPath); err != nil {
		fmt.Printf("[Warning] Failed to init local DB: %v\n", err)
	} else {
		fmt.Println("[Init] Local DB Initialized.")
	}

	// 2. Start File Monitor
	// If Configured Dirs exist, watch them. If empty, use default "./monitor_test" for demo
	watchDirs := appCfg.Client.MonitoredDirs
	if len(watchDirs) == 0 {
		_ = os.MkdirAll("./monitor_test", 0755)
		watchDirs = []string{"./monitor_test"}
	}

	fm, err := monitor.NewFileMonitor(watchDirs)
	if err != nil {
		fmt.Printf("[Error] Failed to start FileMonitor: %v\n", err)
	} else {
		evtChan := fm.MonitorFiles() // This starts the monitoring in a goroutine internally
		fmt.Printf("[Info] File Monitor Started on %v\n", watchDirs)
		defer fm.Close()
		go func() {
			for evt := range evtChan {
				// Optional: Send to Server or Log to Console
				// For now just logging debug if needed
				_ = evt
			}
		}()
	}
	// -------------------------------

	C.client_connect_notification(ctx, cDev)

	// Block forever
	select {}
}

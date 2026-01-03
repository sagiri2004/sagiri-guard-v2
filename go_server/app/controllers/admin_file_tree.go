package controllers

import (
	"demo/network/go_server/app/services"
	"demo/network/go_server/server"
	"encoding/json"
	"fmt"
)

var directoryTreeSvc *services.DirectoryTreeService

func SetDirectoryTreeService(svc *services.DirectoryTreeService) {
	directoryTreeSvc = svc
}

func HandleAdminGetFileTree(adminSock int, payload string) {
	fmt.Printf("[Controller] Admin Get File Tree Request\n")

	var req services.TreeQuery
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		server.SendResponse(adminSock, 0xE9, 400, `{"error": "Invalid Payload"}`)
		return
	}

	if req.DeviceID == "" {
		server.SendResponse(adminSock, 0xE9, 400, `{"error": "Missing Device ID"}`)
		return
	}

	resp, err := directoryTreeSvc.GetTree(req)
	if err != nil {
		fmt.Printf("[Error] Failed to get tree: %v\n", err)
		server.SendResponse(adminSock, 0xE9, 500, `{"error": "Internal Server Error"}`)
		return
	}

	server.SendResponse(adminSock, 0xE9, 200, resp)
}

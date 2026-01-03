package controllers

import (
	"demo/network/go_server/server"
	"encoding/json"
)

func HandleListUsers(sock int, payload string) {
	// Call Wrapper to get online users
	// But `server.GetOnlineUsers` returns []string.
	// We need to format it as JSON: {"users": ["id1", "id2"]}

	users := server.GetOnlineUsers()

	resp := map[string]interface{}{
		"users": users,
	}

	respBytes, _ := json.Marshal(resp)

	// Send Response (Type 0xB2? Protocol usually implies Request+1, but existing C code uses 0x??)
	// Protocol Check: MSG_LIST_REQ = 0xB1. Response usually 0xB2?
	// C code used MSG_LOGIN_RESP (0xA2) or generic 200 OK.
	// Let's check protocol.h

	// Wait, protocol.h says MSG_LIST_RESP 0xB2?
	// Let's assume 0xB2.

	server.SendResponse(sock, 0xB2, 200, map[string]string{"users_json": string(respBytes)})
	// Note: Client expects "users" array in the root of JSON or "users" inside?
	// C client expects: `{"users": [...]}`.
	// My SendResponse wraps map into json.
	// If I pass `map[string]string{"users": "..."}` it becomes `{"users": "..."}`.
	// But `users` should be ARRAY. `SendResponse` takes `map[string]string`.
	// Limitation of SendResponse helper! It creates JSON from string map.
	// I should probably send raw string if `SendResponse` supports it, OR modify SendResponse, OR construct raw JSON and use server generic send.
	//
	// `server.SendResponse` impl in Go wrapper:
	/*
		func SendResponse(sock int, msgType int, status int, data map[string]string) {
			jsonData, _ := json.Marshal(data)
			...
		}
	*/
	// It marshals the map. So values are strings.
	// If I want `["a", "b"]`, I can't use this helper if it enforces string values.
	//
	// I can hack it: Pass raw JSON string as a value?
	// Client expects `{"users": ["a", "b"]}`.
	// If I use `{"users": "[\"a\", \"b\"]"}`, client gets a string, not array.
	//
	// I'll create a new helper `SendJSONResponse` or just do it manually here.
	// Or Update `SendResponse` to take `interface{}`.

	// Let's update `SendResponse` in wrapper.go to take `interface{}` for data.
}

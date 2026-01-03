package server

/*
#cgo CFLAGS: -I../../
#cgo LDFLAGS: ${SRCDIR}/../../server_core.o ${SRCDIR}/../../protocol.o ${SRCDIR}/../../shim.o
#include <stdlib.h>
#include "../../server/core.h"
#include "../../server/shim.h"
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"unsafe"
)

var GlobalCtx *C.ServerContext

// Router Map: MsgType -> ControllerFunc
var Router = make(map[int]func(sock int, payload string))

// Define Message Names
var MsgNames = map[int]string{
	0xA1: "MSG_LOGIN_REQ",
	0xB1: "MSG_LIST_REQ",
	0xC1: "MSG_DEVICE_REQ",
	0xD1: "MSG_ADMIN_COMMAND_GETLOGS_REQ",
	0xD4: "MSG_CLIENT_COMMAND_GETLOG_REQ",
	0xD6: "MSG_ADMIN_LOGIN_REQ",
	0xD8: "MSG_ADMIN_GET_STORED_LOGS_REQ",
	0xDA: "MSG_ADMIN_GET_COMMAND_HISTORY_REQ",
	0xDB: "MSG_ADMIN_GET_COMMAND_HISTORY_RESP",
	0xE1: "MSG_ADMIN_FIREWALL_CONTROL_REQ",
	0xE4: "MSG_CLIENT_GET_FIREWALL_CONFIG_REQ",
	0xE6: "MSG_CLIENT_FILE_SYNC_REQ",
	0xE8: "MSG_ADMIN_GET_FILE_TREE_REQ",
	0xF1: "MSG_BACKUP_INIT_REQ",
	0xF3: "MSG_BACKUP_CHUNK_REQ",
	0xF5: "MSG_BACKUP_FINISH_REQ",
	0xF7: "MSG_BACKUP_CANCEL_REQ",
	0xF8: "MSG_BACKUP_RESUME_REQ",
	0x70: "MSG_ADMIN_RESTORE_REQ",
	0x73: "MSG_RESTORE_INIT_REQ",
	0x75: "MSG_RESTORE_CHUNK_REQ",
	0x77: "MSG_RESTORE_FINISH_REQ",
}

//export goRequestHandler
func goRequestHandler(sock C.int, msgType C.int, payload *C.char) {
	name, ok := MsgNames[int(msgType)]
	if !ok {
		name = fmt.Sprintf("UNKNOWN(0x%X)", int(msgType))
	}
	fmt.Printf("[Go] Received Request Type: %s\n", name)

	goStr := C.GoString(payload)

	// Routing Logic
	if handler, ok := Router[int(msgType)]; ok {
		handler(int(sock), goStr)
	} else {
		// Default 400
		msg := C.CString("{\"error\": \"Route Not Found\"}")
		C.server_send_response(sock, 0, 400, msg) // 0 type means generic error resp
		C.free(unsafe.Pointer(msg))
	}
}

//export goClientConnect
func goClientConnect(deviceID *C.char) {
	id := C.GoString(deviceID)
	fmt.Printf("[Go] Client Connected: %s\n", id)

	// Trigger Command Queue Processing for this Device
	ProcessCommandQueue(id)
}

func Init(port, apiPort int) {
	GlobalCtx = C.server_init(C.int(port), C.int(apiPort))
}

func SetHandler(handler func(sock C.int, msgType C.int, payload *C.char)) {
	// Register Request Handler Shim
	C.server_set_handler(GlobalCtx, C.RequestHandler(C.request_handler_shim))
	// Register Connect Callback Shim
	C.server_set_on_connect(GlobalCtx, C.ClientConnectCallback(C.client_connect_shim))
}

var GoRequestHandler = goRequestHandler // Export for reference? Not needed if Shim calls raw export.

func Start() {
	C.server_start(GlobalCtx)
}

func InitAndStart(port, apiPort int) {
	Init(port, apiPort)
	SetHandler(nil)
	Start()
}

func SendToDevice(deviceID string, msgType int, payload string) bool {
	cID := C.CString(deviceID)
	cPayload := C.CString(payload)
	defer C.free(unsafe.Pointer(cID))
	defer C.free(unsafe.Pointer(cPayload))

	res := C.server_send_to_device(GlobalCtx, cID, C.uint8_t(msgType), cPayload)
	return res == 1
}

func SendResponse(sock int, msgType int, status int, body interface{}) {
	bytes, _ := json.Marshal(body)
	cArgs := C.CString(string(bytes))
	defer C.free(unsafe.Pointer(cArgs))

	// User said "message type representing each router".
	// Let's assume response type is RequestType + 1 (simple convention) or passed in.
	C.server_send_response(C.int(sock), C.int(msgType), C.int(status), cArgs)
}

func GetOnlineUsers() []string {
	if GlobalCtx == nil {
		return []string{}
	}
	var cIds **C.char
	count := C.server_get_online_users(GlobalCtx, &cIds)
	if count == 0 {
		return []string{}
	}

	// cIds is char** (array of strings)
	// We need to free the array itself and call needed free logic if needed
	// BUT C code says "Caller must free values", core.c uses `strdup`.
	// So we need to:
	// 1. Iterate array
	// 2. Convert to Go string
	// 3. Free C string
	// 4. Free array

	defer C.free(unsafe.Pointer(cIds))

	// Manually iterate pointer arithmetic to avoid unsafe.Slice issues if any
	// cIds is **char.
	// To get *char at index i: *(cIds + i)

	// We cast cIds to a large array pointer to index it, which is a common trick
	// OR use uintptr arithmetic.

	// Let's use the slice trick but be very careful or use a helper C function?
	// But unsafe.Slice should work for **char -> []*char.

	// Alternative:
	ptr := unsafe.Pointer(cIds) // *C.char* (effectively)
	sizeOfPtr := unsafe.Sizeof(uintptr(0))

	res := make([]string, count)
	for i := 0; i < int(count); i++ {
		// p is address of the i-th pointer
		p := (*(*C.char))(unsafe.Pointer(uintptr(ptr) + uintptr(i)*sizeOfPtr))
		res[i] = C.GoString(*p)
		C.free(unsafe.Pointer(*p))
	}
	return res
}

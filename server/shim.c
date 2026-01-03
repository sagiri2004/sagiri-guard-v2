#include "shim.h"

// Forward declarations of exported Go functions
extern void goRequestHandler(int sock, int msg_type, char *payload);
extern void goClientConnect(char *device_id);

void request_handler_shim(int sock, int msg_type, char *payload) {
    goRequestHandler(sock, msg_type, payload);
}

void client_connect_shim(char *device_id) {
    goClientConnect(device_id);
}

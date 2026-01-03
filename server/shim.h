#ifndef SHIM_H
#define SHIM_H

#include "core.h"

void request_handler_shim(int sock, int msg_type, char *payload);
void client_connect_shim(char *device_id);

#endif

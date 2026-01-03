#include <stdio.h>

extern void goOnMessage(char *msg);

void on_message_shim(const char *msg) {
    if (msg) goOnMessage((char*)msg);
}

#include <stdio.h>
#include <unistd.h>
#include "core.h"

int main() {
    printf("--- SERVER STARTING ---\n");
    ServerContext *ctx = server_init(8080, 8081);
    
    server_start(ctx);
    
    printf("Server running. Press Enter to stop...\n");
    getchar();
    
    server_stop(ctx);
    printf("Server stopped.\n");
    return 0;
}

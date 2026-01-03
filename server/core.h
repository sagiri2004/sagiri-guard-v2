#ifndef SERVER_CORE_H
#define SERVER_CORE_H

#include <pthread.h>
#include "../protocol/protocol.h"

typedef struct client_node {
    char device_id[64];
    int socket;
    struct client_node *next;
} client_node_t;

// Callback for API Requests (Socket, Type, Payload)
typedef void (*RequestHandler)(int socket, int type, char *payload);
typedef void (*ClientConnectCallback)(char *device_id);

typedef struct {
    int port;
    int api_port;
    int active_clients;
    client_node_t *head;
    pthread_mutex_t lock;
    int running; 
    RequestHandler handler; // Logic Delegate
    ClientConnectCallback on_connect;
} ServerContext;

// Initialize server context
ServerContext* server_init(int port, int api_port);

// Set logic handler
void server_set_handler(ServerContext *ctx, RequestHandler handler);
void server_set_on_connect(ServerContext *ctx, ClientConnectCallback cb);

// Start server threads (Non-blocking: returns after spawning threads)
void server_start(ServerContext *ctx);

// Stop server (Cleanup)
void server_stop(ServerContext *ctx);

// --- CGo Helpers ---

// Get online user IDs (Caller must free *ids)
// Returns count of users
int server_get_online_users(ServerContext *ctx, char ***ids);

// Broadcast message to all clients
void server_broadcast(ServerContext *ctx, char *message);

// Send message to specific client
int server_send_unicast(ServerContext *ctx, char *client_id, char *message);
void broadcast_message(ServerContext *ctx, char *sender_id, char *message);
int server_send_to_device(ServerContext *ctx, char *target_device_id, uint8_t type, char *payload);

// Send API Response (Header + JSON) and Close Socket
void server_send_response(int sock, int type, int status, char *message);

#endif

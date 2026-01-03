#ifndef CLIENT_CORE_H
#define CLIENT_CORE_H

#include "../protocol/protocol.h"
#include <pthread.h>

typedef struct {
    int notification_sock;
    int running;
    MessageCallback on_message;
    // Config
    char server_host[256];
    int server_port;
    int api_port;
    char device_id[256];
} ClientContext;

// Initialize Client Context
ClientContext* client_init(char *host, int port, int api_port);

// Login via API (Blocking, returns 1 on success, 0 on fail)
int client_login(ClientContext *ctx, char *username, char *password, char *device_id);
int client_register_device(ClientContext *ctx, char *json_payload, char *response_buffer);
// Connect to Notification Socket and start listener
void client_connect_notification(ClientContext *ctx, char *device_id);

// Admin Login (No device id required)
int client_admin_login(ClientContext *ctx, char *username, char *password);

// Stop Client
void client_close(ClientContext *ctx);

// Get list of online users via API (Returns 1 on success, stores JSON in buffer)
int client_get_online_users(ClientContext *ctx, char *json_buffer);

// --- CGo Helpers ---

// Register callback for incoming messages
void client_set_on_message(ClientContext *ctx, MessageCallback cb);

// Send a simple message to Notification Server (Async, no wait for response)
void client_send_message(ClientContext *ctx, char *message);

// Send a message and wait for response on the same socket (Blocking)
// WARNING: This may conflict if background listener is running!
int client_send_and_wait(ClientContext *ctx, char *message, char *response_buffer, int buffer_size);

int client_admin_get_logs(ClientContext *ctx, char *json_payload, char *response_buffer);
int client_admin_view_logs(ClientContext *ctx, char *json_payload, char *response_buffer);
int client_admin_get_history(ClientContext *ctx, char *json_payload, char *response_buffer);
int client_upload_logs(ClientContext *ctx, char *logs_payload, char *response_buffer);
int client_get_firewall_config(ClientContext *ctx, char *json_payload, char *response_buffer);
int client_admin_firewall_control(ClientContext *ctx, char *json_payload, char *response_buffer);
int client_file_sync(ClientContext *ctx, char *json_payload, char *response_buffer);

// Browse persistent directory tree on server
int client_admin_get_file_tree(ClientContext *ctx, char *json_payload, char *response_buffer);
int client_admin_restore(ClientContext *ctx, char *json_payload, char *response_buffer);

// Backup operations
int client_backup_init(ClientContext *ctx, char *json_payload, char *response_buffer);
int client_backup_chunk(ClientContext *ctx, char *json_payload, char *response_buffer);
int client_backup_finish(ClientContext *ctx, char *json_payload, char *response_buffer);
int client_backup_cancel(ClientContext *ctx, char *json_payload, char *response_buffer);
int client_backup_resume(ClientContext *ctx, char *json_payload, char *response_buffer);

// Restore functions
int client_restore_init(ClientContext *ctx, char *json_payload, char *response_buffer);
int client_restore_chunk(ClientContext *ctx, char *json_payload, char *response_buffer);
int client_restore_finish(ClientContext *ctx, char *json_payload, char *response_buffer);
int client_restore_resume(ClientContext *ctx, char *json_payload, char *response_buffer);

#endif

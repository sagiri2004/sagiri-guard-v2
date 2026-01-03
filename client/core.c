#include "core.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>
#include <arpa/inet.h>

void *listen_for_notifications(void *arg) {
    ClientContext *ctx = (ClientContext *)arg;
    char buffer[BUFFER_SIZE];
    int read_size;

    while (ctx->running && (read_size = recv_packet(ctx->notification_sock, buffer)) > 0) {
        if (ctx->on_message) {
            ctx->on_message(buffer);
        } else {
            // Default behavior
            printf("\n[NOTIFICATION] %s\nChoice: ", buffer);
            fflush(stdout);
        }
    }
    return 0;
}

// Helper for API requests
static int client_api_request(ClientContext *ctx, uint8_t type, char *json_payload, char *response_buffer) {
    int sock;
    struct sockaddr_in server;
    api_req_header_t req;
    api_resp_header_t resp;
    struct timeval tv;

    sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock == -1) return 0;

    server.sin_addr.s_addr = inet_addr(ctx->server_host);
    server.sin_family = AF_INET;
    server.sin_port = htons(ctx->api_port);

    tv.tv_sec = 10;
    tv.tv_usec = 0;
    setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, (const char*)&tv, sizeof tv);

    if (connect(sock, (struct sockaddr *)&server, sizeof(server)) < 0) {
        close(sock);
        return 0;
    }

    req.type = type;
    int p_len = (json_payload != NULL) ? strlen(json_payload) : 0;
    req.len = htons(p_len);

    if (send(sock, &req, sizeof(req), 0) < 0) {
        close(sock);
        return 0;
    }

    if (p_len > 0) {
        if (send(sock, json_payload, p_len, 0) < 0) {
            close(sock);
            return 0;
        }
    }

    if (recv(sock, &resp, sizeof(resp), 0) < 0) {
        close(sock);
        return 0;
    }

    int len = ntohs(resp.len);
    if (len > 0) {
        int total_read = 0;
        while (total_read < len) {
            int r = recv(sock, response_buffer + total_read, len - total_read, 0);
            if (r <= 0) break;
            total_read += r;
        }
        response_buffer[len] = '\0';
    } else {
        response_buffer[0] = '\0';
    }

    close(sock);
    return ntohs(resp.status) == 200;
}

ClientContext* client_init(char *host, int port, int api_port) {
    ClientContext *ctx = malloc(sizeof(ClientContext));
    ctx->notification_sock = -1;
    ctx->running = 1;
    ctx->on_message = NULL;
    strncpy(ctx->server_host, host, 255);
    ctx->server_port = port;
    ctx->api_port = api_port;
    return ctx;
}

void client_set_on_message(ClientContext *ctx, MessageCallback cb) {
    ctx->on_message = cb;
}

int client_login(ClientContext *ctx, char *username, char *password, char *device_id) {
    // API Login uses ephemeral socket but needs config from ctx
    
    int sock;
    struct sockaddr_in server;
    struct timeval tv;
    char payload[BUFFER_SIZE];
    api_req_header_t req;
    api_resp_header_t resp;
    char response_body[BUFFER_SIZE];

    sprintf(payload, "{\"username\": \"%s\", \"password\": \"%s\", \"device_id\": \"%s\"}", username, password, device_id);
    int payload_len = strlen(payload);

    sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock == -1) return 0;

    server.sin_addr.s_addr = inet_addr(ctx->server_host);
    server.sin_family = AF_INET;
    server.sin_port = htons(ctx->api_port);

    // Set timeout
    tv.tv_sec = 5; 
    tv.tv_usec = 0;
    setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, (const char*)&tv, sizeof tv);

    if (connect(sock, (struct sockaddr *)&server, sizeof(server)) < 0) {
        perror("API Connect failed");
        return 0;
    }

    req.type = MSG_LOGIN_REQ;
    req.len = htons(payload_len);
    write(sock, &req, sizeof(req));
    write(sock, payload, payload_len);

    if (recv(sock, &resp, sizeof(resp), 0) <= 0) {
        close(sock);
        return 0;
    }

    int body_len = ntohs(resp.len);
    if (body_len > BUFFER_SIZE) body_len = BUFFER_SIZE;
    
    int total_read = 0;
    while(total_read < body_len) {
        int r = recv(sock, response_body + total_read, body_len - total_read, 0);
        if (r <= 0) break;
        total_read += r;
    }
    response_body[total_read] = '\0';

    printf("API Response (%d): %s\n", ntohs(resp.status), response_body);
    close(sock);

	return ntohs(resp.status) == 200;
}

// Register probably needs context too now OR we pass host/port
// Since signature is `client_register_device(json, resp)`, we don't have ctx!
// We should update signature OR pass ctx.
// Let's update signature to take Ctx to be consistent?
// Or just pass host/port.
// BETTER: Update signature in core.h to `client_register_device(ClientContext *ctx, ...)`
// But wait, user might call this before login?
// `client_init` is called first. So `ctx` is available.
int client_register_device(ClientContext *ctx, char *json_payload, char *response_buffer) {
    int sock;
    struct sockaddr_in server;

    struct timeval tv;
    api_req_header_t req;
    api_resp_header_t resp;
    int payload_len = strlen(json_payload);

    sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock == -1) return 0;

    server.sin_addr.s_addr = inet_addr(ctx->server_host);
    server.sin_family = AF_INET;
    server.sin_port = htons(ctx->api_port);

    // Set timeout
    tv.tv_sec = 5;
    tv.tv_usec = 0;
    setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, (const char*)&tv, sizeof tv);

    if (connect(sock, (struct sockaddr *)&server, sizeof(server)) < 0) {
        perror("API Connect failed");
        return 0;
    }

    req.type = MSG_DEVICE_REQ;
    req.len = htons(payload_len);
    write(sock, &req, sizeof(req));
    write(sock, json_payload, payload_len);

    if (recv(sock, &resp, sizeof(resp), 0) <= 0) {
        close(sock);
        return 0;
    }

    int body_len = ntohs(resp.len);
    if (body_len > BUFFER_SIZE) body_len = BUFFER_SIZE;
    
    int total_read = 0;
    while(total_read < body_len) {
        int r = recv(sock, response_buffer + total_read, body_len - total_read, 0);
        if (r <= 0) break;
        total_read += r;
    }
    response_buffer[total_read] = '\0';
    close(sock);

    return ntohs(resp.status) == 200;
}

void client_connect_notification(ClientContext *ctx, char *device_id) {
    struct sockaddr_in server;
    char server_reply[BUFFER_SIZE];
    pthread_t listener_thread;

    ctx->notification_sock = socket(AF_INET, SOCK_STREAM, 0);
    if (ctx->notification_sock == -1) {
        perror("Could not create notification socket");
        return;
    }

    server.sin_addr.s_addr = inet_addr(ctx->server_host);
    server.sin_family = AF_INET;
    server.sin_port = htons(ctx->server_port);

    if (connect(ctx->notification_sock, (struct sockaddr *)&server, sizeof(server)) < 0) {
        perror("Notification Connect failed");
        return;
    }

    // Send Device ID
    send_packet(ctx->notification_sock, device_id, strlen(device_id));
    
    if (recv_packet(ctx->notification_sock, server_reply) > 0) {
         printf("Connected to Notification Server: %s\n", server_reply);
    }

    if (pthread_create(&listener_thread, NULL, listen_for_notifications, (void*)ctx) < 0) {
        perror("Could not create listener thread");
        return;
    }
    pthread_detach(listener_thread);
}

void client_close(ClientContext *ctx) {
    ctx->running = 0;
    if (ctx->notification_sock != -1) {
        close(ctx->notification_sock);
    }
    free(ctx);
}

// Same here, needs ctx for config
int client_get_online_users(ClientContext *ctx, char *json_buffer) {
    int sock;
    struct sockaddr_in server;
    struct timeval tv;
    api_req_header_t req;
    api_resp_header_t resp;

    sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock == -1) return 0;

    server.sin_addr.s_addr = inet_addr(ctx->server_host);
    server.sin_family = AF_INET;
    server.sin_port = htons(ctx->api_port);

    // Set timeout
    tv.tv_sec = 5;
    tv.tv_usec = 0;
    setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, (const char*)&tv, sizeof tv);

    if (connect(sock, (struct sockaddr *)&server, sizeof(server)) < 0) {
        perror("API Connect failed");
        return 0;
    }

    req.type = MSG_LIST_REQ;
    req.len = 0; // No Payload
    write(sock, &req, sizeof(req));

    // Receive Response
    if (recv(sock, &resp, sizeof(resp), 0) <= 0) {
        close(sock);
        return 0;
    }

    int body_len = ntohs(resp.len);
    if (body_len > BUFFER_SIZE) body_len = BUFFER_SIZE;
    
    int total_read = 0;
    while(total_read < body_len) {
        int r = recv(sock, json_buffer + total_read, body_len - total_read, 0);
        if (r <= 0) break;
        total_read += r;
    }
    json_buffer[total_read] = '\0';
    close(sock);

    return ntohs(resp.status) == 200;
}

void client_send_message(ClientContext *ctx, char *message) {
    if (ctx->notification_sock != -1) {
        send_packet(ctx->notification_sock, message, strlen(message));
    }
}

int client_send_and_wait(ClientContext *ctx, char *message, char *response_buffer, int buffer_size) {
    if (ctx->notification_sock == -1) return 0;
    
    // Send
    send_packet(ctx->notification_sock, message, strlen(message));
    
    // Recv
    // WARNING: If listen_for_notifications thread is running, it might steal this packet!
    int len = recv_packet(ctx->notification_sock, response_buffer);
    if (len > 0) {
        if (len >= buffer_size) response_buffer[buffer_size-1] = '\0';
        else response_buffer[len] = '\0';
        return 1;
    }
    return 0;
}

// --- Admin Functions ---

int client_admin_login(ClientContext *ctx, char *username, char *password) {
    int sock;
    struct sockaddr_in server;
    struct timeval tv;
    api_req_header_t req;
    api_resp_header_t resp;
    char payload[BUFFER_SIZE];
    char response_body[BUFFER_SIZE];

    sprintf(payload, "{\"username\": \"%s\", \"password\": \"%s\"}", username, password);
    int payload_len = strlen(payload);

    sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock == -1) return 0;

    server.sin_addr.s_addr = inet_addr(ctx->server_host);
    server.sin_family = AF_INET;
    server.sin_port = htons(ctx->api_port);

    tv.tv_sec = 5;
    tv.tv_usec = 0;
    setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, (const char*)&tv, sizeof tv);

    if (connect(sock, (struct sockaddr *)&server, sizeof(server)) < 0) {
        close(sock);
        return 0;
    }

    req.type = MSG_ADMIN_LOGIN_REQ;
    req.len = htons(payload_len);
    write(sock, &req, sizeof(req));
    write(sock, payload, payload_len);

    if (recv(sock, &resp, sizeof(resp), 0) <= 0) {
        close(sock);
        return 0;
    }

    int body_len = ntohs(resp.len);
    if (body_len > BUFFER_SIZE) body_len = BUFFER_SIZE;
    
    int total_read = 0;
    while(total_read < body_len) {
        int r = recv(sock, response_body + total_read, body_len - total_read, 0);
        if (r <= 0) break;
        total_read += r;
    }
    response_body[total_read] = '\0';

    // printf("Admin API Response (%d): %s\n", ntohs(resp.status), response_body);
    close(sock);

	return ntohs(resp.status) == 200;
}

int client_admin_get_logs(ClientContext *ctx, char *json_payload, char *response_buffer) {
    int sock;
    struct sockaddr_in server;
    struct timeval tv;
    api_req_header_t req;
    api_resp_header_t resp;

    sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock == -1) return 0;

    server.sin_addr.s_addr = inet_addr(ctx->server_host);
    server.sin_family = AF_INET;
    server.sin_port = htons(ctx->api_port);

    tv.tv_sec = 5;
    tv.tv_usec = 0;
    setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, (const char*)&tv, sizeof tv);

    if (connect(sock, (struct sockaddr *)&server, sizeof(server)) < 0) {
        close(sock);
        return 0;
    }

    req.type = MSG_ADMIN_COMMAND_GETLOGS_REQ;
    req.len = htons(strlen(json_payload));
    
    if (send(sock, &req, sizeof(req), 0) < 0) {
        close(sock);
        return 0;
    }
    if (send(sock, json_payload, strlen(json_payload), 0) < 0) {
        close(sock);
        return 0;
    }

    if (recv(sock, &resp, sizeof(resp), 0) < 0) {
        close(sock);
        return 0;
    }

    if (resp.type != MSG_ADMIN_COMMAND_GETLOGS_RESP) {
        close(sock);
        return 0;
    }

    int body_len = ntohs(resp.len);
    if (body_len > BUFFER_SIZE) body_len = BUFFER_SIZE; // Safety cap

    int n = recv(sock, response_buffer, body_len, 0);
    if (n >= 0) response_buffer[n] = '\0';
    close(sock);
    return ntohs(resp.status) == 200;
}

int client_admin_view_logs(ClientContext *ctx, char *json_payload, char *response_buffer) {
    int sock;
    struct sockaddr_in server;
    struct timeval tv;
    api_req_header_t req;
    api_resp_header_t resp;

    sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock == -1) return 0;

    server.sin_addr.s_addr = inet_addr(ctx->server_host);
    server.sin_family = AF_INET;
    server.sin_port = htons(ctx->api_port);

    tv.tv_sec = 5;
    tv.tv_usec = 0;
    setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, (const char*)&tv, sizeof tv);

    if (connect(sock, (struct sockaddr *)&server, sizeof(server)) < 0) {
        close(sock);
        return 0;
    }

    req.type = MSG_ADMIN_GET_STORED_LOGS_REQ;
    int p_len = (json_payload != NULL) ? strlen(json_payload) : 0;
    req.len = htons(p_len);
    
    if (send(sock, &req, sizeof(req), 0) < 0) {
        close(sock);
        return 0;
    }
    if (p_len > 0) {
        if (send(sock, json_payload, p_len, 0) < 0) {
            close(sock);
            return 0;
        }
    }

    if (recv(sock, &resp, sizeof(resp), 0) < 0) {
        close(sock);
        return 0;
    }

    // if (resp.type != MSG_ADMIN_GET_STORED_LOGS_RESP) return 0;

    // Buffer might need to be large
    int body_len = ntohs(resp.len);
    int total_read = 0;
    while(total_read < body_len) {
        int r = recv(sock, response_buffer + total_read, body_len - total_read, 0);
        if (r <= 0) break;
        total_read += r;
    }
    response_buffer[total_read] = '\0';
    
    close(sock);
    return ntohs(resp.status) == 200;
}

int client_upload_logs(ClientContext *ctx, char *logs_payload, char *response_buffer) {
    int sock;
    struct sockaddr_in server;
    struct timeval tv;
    api_req_header_t req;
    api_resp_header_t resp;

    sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock == -1) return 0;

    server.sin_addr.s_addr = inet_addr(ctx->server_host);
    server.sin_family = AF_INET;
    server.sin_port = htons(ctx->api_port);

    tv.tv_sec = 5;
    tv.tv_usec = 0;
    setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, (const char*)&tv, sizeof tv);

    if (connect(sock, (struct sockaddr *)&server, sizeof(server)) < 0) {
        close(sock);
        return 0;
    }

    req.type = MSG_CLIENT_COMMAND_GETLOG_REQ;
    int p_len = strlen(logs_payload);
    req.len = htons(p_len);
    
    send(sock, &req, sizeof(req), 0);
    send(sock, logs_payload, p_len, 0);

    // Wait for Ack
    recv(sock, &resp, sizeof(resp), 0);
    int resp_len = ntohs(resp.len);
    if (resp_len > BUFFER_SIZE) resp_len = BUFFER_SIZE;

    int n = recv(sock, response_buffer, resp_len, 0);
    if (n >= 0) response_buffer[n] = '\0';
    
    close(sock);
    return 1;
}

int client_admin_get_history(ClientContext *ctx, char *json_payload, char *response_buffer) {
    int sock;
    struct sockaddr_in server;
    struct timeval tv;
    api_req_header_t req;
    api_resp_header_t resp;

    sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock == -1) return 0;

    server.sin_addr.s_addr = inet_addr(ctx->server_host);
    server.sin_family = AF_INET;
    server.sin_port = htons(ctx->api_port);

    tv.tv_sec = 5;
    tv.tv_usec = 0;
    setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, (const char*)&tv, sizeof tv);

    if (connect(sock, (struct sockaddr *)&server, sizeof(server)) < 0) {
        close(sock);
        return 0;
    }

    req.type = MSG_ADMIN_GET_COMMAND_HISTORY_REQ;
    int p_len = (json_payload != NULL) ? strlen(json_payload) : 0;
    req.len = htons(p_len);
    
    if (send(sock, &req, sizeof(req), 0) < 0) {
        close(sock);
        return 0;
    }
    if (p_len > 0) {
        if (send(sock, json_payload, p_len, 0) < 0) {
            close(sock);
            return 0;
        }
    }

    if (recv(sock, &resp, sizeof(resp), 0) < 0) {
        close(sock);
        return 0;
    }

    int len = ntohs(resp.len);
    if (len > 0) {
        int total_read = 0;
        while (total_read < len) {
            int r = recv(sock, response_buffer + total_read, len - total_read, 0);
            if (r <= 0) break;
            total_read += r;
        }
        response_buffer[len] = '\0';
    } else {
        response_buffer[0] = '\0';
    }

    close(sock);
    return ntohs(resp.status) == 200;
}

int client_get_firewall_config(ClientContext *ctx, char *json_payload, char *response_buffer) {
    int sock;
    struct sockaddr_in server;
    struct timeval tv;
    api_req_header_t req;
    api_resp_header_t resp;

    sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock == -1) return 0;

    server.sin_addr.s_addr = inet_addr(ctx->server_host);
    server.sin_family = AF_INET;
    server.sin_port = htons(ctx->api_port);

    tv.tv_sec = 5;
    tv.tv_usec = 0;
    setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, (const char*)&tv, sizeof tv);

    if (connect(sock, (struct sockaddr *)&server, sizeof(server)) < 0) {
        close(sock);
        return 0;
    }

    req.type = MSG_CLIENT_GET_FIREWALL_CONFIG_REQ;
    int p_len = (json_payload != NULL) ? strlen(json_payload) : 0;
    req.len = htons(p_len);
    
    if (send(sock, &req, sizeof(req), 0) < 0) {
        close(sock);
        return 0;
    }
    if (p_len > 0) {
        if (send(sock, json_payload, p_len, 0) < 0) {
            close(sock);
            return 0;
        }
    }

    if (recv(sock, &resp, sizeof(resp), 0) < 0) {
        close(sock);
        return 0;
    }

    int len = ntohs(resp.len);
    if (len > 0) {
        int total_read = 0;
        while (total_read < len) {
            int r = recv(sock, response_buffer + total_read, len - total_read, 0);
            if (r <= 0) break;
            total_read += r;
        }
        response_buffer[len] = '\0';
    } else {
        response_buffer[0] = '\0';
    }

    close(sock);
    return ntohs(resp.status) == 200;
}

int client_admin_firewall_control(ClientContext *ctx, char *json_payload, char *response_buffer) {
    int sock;
    struct sockaddr_in server;
    struct timeval tv;
    api_req_header_t req;
    api_resp_header_t resp;

    sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock == -1) return 0;

    server.sin_addr.s_addr = inet_addr(ctx->server_host);
    server.sin_family = AF_INET;
    server.sin_port = htons(ctx->api_port);

    tv.tv_sec = 5;
    tv.tv_usec = 0;
    setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, (const char*)&tv, sizeof tv);

    if (connect(sock, (struct sockaddr *)&server, sizeof(server)) < 0) {
        close(sock);
        return 0;
    }

    req.type = MSG_ADMIN_FIREWALL_CONTROL_REQ;
    int p_len = (json_payload != NULL) ? strlen(json_payload) : 0;
    req.len = htons(p_len);
    
    if (send(sock, &req, sizeof(req), 0) < 0) {
        close(sock);
        return 0;
    }
    if (p_len > 0) {
        if (send(sock, json_payload, p_len, 0) < 0) {
            close(sock);
            return 0;
        }
    }

    if (recv(sock, &resp, sizeof(resp), 0) < 0) {
        close(sock);
        return 0;
    }

    printf("[DEBUG] Firewall Resp Status: %d\n", ntohs(resp.status));

    int len = ntohs(resp.len);
    if (len > 0) {
        int total_read = 0;
        while (total_read < len) {
            int r = recv(sock, response_buffer + total_read, len - total_read, 0);
            if (r <= 0) break;
            total_read += r;
        }
        response_buffer[len] = '\0';
    } else {
        response_buffer[0] = '\0';
    }

    close(sock);
    return ntohs(resp.status) == 200;
}

int client_file_sync(ClientContext *ctx, char *json_payload, char *response_buffer) {
    return client_api_request(ctx, MSG_CLIENT_FILE_SYNC_REQ, json_payload, response_buffer);
}
int client_admin_get_file_tree(ClientContext *ctx, char *json_payload, char *response_buffer) {
    return client_api_request(ctx, MSG_ADMIN_GET_FILE_TREE_REQ, json_payload, response_buffer);
}

int client_backup_init(ClientContext *ctx, char *json_payload, char *response_buffer) {
    return client_api_request(ctx, MSG_BACKUP_INIT_REQ, json_payload, response_buffer);
}

int client_backup_chunk(ClientContext *ctx, char *json_payload, char *response_buffer) {
    return client_api_request(ctx, MSG_BACKUP_CHUNK_REQ, json_payload, response_buffer);
}

int client_backup_finish(ClientContext *ctx, char *json_payload, char *response_buffer) {
    return client_api_request(ctx, MSG_BACKUP_FINISH_REQ, json_payload, response_buffer);
}

int client_backup_cancel(ClientContext *ctx, char *json_payload, char *response_buffer) {
    return client_api_request(ctx, MSG_BACKUP_CANCEL_REQ, json_payload, response_buffer);
}


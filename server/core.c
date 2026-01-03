#include "core.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>

// Internal struct to pass context to threads
typedef struct {
    ServerContext *ctx;
    int socket; // For handling API/Client specific socket
} ThreadArgs;

typedef struct {
    ServerContext *ctx;
    int socket;
    char device_id[64];
} client_t;

// --- Helper Functions (From old main.c but now using Context) ---

void add_client_to_list(ServerContext *ctx, char *device_id, int socket) {
    pthread_mutex_lock(&ctx->lock);
    client_node_t *new_node = (client_node_t *)malloc(sizeof(client_node_t));
    strncpy(new_node->device_id, device_id, 63);
    new_node->device_id[63] = '\0';
    new_node->socket = socket;
    new_node->next = ctx->head;
    ctx->head = new_node;
    pthread_mutex_unlock(&ctx->lock);
}

void remove_client_from_list(ServerContext *ctx, char *device_id) {
    pthread_mutex_lock(&ctx->lock);
    client_node_t *temp = ctx->head, *prev = NULL;

    if (temp != NULL && strcmp(temp->device_id, device_id) == 0) {
        ctx->head = temp->next;
        free(temp);
        pthread_mutex_unlock(&ctx->lock);
        return;
    }

    while (temp != NULL && strcmp(temp->device_id, device_id) != 0) {
        prev = temp;
        temp = temp->next;
    }

    if (temp == NULL) {
        pthread_mutex_unlock(&ctx->lock);
        return;
    }

    prev->next = temp->next;
    free(temp);
    pthread_mutex_unlock(&ctx->lock);
}


void print_online_users(ServerContext *ctx) {
    pthread_mutex_lock(&ctx->lock);
    client_node_t *temp = ctx->head;
    printf("Online Users: ");
    while (temp != NULL) {
        printf("%s ", temp->device_id);
        temp = temp->next;
    }
    printf("\n");
    pthread_mutex_unlock(&ctx->lock);
}

int server_send_to_device(ServerContext *ctx, char *target_device_id, uint8_t type, char *payload) {
    pthread_mutex_lock(&ctx->lock);
    client_node_t *temp = ctx->head;
    while(temp != NULL) {
        if (strcmp(temp->device_id, target_device_id) == 0) {
            // Found
            proto_header_t header;
            header.type = type;
            int len = strlen(payload);
            header.len = htons(len);
            write(temp->socket, &header, sizeof(header));
            write(temp->socket, payload, len);
            pthread_mutex_unlock(&ctx->lock);
            return 1;
        }
        temp = temp->next;
    }
    pthread_mutex_unlock(&ctx->lock);
    return 0;
}

void broadcast_message(ServerContext *ctx, char *sender_id, char *message) {
    pthread_mutex_lock(&ctx->lock);
    client_node_t *temp = ctx->head;
    while (temp != NULL) {
        // Only send notifications to ADMIN_CONSOLE, and not back to sender
        if ((sender_id == NULL || strcmp(temp->device_id, sender_id) != 0) &&
            strcmp(temp->device_id, "ADMIN_CONSOLE") == 0) {
            send_packet(temp->socket, message, strlen(message));
        }
        temp = temp->next;
    }
    pthread_mutex_unlock(&ctx->lock);
}

// --- Handlers ---

void *handle_client(void *arg) {
    client_t *client_info = (client_t*)arg;
    ServerContext *ctx = client_info->ctx;
    int sock = client_info->socket;
    char device_id[64];
    int read_size;
    char client_message[BUFFER_SIZE];
    char message[BUFFER_SIZE];

    // Read Device ID
    char id_buffer[128];
    if (recv_packet(sock, id_buffer) <= 0) {
        close(sock);
        free(client_info);
        return 0;
    }
    // Just wrap it if needed or use directly
    strncpy(device_id, id_buffer, 63);
    device_id[63] = '\0';
    strcpy(client_info->device_id, device_id);
    
    add_client_to_list(ctx, device_id, sock);
    print_online_users(ctx);
    
    // Broadcast join (Only to Admin)
    // Broadcast join (Only to Admin)
    sprintf(message, "Device %s Online\n", device_id);
    broadcast_message(ctx, device_id, message);

    // Removed duplicate broadcast call


    // Call Go Callback if set
    if (ctx->on_connect) {
        ctx->on_connect(device_id);
    }

    sprintf(message, "Hello %s from server handler\n", device_id);
    send_packet(sock, message, strlen(message));

    while ((read_size = recv_packet(sock, client_message)) > 0) {
        printf("Client %s: %s\n", device_id, client_message);
        send_packet(sock, client_message, strlen(client_message)); // Echo back
    }

    if (read_size == 0 || read_size == -1) {
         printf("Client %s disconnected\n", device_id);
    }

    remove_client_from_list(ctx, device_id);
    print_online_users(ctx);
    
    // Broadcast leave
    sprintf(message, "Device %s Offline\n", device_id);
    broadcast_message(ctx, device_id, message);

    pthread_mutex_lock(&ctx->lock);
    ctx->active_clients--;
    printf("Total active clients: %d\n", ctx->active_clients);
    pthread_mutex_unlock(&ctx->lock);

    close(sock);
    free(client_info);
    return 0;
}

void *handle_api_request(void *arg) {
    ThreadArgs *args = (ThreadArgs*)arg;
    int sock = args->socket;
    ServerContext *ctx = args->ctx;
    
    // Read Header (1st byte)
    uint8_t first_byte;
    if (recv(sock, &first_byte, 1, 0) <= 0) {
        close(sock);
        free(arg);
        return 0;
    }

    uint8_t req_type;
    uint32_t payload_len;

    if (first_byte == PROTOCOL_MAGIC_EXT) {
        // Extended Header (0xFE + type + uint32_t len)
        struct {
            uint8_t type;
            uint32_t len;
        } __attribute__((packed)) ext;
        if (recv(sock, &ext, sizeof(ext), 0) <= 0) {
            close(sock);
            free(arg);
            return 0;
        }
        req_type = ext.type;
        payload_len = ntohl(ext.len);
        printf("[API] Extended Header Detected (Type: 0x%02X, Len: %u)\n", req_type, payload_len);
    } else {
        // Standard Header (type + uint16_t len)
        req_type = first_byte;
        uint16_t len16;
        if (recv(sock, &len16, sizeof(len16), 0) <= 0) {
            close(sock);
            free(arg);
            return 0;
        }
        payload_len = ntohs(len16);
    }

    // Read Body
    char *payload = malloc(payload_len + 1);
    if (!payload) {
        close(sock);
        free(arg);
        return 0;
    }
    
    uint32_t total_read = 0;
    while (total_read < payload_len) {
        int r = recv(sock, payload + total_read, payload_len - total_read, 0);
        if (r <= 0) break;
        total_read += r;
    }
    payload[payload_len] = '\0';

    // Delegate to Handler if set
    if (ctx->handler) {
        printf("[API] Delegating to Handler\n");
        ctx->handler(sock, req_type, payload);
        // Handler is responsible for response and closing socket
        // But wait, if handler is async/go, we might need to be careful.
        // Assuming handler is blocking for this request.
        // We do NOT close sock here if handler takes over?
        // Let's assume handler does Everything.
        free(payload);
        free(arg);
        return 0;
    }

    api_resp_header_t resp_header;
    char response_body[BUFFER_SIZE];
    
    if (req_type == MSG_LOGIN_REQ) {
        printf("[API] Login Request: %s\n", payload);
        if (strstr(payload, "\"username\": \"admin\"") && strstr(payload, "\"password\": \"admin\"")) {
            resp_header.status = htons(200);
            sprintf(response_body, "{\"message\": \"Login Successful\"}");
        } else {
            resp_header.status = htons(401);
            sprintf(response_body, "{\"error\": \"Invalid Credentials\"}");
        }
    } else if (req_type == MSG_LIST_REQ) {
        printf("[API] List Request\n");
        // Build JSON List
        strcpy(response_body, "{\"users\": [");
        pthread_mutex_lock(&ctx->lock);
        client_node_t *temp = ctx->head;
        while (temp != NULL) {
            char id_str[64];
            sprintf(id_str, "\"%s\"", temp->device_id); // Quote the string
            strcat(response_body, id_str);
            if (temp->next != NULL) strcat(response_body, ", ");
            temp = temp->next;
        }
        pthread_mutex_unlock(&ctx->lock);
        strcat(response_body, "]}");
        resp_header.status = htons(200);

    } else {
         resp_header.status = htons(400); 
         sprintf(response_body, "{\"error\": \"Unknown Request Type\"}");
    }
    
    resp_header.type = MSG_LOGIN_RESP;
    resp_header.len = htons(strlen(response_body));
    
    write(sock, &resp_header, sizeof(resp_header));
    write(sock, response_body, strlen(response_body));

    free(payload);
    close(sock);
    free(arg);
    return 0;
}

void *run_notification_server(void *arg) {
    ServerContext *ctx = (ServerContext *)arg;
    int socket_desc, client_sock, c;
    struct sockaddr_in server, client;

    socket_desc = socket(AF_INET, SOCK_STREAM, 0);
    if (socket_desc == -1) return 0;

    int opt = 1;
    setsockopt(socket_desc, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));

    server.sin_family = AF_INET;
    server.sin_addr.s_addr = INADDR_ANY;
    server.sin_port = htons(ctx->port);

    if (bind(socket_desc, (struct sockaddr *)&server, sizeof(server)) < 0) {
        perror("[Notification] bind failed");
        return 0;
    }
    listen(socket_desc, 3);
    
    printf("[Notification] Bind done on port %d\n", ctx->port);
    c = sizeof(struct sockaddr_in);

    while (ctx->running && (client_sock = accept(socket_desc, (struct sockaddr *)&client, (socklen_t*)&c))) {
        pthread_mutex_lock(&ctx->lock);
        ctx->active_clients++;
        pthread_mutex_unlock(&ctx->lock);

        pthread_t sniffer_thread;
        client_t *new_client = malloc(sizeof(client_t));
        new_client->socket = client_sock;
        new_client->ctx = ctx;

        if (pthread_create(&sniffer_thread, NULL, handle_client, (void*)new_client) < 0) {
            perror("[Notification] could not create thread");
            return 0;
        }
        pthread_detach(sniffer_thread);
    }
    return 0;
}

void *run_api_server(void *arg) {
    ServerContext *ctx = (ServerContext *)arg;
    int socket_desc, client_sock, c;
    struct sockaddr_in server, client;

    socket_desc = socket(AF_INET, SOCK_STREAM, 0);
    if (socket_desc == -1) return 0;
    
    int opt = 1;
    setsockopt(socket_desc, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));

    server.sin_family = AF_INET;
    server.sin_addr.s_addr = INADDR_ANY;
    server.sin_port = htons(ctx->api_port);

    if (bind(socket_desc, (struct sockaddr *)&server, sizeof(server)) < 0) {
        perror("[API] bind failed");
        return 0;
    }
    listen(socket_desc, 3);
    printf("[API] Bind done on port %d\n", ctx->api_port);
    c = sizeof(struct sockaddr_in);

    while (ctx->running && (client_sock = accept(socket_desc, (struct sockaddr *)&client, (socklen_t*)&c))) {
        pthread_t api_thread;
        
        ThreadArgs *args = malloc(sizeof(ThreadArgs));
        args->ctx = ctx;
        args->socket = client_sock;

        if (pthread_create(&api_thread, NULL, handle_api_request, (void*)args) < 0) {
            perror("[API] could not create thread");
            return 0;
        }
        pthread_detach(api_thread);
    }
    return 0;
}

// --- Core API ---

ServerContext* server_init(int port, int api_port) {
    ServerContext *ctx = malloc(sizeof(ServerContext));
    ctx->port = port;
    ctx->api_port = api_port;
    ctx->active_clients = 0;
    ctx->head = NULL;
    ctx->running = 1;
    ctx->handler = NULL;
    ctx->on_connect = NULL;
    pthread_mutex_init(&ctx->lock, NULL);
    return ctx;
}

void server_set_handler(ServerContext *ctx, RequestHandler handler) {
    ctx->handler = handler;
}

void server_set_on_connect(ServerContext *ctx, ClientConnectCallback cb) {
    ctx->on_connect = cb;
}

void server_start(ServerContext *ctx) {
    pthread_t notification_thread, api_thread;

    if (pthread_create(&notification_thread, NULL, run_notification_server, (void*)ctx) < 0) {
        perror("Could not create notification thread");
        return;
    }

    if (pthread_create(&api_thread, NULL, run_api_server, (void*)ctx) < 0) {
        perror("Could not create api thread");
        return;
    }
    
    // In a real library, we might want to return thread IDs or allow joining.
    // For now, we detach or just let them run. 
    // Since `server_start` is usually blocking in the old code, here we spawning them.
    // If the caller (Go) wants to block, it needs a way to wait. 
    // We will detach them and let the caller decide when to stop.
    pthread_detach(notification_thread);
    pthread_detach(api_thread);
}

void server_stop(ServerContext *ctx) {
    ctx->running = 0;
    // Real cleanup would involve closing sockets to break accept loops
    // and freeing list. For demo, just flipping flag.
}

int server_get_online_users(ServerContext *ctx, char ***ids) {
    pthread_mutex_lock(&ctx->lock);
    int count = ctx->active_clients;
    if (count == 0) {
        *ids = NULL;
        pthread_mutex_unlock(&ctx->lock);
        return 0;
    }
    
    *ids = malloc(count * sizeof(char*));
    int i = 0;
    client_node_t *temp = ctx->head;
    while(temp != NULL && i < count) {
        (*ids)[i] = strdup(temp->device_id); // Caller must free each string AND the array
        i++;
        temp = temp->next;
    }
    pthread_mutex_unlock(&ctx->lock);
    return i; /* actual count */
}

void server_broadcast(ServerContext *ctx, char *message) {
    // Reuse internal helper but logic is same
    // Internal helper was `broadcast_message` which took (ctx, sender_id).
    // If we want to broadcast as "Server" or "System", we can pass sender_id = -1
    // provided we update logic to handle -1 (always display).
    // Current broadcast_message checks `if (temp->id != sender_id)`.
    // If we pass -1, and IDs are positive, it sends to everyone.
    // broadcast_message(ctx, message, -1);
    broadcast_message(ctx, message, NULL);
}

void server_send_response(int sock, int type, int status, char *message) {
    uint32_t msg_len = (message != NULL) ? (uint32_t)strlen(message) : 0;

    if (msg_len > 65535) {
        api_resp_header_ext_t resp_header;
        resp_header.magic = PROTOCOL_MAGIC_EXT;
        resp_header.type = (uint8_t)type;
        resp_header.len = htonl(msg_len);
        resp_header.status = htons(status);
        write(sock, &resp_header, sizeof(resp_header));
    } else {
        api_resp_header_t resp_header;
        resp_header.type = (uint8_t)type;
        resp_header.status = htons(status);
        resp_header.len = htons((uint16_t)msg_len);
        write(sock, &resp_header, sizeof(resp_header));
    }

    if (msg_len > 0) {
        uint32_t total_written = 0;
        while (total_written < msg_len) {
            ssize_t n = write(sock, message + total_written, msg_len - total_written);
            if (n <= 0) break;
            total_written += (uint32_t)n;
        }
    }
    close(sock);
}

int server_send_unicast(ServerContext *ctx, char *client_id, char *message) {
    pthread_mutex_lock(&ctx->lock);
    client_node_t *temp = ctx->head;
    while (temp != NULL) {
        if (strcmp(temp->device_id, client_id) == 0) {
             send_packet(temp->socket, message, strlen(message));
             pthread_mutex_unlock(&ctx->lock);
             return 1; // Success
        }
        temp = temp->next;
    }
    pthread_mutex_unlock(&ctx->lock);
    return 0; // Not found
}

int server_get_device_id_by_sock(ServerContext *ctx, int sock, char *buffer) {
    pthread_mutex_lock(&ctx->lock);
    client_node_t *temp = ctx->head;
    while(temp != NULL) {
        if (temp->socket == sock) {
            strcpy(buffer, temp->device_id);
            pthread_mutex_unlock(&ctx->lock);
            return 1;
        }
        temp = temp->next;
    }
    pthread_mutex_unlock(&ctx->lock);
    return 0;
}

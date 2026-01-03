#ifndef PROTOCOL_H
#define PROTOCOL_H

#include <stdint.h>
#include <unistd.h>
#include <arpa/inet.h>
#include <stdio.h>

#define PORT 8080
#define API_PORT 8081
#define BUFFER_SIZE 1024

#define MSG_SOCKET 0x01

// Restore Flow
#define MSG_ADMIN_RESTORE_REQ      0x70
#define MSG_ADMIN_RESTORE_RESP     0x71
#define MSG_SERVER_RESTORE_CMD     0x72
#define MSG_RESTORE_INIT_REQ       0x73
#define MSG_RESTORE_INIT_RESP      0x74
#define MSG_RESTORE_CHUNK_REQ      0x75
#define MSG_RESTORE_CHUNK_RESP     0x76
#define MSG_RESTORE_FINISH_REQ     0x77
#define MSG_RESTORE_FINISH_RESP    0x78
#define MSG_RESTORE_RESUME_REQ     0x79
#define MSG_RESTORE_RESUME_RESP     0x7A

#define MSG_LOGIN_REQ 0xA1
#define MSG_LOGIN_RESP 0xA2
#define MSG_LIST_REQ 0xB1
#define MSG_LIST_RESP 0xB2
#define MSG_DEVICE_REQ 0xC1
#define MSG_DEVICE_RESP 0xC2

// Admin Command Flow
#define MSG_ADMIN_COMMAND_GETLOGS_REQ 0xD1
#define MSG_ADMIN_COMMAND_GETLOGS_RESP 0xD2
#define MSG_SERVER_COMMAND_GETLOG 0xD3
#define MSG_CLIENT_COMMAND_GETLOG_REQ 0xD4
#define MSG_CLIENT_COMMAND_GETLOG_RESP 0xD5
#define MSG_ADMIN_LOGIN_REQ 0xD6
#define MSG_ADMIN_LOGIN_RESP 0xD7
#define MSG_ADMIN_GET_STORED_LOGS_REQ 0xD8
#define MSG_ADMIN_GET_STORED_LOGS_RESP 0xD9
#define MSG_ADMIN_GET_COMMAND_HISTORY_REQ 0xDA
#define MSG_ADMIN_GET_COMMAND_HISTORY_RESP 0xDB

#define MSG_ADMIN_FIREWALL_CONTROL_REQ 0xE1
#define MSG_ADMIN_FIREWALL_CONTROL_RESP 0xE2
#define MSG_SERVER_FIREWALL_UPDATE_CMD 0xE3

#define MSG_CLIENT_GET_FIREWALL_CONFIG_REQ 0xE4
#define MSG_CLIENT_GET_FIREWALL_CONFIG_RESP 0xE5

#define MSG_CLIENT_FILE_SYNC_REQ 0xE6
#define MSG_CLIENT_FILE_SYNC_RESP 0xE7

#define MSG_ADMIN_GET_FILE_TREE_REQ 0xE8
#define MSG_ADMIN_GET_FILE_TREE_RESP 0xE9

#define MSG_BACKUP_INIT_REQ        0xF1
#define MSG_BACKUP_INIT_RESP       0xF2
#define MSG_BACKUP_CHUNK_REQ       0xF3
#define MSG_BACKUP_CHUNK_RESP      0xF4
#define MSG_BACKUP_FINISH_REQ      0xF5
#define MSG_BACKUP_FINISH_RESP     0xF6
#define MSG_BACKUP_CANCEL_REQ      0xF7
#define MSG_BACKUP_RESUME_REQ      0xF8
#define MSG_BACKUP_RESUME_RESP     0xF9

// Callback type for receiving messages
typedef void (*MessageCallback)(const char *message);

#define PROTOCOL_MAGIC_EXT 0xFE

typedef struct __attribute__((packed)) {
    uint8_t type;
    uint16_t len;
} proto_header_t; // 8080

typedef struct __attribute__((packed)) {
    uint8_t type;
    uint16_t len;
} api_req_header_t; // 8081

typedef struct __attribute__((packed)) {
    uint8_t magic; // 0xFE
    uint8_t type;
    uint32_t len;
} api_req_header_ext_t; // 8081 Extended

typedef struct __attribute__((packed)) {
    uint8_t type;
    uint16_t len;
    uint16_t status;
} api_resp_header_t; // 8081

typedef struct __attribute__((packed)) {
    uint8_t magic; // 0xFE
    uint8_t type;
    uint32_t len;
    uint16_t status;
} api_resp_header_ext_t; // 8081 Extended

// Helper to send packet
void send_packet(int sock, void *data, uint16_t len);

// Helper to recv packet
int recv_packet(int sock, char *buffer);

#endif

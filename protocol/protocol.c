#include "protocol.h"

// Helper to send packet
void send_packet(int sock, void *data, uint16_t len) {
    proto_header_t header;
    header.type = MSG_SOCKET;
    header.len = htons(len);

    write(sock, &header, sizeof(header));
    write(sock, data, len);
}

// Helper to recv packet
int recv_packet(int sock, char *buffer) {
    proto_header_t header;
    if (recv(sock, &header, sizeof(header), 0) <= 0) return -1;
    
    // if (header.type != MSG_SOCKET && header.type != MSG_SERVER_COMMAND_GETLOG && header.type != MSG_SERVER_FIREWALL_UPDATE_CMD) {
    //     printf("Invalid packet type: 0x%02X\n", header.type);
    //     // For now, return -1. Or should we allow it?
    //     // Let's allow it if we want generic handling
    //     // return -1;
    // }

    uint16_t len = ntohs(header.len);
    if (len > BUFFER_SIZE) len = BUFFER_SIZE; // Cap to buffer

    int total_read = 0;
    while (total_read < len) {
        int r = recv(sock, buffer + total_read, len - total_read, 0);
        if (r <= 0) return -1;
        total_read += r;
    }
    buffer[total_read] = '\0';
    return total_read;
}

#include <stdio.h>
#include <unistd.h>
#include "core.h"

int main() {
    int choice;
    char username[50], password[50];
    char id[64];
    
    printf("--- CLIENT START ---\n");
    printf("Enter Username: ");
    scanf("%s", username);
    printf("Enter Password: ");
    scanf("%s", password);

    // Hardcoded defaults for legacy C app
    ClientContext *ctx = client_init("127.0.0.1", 8080, 8081);
    sprintf(id, "C-CLIENT-%d", getpid()); // Temporary ID

    printf("Connecting...\n");
    // Passing "C-CLIENT" as device_id for legacy test
    if (!client_login(ctx, username, password, id)) {
        printf("Login Failed\n");
        return 0;
    }

    printf("Login Success\n");
    client_connect_notification(ctx, id);

    // 3. Main Menu
    while (1) {
        printf("\n1. List Online Users\n");
        printf("2. Exit\n");
        printf("Choice: ");
        
        if (scanf("%d", &choice) != 1) {
            char c;
            while ((c = getchar()) != '\n' && c != EOF);
            printf("Invalid input. Please enter a number.\n");
            continue;
        }

        if (choice == 1) {
           char list_json[1024];
           if (client_get_online_users(ctx, list_json)) {
                printf("Online Users: %s\n", list_json);
           } else {
                printf("Failed to get list.\n");
           }
        } else if (choice == 2) {
            break;
        } else {
            printf("Invalid choice\n");
        }
        
        usleep(100000); 
    }

    client_close(ctx);
    return 0;
}

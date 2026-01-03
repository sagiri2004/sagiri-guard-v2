CC = gcc
CFLAGS = -Wall -Wextra -I./protocol -I./server -I./client

all: server_app client_app server_go client_go client_admin

protocol.o: protocol/protocol.c protocol/protocol.h
	$(CC) $(CFLAGS) -c protocol/protocol.c -o protocol.o

server_core.o: server/core.c server/core.h protocol.o
	$(CC) $(CFLAGS) -c server/core.c -o server_core.o

shim.o: server/shim.c server/shim.h
	$(CC) $(CFLAGS) -c server/shim.c -o shim.o

client_core.o: client/core.c client/core.h protocol.o
	$(CC) $(CFLAGS) -c client/core.c -o client_core.o

server_app: server/main.c server_core.o protocol.o
	$(CC) $(CFLAGS) -o server_app server/main.c server_core.o protocol.o -pthread

client_app: client/main.c client_core.o protocol.o
	$(CC) $(CFLAGS) -o client_app client/main.c client_core.o protocol.o -pthread

server_go: server_core.o protocol.o shim.o
	go build -v -o server_go ./go_server

client_go: client_core.o protocol.o
	go build -v -o client_go ./go_client

client_admin: client_core.o protocol.o
	go build -v -o client_admin ./go_client_admin

clean:
	rm -f server_app client_app server_go client_go client_admin *.o

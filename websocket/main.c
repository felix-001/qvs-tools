// Copyright (c) 2020 Cesanta Software Limited
// All rights reserved
//
// Example Websocket server. Usage:
//  1. Start this server, type `make`
//  2. Open https://www.websocket.org/echo.html in your browser
//  3. In the "Location" text field, type ws://127.0.0.1:8000/websocket
#include <unistd.h> 
#include <stdio.h> 
#include <sys/socket.h> 
#include <stdlib.h> 
#include <netinet/in.h> 
#include <string.h> 
#include "mongoose.h"

static const char *s_web_directory = ".";

// This RESTful server implements the following endpoints:
//   /websocket - upgrade to Websocket, and implement websocket echo server
//   /api/rest - respond with JSON string {"result": 123}
//   any other URI serves static files from s_web_directory
static void fn(struct mg_connection *c, int ev, void *ev_data, void *fn_data) {
  if (ev == MG_EV_HTTP_MSG) {
    struct mg_http_message *hm = (struct mg_http_message *) ev_data;
    if (mg_http_match_uri(hm, "/websocket")) {
      // Upgrade to websocket. From now on, a connection is a full-duplex
      // Websocket connection, which will receive MG_EV_WS_MSG events.
      mg_ws_upgrade(c, hm, NULL);
    } else if (mg_http_match_uri(hm, "/rest")) {
      // Serve REST response
      mg_http_reply(c, 200, "", "{\"result\": %d}\n", 123);
    } else {
      // Serve static files
      struct mg_http_serve_opts opts = {.root_dir = s_web_directory};
      mg_http_serve_dir(c, ev_data, &opts);
    }
  } else if (ev == MG_EV_WS_MSG) {
    // Got websocket frame. Received data is wm->data. Echo it back!
    struct mg_ws_message *wm = (struct mg_ws_message *) ev_data;
    mg_ws_send(c, wm->data.ptr, wm->data.len, WEBSOCKET_OP_TEXT);
    mg_iobuf_delete(&c->recv, c->recv.len);
  } else if (ev == MG_EV_CLOSE) {
    printf("got message MG_EV_CLOSE\n");
  }
  (void) fn_data;
}

int init_net(struct sockaddr_in *peer_addr)
{
    int server_fd, new_socket; 
    struct sockaddr_in address; 
    //int opt = 1; 
    int addrlen = sizeof(address); 
       
    // Creating socket file descriptor 
    if ((server_fd = socket(AF_INET, SOCK_STREAM, 0)) == 0) 
    { 
        perror("socket failed"); 
        exit(EXIT_FAILURE); 
    } 
       
    // Forcefully attaching socket to the port 8080 
    /*
    if (setsockopt(server_fd, SOL_SOCKET, SO_REUSEADDR | SO_REUSEPORT, 
                                                  &opt, sizeof(opt))) 
    { 
        perror("setsockopt"); 
        exit(EXIT_FAILURE); 
    } 
    */
    address.sin_family = AF_INET; 
    address.sin_addr.s_addr = INADDR_ANY; 
    address.sin_port = htons(8000); 
       
    // Forcefully attaching socket to the port 8000 
    if (bind(server_fd, (struct sockaddr *)&address,  
                                 sizeof(address))<0) 
    { 
        perror("bind failed"); 
        exit(EXIT_FAILURE); 
    } 
    if (listen(server_fd, 3) < 0) 
    { 
        perror("listen"); 
        exit(EXIT_FAILURE); 
    } 
    if ((new_socket = accept(server_fd, (struct sockaddr *)&address,  
                       (socklen_t*)&addrlen))<0) 
    { 
        perror("accept"); 
        exit(EXIT_FAILURE); 
    } 
    *peer_addr = address;
    return new_socket;
}

int main(void) {
  struct mg_mgr mgr;                            // Event manager
  struct sockaddr_in peer_addr;
  int fd = init_net(&peer_addr);
  mg_mgr_init(&mgr);                            // Initialise event manager
  struct mg_connection *conn = mg_http_new_connection(&mgr, fd, &peer_addr, fn, NULL);  // Create HTTP listener
  if (!conn) {
    printf("check conn error\n");
    return 0;
  }
  while (!conn->is_closing) mg_mgr_poll(&mgr, 1000);             // Infinite event loop
  mg_mgr_free(&mgr);
  // FIXME: 是否需要close fd, mongoose有没有close
  // 结束时client会发送一个opcode为WEBSOCKET_OP_CLOSE
  // mongoose会close掉socket
  return 0;
}

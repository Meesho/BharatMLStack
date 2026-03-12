#ifndef EIGENIX_DIST_NET_HPP
#define EIGENIX_DIST_NET_HPP

#include "dist_protocol.hpp"
#include <cstddef>
#include <cstdint>
#include <string>
#include <vector>

namespace eigenix::dist {

// Blocking send: loops until all bytes sent. Returns true on success.
bool send_all(int fd, const void* buf, size_t len);

// Blocking recv: loops until exactly len bytes received. Returns true on success.
bool recv_all(int fd, void* buf, size_t len);

// Send header + payload as one logical message.
bool send_msg(int fd, MsgType type, const void* payload = nullptr, uint32_t payload_len = 0);

// Receive header + payload. Allocates payload into out_payload.
bool recv_msg(int fd, MsgHeader& hdr, std::vector<uint8_t>& out_payload);

// Create a listening TCP socket on port. Returns fd or -1.
int make_listener(uint16_t port, int backlog = 64);

// Accept one connection from listen_fd. Sets TCP_NODELAY. Returns fd or -1.
int accept_one(int listen_fd);

// Connect to host:port with retry. Returns fd or -1.
int connect_to(const std::string& host, uint16_t port,
               int max_retries = 10, int retry_delay_ms = 500);

// Close socket fd.
void close_fd(int fd);

// Parse "host:port" string.
bool parse_host_port(const std::string& s, std::string& host, uint16_t& port);

}  // namespace eigenix::dist

#endif  // EIGENIX_DIST_NET_HPP

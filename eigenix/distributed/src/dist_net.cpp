#include "dist_net.hpp"

#include <arpa/inet.h>
#include <cerrno>
#include <cstdio>
#include <cstring>
#include <netdb.h>
#include <netinet/tcp.h>
#include <sys/socket.h>
#include <unistd.h>

#ifdef __APPLE__
#include <sys/types.h>
#endif

namespace eigenix::dist {

bool send_all(int fd, const void* buf, size_t len) {
    const uint8_t* ptr = static_cast<const uint8_t*>(buf);
    size_t remaining = len;
    while (remaining > 0) {
#ifdef __APPLE__
        ssize_t sent = ::send(fd, ptr, remaining, 0);
#else
        ssize_t sent = ::send(fd, ptr, remaining, MSG_NOSIGNAL);
#endif
        if (sent <= 0) return false;
        ptr += sent;
        remaining -= static_cast<size_t>(sent);
    }
    return true;
}

bool recv_all(int fd, void* buf, size_t len) {
    uint8_t* ptr = static_cast<uint8_t*>(buf);
    size_t remaining = len;
    while (remaining > 0) {
        ssize_t got = ::recv(fd, ptr, remaining, 0);
        if (got <= 0) return false;
        ptr += got;
        remaining -= static_cast<size_t>(got);
    }
    return true;
}

bool send_msg(int fd, MsgType type, const void* payload, uint64_t payload_len) {
    MsgHeader hdr{};
    hdr.magic = PROTOCOL_MAGIC;
    hdr.msg_type = static_cast<uint32_t>(type);
    hdr.payload_len = payload_len;
    if (!send_all(fd, &hdr, sizeof(hdr))) return false;
    if (payload_len > 0 && payload) {
        if (!send_all(fd, payload, static_cast<size_t>(payload_len))) return false;
    }
    return true;
}

bool send_msg_header(int fd, MsgType type, uint64_t payload_len) {
    MsgHeader hdr{};
    hdr.magic = PROTOCOL_MAGIC;
    hdr.msg_type = static_cast<uint32_t>(type);
    hdr.payload_len = payload_len;
    return send_all(fd, &hdr, sizeof(hdr));
}

bool recv_msg_header(int fd, MsgHeader& hdr) {
    if (!recv_all(fd, &hdr, sizeof(hdr))) return false;
    if (hdr.magic != PROTOCOL_MAGIC) {
        std::fprintf(stderr, "[NET] Bad magic: 0x%08X\n", hdr.magic);
        return false;
    }
    return true;
}

bool recv_msg(int fd, MsgHeader& hdr, std::vector<uint8_t>& out_payload) {
    if (!recv_all(fd, &hdr, sizeof(hdr))) return false;
    if (hdr.magic != PROTOCOL_MAGIC) {
        std::fprintf(stderr, "[NET] Bad magic: 0x%08X\n", hdr.magic);
        return false;
    }
    out_payload.resize(hdr.payload_len);
    if (hdr.payload_len > 0) {
        if (!recv_all(fd, out_payload.data(), hdr.payload_len)) return false;
    }
    return true;
}

int make_listener(uint16_t port, int backlog) {
    int fd = ::socket(AF_INET, SOCK_STREAM, 0);
    if (fd < 0) { perror("socket"); return -1; }

    int opt = 1;
    ::setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));

#ifdef __APPLE__
    ::setsockopt(fd, SOL_SOCKET, SO_NOSIGPIPE, &opt, sizeof(opt));
#endif

    sockaddr_in addr{};
    addr.sin_family = AF_INET;
    addr.sin_addr.s_addr = INADDR_ANY;
    addr.sin_port = htons(port);

    if (::bind(fd, reinterpret_cast<sockaddr*>(&addr), sizeof(addr)) < 0) {
        perror("bind");
        ::close(fd);
        return -1;
    }
    if (::listen(fd, backlog) < 0) {
        perror("listen");
        ::close(fd);
        return -1;
    }
    return fd;
}

int accept_one(int listen_fd) {
    sockaddr_in client{};
    socklen_t len = sizeof(client);
    int fd = ::accept(listen_fd, reinterpret_cast<sockaddr*>(&client), &len);
    if (fd < 0) { perror("accept"); return -1; }

    int opt = 1;
    ::setsockopt(fd, IPPROTO_TCP, TCP_NODELAY, &opt, sizeof(opt));
#ifdef __APPLE__
    ::setsockopt(fd, SOL_SOCKET, SO_NOSIGPIPE, &opt, sizeof(opt));
#endif
    return fd;
}

int connect_to(const std::string& host, uint16_t port,
               int max_retries, int retry_delay_ms) {
    sockaddr_in addr{};
    addr.sin_family = AF_INET;
    addr.sin_port = htons(port);

    if (inet_pton(AF_INET, host.c_str(), &addr.sin_addr) != 1) {
        // Try DNS resolution
        struct addrinfo hints{}, *res = nullptr;
        hints.ai_family = AF_INET;
        hints.ai_socktype = SOCK_STREAM;
        if (getaddrinfo(host.c_str(), nullptr, &hints, &res) != 0 || !res) {
            std::fprintf(stderr, "[NET] Cannot resolve host: %s\n", host.c_str());
            return -1;
        }
        addr.sin_addr = reinterpret_cast<sockaddr_in*>(res->ai_addr)->sin_addr;
        freeaddrinfo(res);
    }

    for (int attempt = 0; attempt < max_retries; ++attempt) {
        int fd = ::socket(AF_INET, SOCK_STREAM, 0);
        if (fd < 0) { perror("socket"); return -1; }

        if (::connect(fd, reinterpret_cast<sockaddr*>(&addr), sizeof(addr)) == 0) {
            int opt = 1;
            ::setsockopt(fd, IPPROTO_TCP, TCP_NODELAY, &opt, sizeof(opt));
#ifdef __APPLE__
            ::setsockopt(fd, SOL_SOCKET, SO_NOSIGPIPE, &opt, sizeof(opt));
#endif
            return fd;
        }
        ::close(fd);
        if (attempt + 1 < max_retries) {
            std::fprintf(stderr, "[NET] Connect to %s:%d failed (attempt %d/%d), retrying...\n",
                         host.c_str(), port, attempt + 1, max_retries);
            usleep(static_cast<useconds_t>(retry_delay_ms) * 1000);
        }
    }
    std::fprintf(stderr, "[NET] Failed to connect to %s:%d after %d attempts\n",
                 host.c_str(), port, max_retries);
    return -1;
}

void close_fd(int fd) {
    if (fd >= 0) ::close(fd);
}

bool parse_host_port(const std::string& s, std::string& host, uint16_t& port) {
    auto colon = s.rfind(':');
    if (colon == std::string::npos) return false;
    host = s.substr(0, colon);
    try {
        int p = std::stoi(s.substr(colon + 1));
        if (p <= 0 || p > 65535) return false;
        port = static_cast<uint16_t>(p);
    } catch (...) {
        return false;
    }
    return true;
}

}  // namespace eigenix::dist

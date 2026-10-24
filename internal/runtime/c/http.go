package c

const HTTPHeaderName = "trux_http_runtime.h"

const HTTPSource = `
#ifndef TRUX_HTTP_RUNTIME_H
#define TRUX_HTTP_RUNTIME_H

#if defined(__linux__)

#ifndef _GNU_SOURCE
#define _GNU_SOURCE
#endif

#include <arpa/inet.h>
#include <ctype.h>
#include <errno.h>
#include <fcntl.h>
#include <sys/types.h>
/* liburing.h includes <atomic> in C++ before its own extern "C" block.
   Wrapping the include in extern "C" makes those templates ill-formed. */
#include <liburing.h>
#include <poll.h>
#include <netinet/in.h>
#include <netinet/tcp.h>
#include <pthread.h>
#include <sched.h>
#include <signal.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <sys/time.h>
#include <sys/uio.h>
#include <unistd.h>

#ifndef SO_REUSEPORT
#define SO_REUSEPORT 15
#endif

#define RT_HTTP_MAX_HEADER_BYTES 65536
#define RT_HTTP_MAX_BODY_BYTES 10485760
#define RT_HTTP_READ_TIMEOUT_SECONDS 15
#define RT_HTTP_MAX_ROUTES 128
#define RT_HTTP_MAX_MIDDLEWARES 64
#define RT_HTTP_MAX_PARAMS 16
#define RT_HTTP_MAX_CONTEXT 32
#define RT_HTTP_MAX_RESP_HEADERS 32
#define RT_HTTP_MAX_HEADERS 64

#define RT_URING_ENTRIES 2048
#define RT_BUF_RING_SIZE 1024
#define RT_BUF_SIZE 4096
#define RT_BUF_GROUP_ID 1
#define RT_MAX_CONNS 65536
#define RT_HTTP_UDATA_ACCEPT ((uint64_t)-1)
#define RT_HTTP_UDATA_SEND (1ULL << 32)
#define RT_HTTP_UDATA_GPU (2ULL << 32)
#define RT_HTTP_UDATA_GPU_TIMEOUT (3ULL << 32)

typedef int64_t (*rt_http_handler_fn)(rt_context* trux_ctx, rt_arena* trux_result_arena, int64_t request);

typedef struct {
    const uint8_t* name;
    size_t name_len;
    const uint8_t* value;
    size_t value_len;
} rt_http_header_pair;

typedef struct {
    rt_string key;
    rt_string val;
} rt_http_kv;

typedef struct {
    rt_string method;
    rt_string path;
    rt_http_handler_fn handler;
    bool uses_gpu;
} rt_http_route_entry;

typedef struct {
    rt_string prefix;
    rt_http_handler_fn handler;
} rt_http_middleware_entry;

typedef struct {
    rt_http_route_entry routes[RT_HTTP_MAX_ROUTES];
    size_t route_count;
    rt_http_middleware_entry middlewares[RT_HTTP_MAX_MIDDLEWARES];
    size_t middleware_count;
    rt_http_handler_fn not_found_handler;
} rt_http_router;

static rt_http_router g_trux_http_router = {0};

typedef struct {
    int fd;
    uint8_t* buffer;
    size_t len;
    size_t cap;
    size_t header_scan_start;
    uint8_t* tx_buf;
    size_t tx_cap;
    bool tx_in_flight;
    bool close_after_tx;
    bool in_use;
    int next_ready_fd;
    bool in_ready_queue;
} rt_http_conn;

typedef struct {
    int fd;
    struct io_uring* ring;
    bool* need_submit;
    rt_http_conn* conn;
    const uint8_t* method;
    size_t method_len;
    const uint8_t* path;
    size_t path_len;
    const uint8_t* query;
    size_t query_len;
    const uint8_t* body;
    size_t body_len;
    rt_http_header_pair headers[RT_HTTP_MAX_HEADERS];
    size_t header_count;

    rt_http_kv params[RT_HTTP_MAX_PARAMS];
    size_t param_count;

    rt_http_kv context[RT_HTTP_MAX_CONTEXT];
    size_t context_count;

    rt_http_kv resp_headers[RT_HTTP_MAX_RESP_HEADERS];
    size_t resp_header_count;

    bool responded;
    bool close_after_response;
    bool parked;
    rt_http_handler_fn park_cont;
} rt_http_request;

typedef struct {
    rt_http_request* req;
    rt_http_handler_fn cont;
    rt_http_park_slot slot;
} rt_http_gpu_park;

static __thread rt_http_gpu_park* g_rt_http_parked;
static __thread rt_http_gpu_park g_rt_http_park_storage;
static __thread rt_http_request g_rt_http_park_req_storage;
static __thread rt_context g_rt_http_worker_ctx;
static __thread int g_rt_http_ready_head = -1;
static __thread int g_rt_http_ready_tail = -1;
static __thread int g_rt_http_in_continuation;
static __thread uint32_t g_rt_http_gpu_gen;
static __thread uint32_t g_rt_http_gpu_timer_gen;
static __thread bool g_rt_http_gpu_timer_armed;
static __thread struct __kernel_timespec g_rt_http_gpu_ts;

static RT_UNUSED const char* rt_http_reason(int64_t status) {
    switch (status) {
    case 200: return "OK";
    case 201: return "Created";
    case 202: return "Accepted";
    case 204: return "No Content";
    case 301: return "Moved Permanently";
    case 302: return "Found";
    case 304: return "Not Modified";
    case 400: return "Bad Request";
    case 401: return "Unauthorized";
    case 403: return "Forbidden";
    case 404: return "Not Found";
    case 405: return "Method Not Allowed";
    case 408: return "Request Timeout";
    case 413: return "Payload Too Large";
    case 500: return "Internal Server Error";
    case 502: return "Bad Gateway";
    case 503: return "Service Unavailable";
    default: return "OK";
    }
}

static RT_UNUSED int rt_http_ascii_lower(int c) {
    if (c >= 'A' && c <= 'Z') return c + ('a' - 'A');
    return c;
}

static RT_UNUSED bool rt_http_token_equal_ci(const uint8_t* left, size_t left_len, const uint8_t* right, size_t right_len) {
    if (left_len != right_len) return false;
    for (size_t i = 0; i < left_len; i++) {
        if (rt_http_ascii_lower(left[i]) != rt_http_ascii_lower(right[i])) return false;
    }
    return true;
}

static RT_UNUSED bool rt_http_string_equal_ci(rt_string left, const uint8_t* right, size_t right_len) {
    return rt_http_token_equal_ci(left.data, left.len, right, right_len);
}

static RT_UNUSED void rt_http_trim_ows(uint8_t** data, size_t* len) {
    while (*len > 0 && ((*data)[0] == ' ' || (*data)[0] == '\t')) {
        (*data)++;
        (*len)--;
    }
    while (*len > 0 && ((*data)[*len - 1] == ' ' || (*data)[*len - 1] == '\t')) {
        (*len)--;
    }
}

static RT_UNUSED bool rt_http_parse_content_length(const uint8_t* data, size_t len, size_t* out) {
    size_t value = 0;
    if (len == 0) return false;
    for (size_t i = 0; i < len; i++) {
        if (data[i] < '0' || data[i] > '9') return false;
        size_t digit = (size_t)(data[i] - '0');
        if (value > (SIZE_MAX - digit) / 10) return false;
        value = value * 10 + digit;
    }
    *out = value;
    return true;
}

static RT_UNUSED bool rt_http_contains_token_ci(const uint8_t* data, size_t len, const uint8_t* token, size_t token_len) {
    const uint8_t* cursor = data;
    size_t remaining = len;
    for (;;) {
        const uint8_t* comma = (const uint8_t*)memchr(cursor, ',', remaining);
        size_t part_len = comma == NULL ? remaining : (size_t)(comma - cursor);
        uint8_t* part = (uint8_t*)cursor;
        rt_http_trim_ows(&part, &part_len);
        if (rt_http_token_equal_ci(part, part_len, token, token_len)) return true;
        if (comma == NULL) return false;
        size_t consumed = (size_t)(comma + 1 - cursor);
        cursor = comma + 1;
        remaining -= consumed;
    }
}

static RT_UNUSED ssize_t rt_http_find_header_end(const uint8_t* buf, size_t len, size_t start) {
    if (len < 4) return -1;
    if (start > 3) start -= 3;
    for (size_t i = start; i + 3 < len; i++) {
        if (buf[i] == '\r' && buf[i + 1] == '\n' && buf[i + 2] == '\r' && buf[i + 3] == '\n') {
            return (ssize_t)(i + 4);
        }
    }
    return -1;
}

static RT_UNUSED bool rt_http_match_route(rt_string pattern, const uint8_t* path, size_t path_len, rt_http_kv* params, size_t* param_count, size_t max_params) {
    size_t pat_idx = 0;
    size_t path_idx = 0;
    *param_count = 0;

    while (pat_idx < pattern.len || path_idx < path_len) {
        while (pat_idx < pattern.len && pattern.data[pat_idx] == '/') pat_idx++;
        while (path_idx < path_len && path[path_idx] == '/') path_idx++;

        if (pat_idx >= pattern.len && path_idx >= path_len) return true;
        if (pat_idx >= pattern.len || path_idx >= path_len) return false;

        size_t pat_start = pat_idx;
        while (pat_idx < pattern.len && pattern.data[pat_idx] != '/') pat_idx++;
        size_t pat_seg_len = pat_idx - pat_start;

        size_t path_start = path_idx;
        while (path_idx < path_len && path[path_idx] != '/') path_idx++;
        size_t path_seg_len = path_idx - path_start;

        if (pattern.data[pat_start] == ':') {
            if (*param_count < max_params) {
                params[*param_count].key.data = pattern.data + pat_start + 1;
                params[*param_count].key.len = pat_seg_len - 1;
                params[*param_count].val.data = path + path_start;
                params[*param_count].val.len = path_seg_len;
                (*param_count)++;
            }
        } else {
            if (pat_seg_len != path_seg_len || memcmp(pattern.data + pat_start, path + path_start, pat_seg_len) != 0) {
                return false;
            }
        }
    }
    return true;
}

static RT_UNUSED bool rt_http_match_prefix(rt_string prefix, const uint8_t* path, size_t path_len) {
    if (prefix.len == 0 || prefix.data == NULL) return true;
    if (path_len < prefix.len) return false;
    if (memcmp(path, prefix.data, prefix.len) != 0) return false;
    if (path_len == prefix.len || path[prefix.len] == '/') return true;
    return false;
}

static RT_UNUSED bool rt_http_send_all_iov(int fd, struct iovec* iov, int iovcnt) {
    while (iovcnt > 0) {
        ssize_t n = writev(fd, iov, iovcnt);
        if (n < 0) {
            if (errno == EINTR) continue;
            return false;
        }
        if (n == 0) return false;
        size_t sent = (size_t)n;
        while (iovcnt > 0 && sent >= iov[0].iov_len) {
            sent -= iov[0].iov_len;
            iov++;
            iovcnt--;
        }
        if (iovcnt > 0 && sent > 0) {
            iov[0].iov_base = (uint8_t*)iov[0].iov_base + sent;
            iov[0].iov_len -= sent;
        }
    }
    return true;
}

static RT_UNUSED void rt_http_send_simple(int fd, int status, const char* body, bool close_conn) {
    char header[256];
    size_t body_len = strlen(body);
    int n = snprintf(header, sizeof(header),
        "HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nContent-Length: %zu\r\nConnection: %s\r\n\r\n",
        status, rt_http_reason(status), body_len, close_conn ? "close" : "keep-alive");
    if (n < 0 || (size_t)n >= sizeof(header)) return;
    struct iovec iov[2] = {
        {(void*)header, (size_t)n},
        {(void*)body, body_len},
    };
    (void)rt_http_send_all_iov(fd, iov, body_len == 0 ? 1 : 2);
}

static RT_UNUSED rt_string rt_http_clone_string(rt_arena* arena, const uint8_t* data, size_t len) {
    if (len == 0 || data == NULL) return RT_LIT(rt_string, NULL, 0);
    uint8_t* out = (uint8_t*)rt_arena_alloc(arena, len);
    memcpy(out, data, len);
    return RT_LIT(rt_string, out, len);
}

static RT_UNUSED rt_http_request* rt_http_request_from_handle(int64_t handle) {
    rt_http_request* req = (rt_http_request*)(intptr_t)handle;
    if (req == NULL) rt_runtime_fail("invalid http request");
    return req;
}

static RT_UNUSED void rt_http_register_route(rt_string method, rt_string path, rt_http_handler_fn handler) {
    if (g_trux_http_router.route_count >= RT_HTTP_MAX_ROUTES) {
        rt_runtime_fail("exceeded maximum number of http routes");
    }
    g_trux_http_router.routes[g_trux_http_router.route_count++] = RT_LIT(rt_http_route_entry, method, path, handler, false);
}

static RT_UNUSED void rt_http_register_route_gpu(rt_string method, rt_string path, rt_http_handler_fn handler) {
    if (g_trux_http_router.route_count >= RT_HTTP_MAX_ROUTES) {
        rt_runtime_fail("exceeded maximum number of http routes");
    }
    g_trux_http_router.routes[g_trux_http_router.route_count++] = RT_LIT(rt_http_route_entry, method, path, handler, true);
}

static RT_UNUSED void rt_http_register_middleware(rt_string prefix, rt_http_handler_fn handler) {
    if (g_trux_http_router.middleware_count >= RT_HTTP_MAX_MIDDLEWARES) {
        rt_runtime_fail("exceeded maximum number of http middlewares");
    }
    g_trux_http_router.middlewares[g_trux_http_router.middleware_count++] = RT_LIT(rt_http_middleware_entry, prefix, handler);
}

static RT_UNUSED void rt_http_register_not_found(rt_http_handler_fn handler) {
    g_trux_http_router.not_found_handler = handler;
}

static RT_UNUSED rt_string rt_http_method(rt_arena* arena, int64_t handle) {
    (void)arena;
    rt_http_request* req = rt_http_request_from_handle(handle);
    return RT_LIT(rt_string, req->method, req->method_len);
}

static RT_UNUSED rt_string rt_http_path(rt_arena* arena, int64_t handle) {
    (void)arena;
    rt_http_request* req = rt_http_request_from_handle(handle);
    return RT_LIT(rt_string, req->path, req->path_len);
}

static RT_UNUSED rt_string rt_http_query(rt_arena* arena, int64_t handle) {
    (void)arena;
    rt_http_request* req = rt_http_request_from_handle(handle);
    return RT_LIT(rt_string, req->query, req->query_len);
}

static RT_UNUSED rt_string rt_http_body(rt_arena* arena, int64_t handle) {
    (void)arena;
    rt_http_request* req = rt_http_request_from_handle(handle);
    return RT_LIT(rt_string, req->body, req->body_len);
}

static RT_UNUSED rt_string rt_http_header(rt_arena* arena, int64_t handle, rt_string name) {
    (void)arena;
    rt_http_request* req = rt_http_request_from_handle(handle);
    for (size_t i = 0; i < req->header_count; i++) {
        rt_http_header_pair header = req->headers[i];
        if (rt_http_string_equal_ci(name, header.name, header.name_len)) {
            return RT_LIT(rt_string, header.value, header.value_len);
        }
    }
    return RT_LIT(rt_string, NULL, 0);
}

static RT_UNUSED rt_string rt_http_param(rt_arena* arena, int64_t handle, rt_string name) {
    (void)arena;
    rt_http_request* req = rt_http_request_from_handle(handle);
    for (size_t i = 0; i < req->param_count; i++) {
        if (req->params[i].key.len == name.len && memcmp(req->params[i].key.data, name.data, name.len) == 0) {
            return RT_LIT(rt_string, req->params[i].val.data, req->params[i].val.len);
        }
    }
    return RT_LIT(rt_string, NULL, 0);
}

static RT_UNUSED void rt_http_set_context(int64_t handle, rt_string key, rt_string val) {
    rt_http_request* req = rt_http_request_from_handle(handle);
    for (size_t i = 0; i < req->context_count; i++) {
        if (req->context[i].key.len == key.len && memcmp(req->context[i].key.data, key.data, key.len) == 0) {
            req->context[i].val = val;
            return;
        }
    }
    if (req->context_count < RT_HTTP_MAX_CONTEXT) {
        req->context[req->context_count++] = RT_LIT(rt_http_kv, key, val);
    }
}

static RT_UNUSED rt_string rt_http_context(rt_arena* arena, int64_t handle, rt_string key) {
    rt_http_request* req = rt_http_request_from_handle(handle);
    for (size_t i = 0; i < req->context_count; i++) {
        if (req->context[i].key.len == key.len && memcmp(req->context[i].key.data, key.data, key.len) == 0) {
            return rt_http_clone_string(arena, req->context[i].val.data, req->context[i].val.len);
        }
    }
    return RT_LIT(rt_string, NULL, 0);
}

static RT_UNUSED void rt_http_set_header(int64_t handle, rt_string name, rt_string val) {
    rt_http_request* req = rt_http_request_from_handle(handle);
    if (req->resp_header_count < RT_HTTP_MAX_RESP_HEADERS) {
        req->resp_headers[req->resp_header_count++] = RT_LIT(rt_http_kv, name, val);
    }
}

static RT_UNUSED void rt_http_respond(int64_t handle, int64_t status, rt_string content_type, rt_string body) {
    rt_http_request* req = rt_http_request_from_handle(handle);
    if (req->responded) return;
    if (status < 100 || status > 999) status = 500;

    char header[4096];
    int pos = snprintf(header, sizeof(header),
        "HTTP/1.1 %lld %s\r\nContent-Type: %.*s\r\nContent-Length: %zu\r\nConnection: %s\r\n",
        (long long)status,
        rt_http_reason(status),
        (int)content_type.len,
        content_type.data == NULL ? (const uint8_t*)"" : content_type.data,
        body.len,
        req->close_after_response ? "close" : "keep-alive");

    if (pos > 0 && (size_t)pos < sizeof(header)) {
        for (size_t i = 0; i < req->resp_header_count; i++) {
            rt_http_kv kv = req->resp_headers[i];
            int added = snprintf(header + pos, sizeof(header) - (size_t)pos, "%.*s: %.*s\r\n",
                (int)kv.key.len, kv.key.data == NULL ? (const uint8_t*)"" : kv.key.data,
                (int)kv.val.len, kv.val.data == NULL ? (const uint8_t*)"" : kv.val.data);
            if (added > 0 && (size_t)(pos + added) < sizeof(header)) {
                pos += added;
            } else {
                break;
            }
        }
        if ((size_t)(pos + 2) < sizeof(header)) {
            header[pos++] = '\r';
            header[pos++] = '\n';
        }
    }

    if (pos < 0 || (size_t)pos >= sizeof(header)) {
        req->close_after_response = true;
        req->responded = true;
        return;
    }

    size_t body_len = (body.len > 0 && body.data != NULL) ? body.len : 0;
    size_t total_len = (size_t)pos + body_len;

    if (req->ring != NULL && req->conn != NULL && !req->conn->tx_in_flight && total_len <= 65536) {
        if (req->conn->tx_buf == NULL) {
            req->conn->tx_cap = 8192;
            while (req->conn->tx_cap < total_len) req->conn->tx_cap *= 2;
            req->conn->tx_buf = (uint8_t*)malloc(req->conn->tx_cap);
        } else if (total_len > req->conn->tx_cap) {
            size_t ncap = req->conn->tx_cap * 2;
            while (ncap < total_len) ncap *= 2;
            uint8_t* nbuf = (uint8_t*)realloc(req->conn->tx_buf, ncap);
            if (nbuf) {
                req->conn->tx_buf = nbuf;
                req->conn->tx_cap = ncap;
            }
        }
        if (req->conn->tx_buf != NULL && total_len <= req->conn->tx_cap) {
            memcpy(req->conn->tx_buf, header, (size_t)pos);
            if (body_len > 0) {
                memcpy(req->conn->tx_buf + (size_t)pos, body.data, body_len);
            }
            struct io_uring_sqe* sqe = io_uring_get_sqe(req->ring);
            if (!sqe) {
                io_uring_submit(req->ring);
                sqe = io_uring_get_sqe(req->ring);
            }
            if (sqe) {
                io_uring_prep_send(sqe, req->fd, req->conn->tx_buf, total_len, MSG_NOSIGNAL);
                sqe->user_data = RT_HTTP_UDATA_SEND | (uint64_t)(uint32_t)req->fd;
                req->conn->tx_in_flight = true;
                req->conn->close_after_tx = req->close_after_response;
                if (req->need_submit) *req->need_submit = true;
                req->responded = true;
                return;
            }
        }
    }

    if (req->ring != NULL && req->conn != NULL && req->conn->tx_in_flight) {
        io_uring_submit(req->ring);
        if (req->need_submit) *req->need_submit = false;
    }

    struct iovec iov[2] = {
        {(void*)header, (size_t)pos},
        {(void*)body.data, body.len},
    };
    int iovcnt = (body.len > 0 && body.data != NULL) ? 2 : 1;
    if (!rt_http_send_all_iov(req->fd, iov, iovcnt)) {
        req->close_after_response = true;
    }
    req->responded = true;
}

static RT_UNUSED void rt_http_text(int64_t handle, int64_t status, rt_string body) {
    rt_http_respond(handle, status, RT_LIT(rt_string, (const uint8_t*)"text/plain", 10), body);
}

static RT_UNUSED void rt_http_json(int64_t handle, int64_t status, rt_string body) {
    rt_http_respond(handle, status, RT_LIT(rt_string, (const uint8_t*)"application/json", 16), body);
}

static RT_UNUSED void rt_http_status(int64_t handle, int64_t status) {
    rt_http_respond(handle, status, RT_LIT(rt_string, (const uint8_t*)"text/plain", 10), RT_LIT(rt_string, NULL, 0));
}

static RT_UNUSED bool rt_http_parse_request_headers(uint8_t* buf, size_t len, size_t header_end, rt_http_request* req, size_t* body_len_out) {
    (void)len;
    size_t header_len = header_end - 2;
    uint8_t* start = buf;
    uint8_t* end = buf + header_len;
    uint8_t* line_end = (uint8_t*)memchr(start, '\r', (size_t)(end - start));
    if (line_end == NULL || line_end + 1 >= end || line_end[1] != '\n') return false;

    uint8_t* sp1 = (uint8_t*)memchr(start, ' ', (size_t)(line_end - start));
    if (sp1 == NULL) return false;
    uint8_t* sp2 = (uint8_t*)memchr(sp1 + 1, ' ', (size_t)(line_end - (sp1 + 1)));
    if (sp2 == NULL) return false;

    req->method = start;
    req->method_len = (size_t)(sp1 - start);
    uint8_t* target = sp1 + 1;
    size_t target_len = (size_t)(sp2 - target);
    uint8_t* question = (uint8_t*)memchr(target, '?', target_len);
    if (question == NULL) {
        req->path = target;
        req->path_len = target_len;
        req->query = target + target_len;
        req->query_len = 0;
    } else {
        req->path = target;
        req->path_len = (size_t)(question - target);
        req->query = question + 1;
        req->query_len = target_len - req->path_len - 1;
    }

    uint8_t* version = sp2 + 1;
    size_t version_len = (size_t)(line_end - version);
    if (!(version_len == 8 && memcmp(version, "HTTP/1.", 7) == 0 && (version[7] == '0' || version[7] == '1'))) {
        return false;
    }
    req->close_after_response = (version[7] == '0');

    size_t content_length = 0;
    bool has_content_length = false;
    bool has_transfer_encoding = false;
    req->header_count = 0;

    uint8_t* line = line_end + 2;
    while (line < end) {
        uint8_t* next = (uint8_t*)memchr(line, '\r', (size_t)(end - line));
        if (next == NULL || next + 1 > end || next[1] != '\n') return false;
        uint8_t* colon = (uint8_t*)memchr(line, ':', (size_t)(next - line));
        if (colon == NULL) return false;

        uint8_t* name = line;
        size_t name_len = (size_t)(colon - line);
        uint8_t* value = colon + 1;
        size_t value_len = (size_t)(next - value);
        rt_http_trim_ows(&value, &value_len);

        if (req->header_count < RT_HTTP_MAX_HEADERS) {
            req->headers[req->header_count++] = RT_LIT(rt_http_header_pair, name, name_len, value, value_len);
        }

        if (rt_http_token_equal_ci(name, name_len, (const uint8_t*)"Content-Length", 14)) {
            if (has_content_length || has_transfer_encoding) return false;
            if (!rt_http_parse_content_length(value, value_len, &content_length)) return false;
            has_content_length = true;
        } else if (rt_http_token_equal_ci(name, name_len, (const uint8_t*)"Transfer-Encoding", 17)) {
            if (has_transfer_encoding || has_content_length) return false;
            has_transfer_encoding = true;
            return false;
        } else if (rt_http_token_equal_ci(name, name_len, (const uint8_t*)"Connection", 10)) {
            if (rt_http_contains_token_ci(value, value_len, (const uint8_t*)"close", 5)) {
                req->close_after_response = true;
            }
        }
        line = next + 2;
    }

    if (!has_content_length) content_length = 0;
    *body_len_out = content_length;
    return true;
}

static RT_UNUSED const uint8_t* rt_http_dup_bytes(rt_arena* arena, const uint8_t* data, size_t len) {
    if (data == NULL || len == 0) return NULL;
    uint8_t* out = (uint8_t*)rt_arena_alloc(arena, len);
    memcpy(out, data, len);
    return out;
}

static RT_UNUSED void rt_http_arm_gpu_poll(struct io_uring* ring, bool* need_submit) {
    int fd = rt_gpu_event_fd();
    if (fd < 0 || ring == NULL) return;
    struct io_uring_sqe* sqe = io_uring_get_sqe(ring);
    if (sqe == NULL) {
        io_uring_submit(ring);
        sqe = io_uring_get_sqe(ring);
    }
    if (sqe == NULL) return;
    io_uring_prep_poll_add(sqe, fd, POLLIN);
    sqe->user_data = RT_HTTP_UDATA_GPU;
    if (need_submit != NULL) *need_submit = true;
}

static RT_UNUSED void rt_http_arm_gpu_timeout(struct io_uring* ring, bool* need_submit) {
    if (rt_gpu_event_fd() < 0 || ring == NULL || g_rt_http_gpu_timer_armed) return;
    g_rt_http_gpu_ts.tv_sec = 1;
    g_rt_http_gpu_ts.tv_nsec = 0;
    struct io_uring_sqe* sqe = io_uring_get_sqe(ring);
    if (sqe == NULL) {
        io_uring_submit(ring);
        sqe = io_uring_get_sqe(ring);
    }
    if (sqe == NULL) return;
    g_rt_http_gpu_timer_gen = g_rt_http_gpu_gen;
    g_rt_http_gpu_timer_armed = true;
    io_uring_prep_timeout(sqe, &g_rt_http_gpu_ts, 0, 0);
    sqe->user_data = RT_HTTP_UDATA_GPU_TIMEOUT | (uint64_t)g_rt_http_gpu_timer_gen;
    if (need_submit != NULL) *need_submit = true;
}

static RT_UNUSED void rt_http_trim_worker_pool(rt_context* ctx) {
    size_t kept = 0;
    rt_arena_chunk** link = &ctx->chunk_pool;
    while (*link != NULL) {
        rt_arena_chunk* c = *link;
        if (kept < 16 && c->cap <= 65536) {
            kept++;
            link = &c->next;
        } else {
            *link = c->next;
            free(c);
        }
    }
}

static RT_UNUSED void rt_http_enqueue_ready(rt_http_conn* conns, int fd) {
    if (conns == NULL || fd < 0 || fd >= RT_MAX_CONNS) return;
    rt_http_conn* c = &conns[fd];
    if (!c->in_use || c->len == 0 || c->in_ready_queue) return;
    c->next_ready_fd = -1;
    c->in_ready_queue = true;
    if (g_rt_http_ready_tail >= 0) {
        conns[g_rt_http_ready_tail].next_ready_fd = fd;
    } else {
        g_rt_http_ready_head = fd;
    }
    g_rt_http_ready_tail = fd;
}

static RT_UNUSED void rt_http_gpu_finish(int64_t handle, rt_http_handler_fn continuation) {
    if (g_rt_http_in_continuation) rt_runtime_fail("gpu.finish cannot be called from a GPU continuation");
    rt_http_request* req = rt_http_request_from_handle(handle);
    if (req->responded) rt_runtime_fail("gpu.finish cannot run after the handler responded");
    if (req->parked) rt_runtime_fail("gpu.finish can only be called once");
    if (continuation == NULL) rt_runtime_fail("gpu.finish continuation is missing");
    req->parked = true;
    req->park_cont = continuation;
    rt_gpu_submit_signal();
}

static RT_UNUSED void rt_http_retain_parked(rt_http_request* stack_req, rt_arena* stack_arena, rt_context* stack_ctx) {
    rt_http_gpu_park* park = &g_rt_http_park_storage;
    memset(park, 0, sizeof(*park));
    if (!rt_http_park_begin(&park->slot, stack_arena, stack_ctx)) {
        rt_runtime_fail("a GPU request is already in flight on this worker");
    }
    rt_arena* arena = park->slot.arena;
    rt_http_request* req = &g_rt_http_park_req_storage;
    *req = *stack_req;
    req->method = rt_http_dup_bytes(arena, stack_req->method, stack_req->method_len);
    req->path = rt_http_dup_bytes(arena, stack_req->path, stack_req->path_len);
    req->query = rt_http_dup_bytes(arena, stack_req->query, stack_req->query_len);
    req->body = rt_http_dup_bytes(arena, stack_req->body, stack_req->body_len);
    for (size_t i = 0; i < stack_req->header_count && i < RT_HTTP_MAX_HEADERS; i++) {
        rt_http_header_pair header = stack_req->headers[i];
        req->headers[i].name = rt_http_dup_bytes(arena, header.name, header.name_len);
        req->headers[i].name_len = header.name_len;
        req->headers[i].value = rt_http_dup_bytes(arena, header.value, header.value_len);
        req->headers[i].value_len = header.value_len;
    }
    for (size_t i = 0; i < stack_req->param_count && i < RT_HTTP_MAX_PARAMS; i++) {
        rt_http_kv kv = stack_req->params[i];
        req->params[i].key.data = rt_http_dup_bytes(arena, kv.key.data, kv.key.len);
        req->params[i].key.len = kv.key.len;
        req->params[i].val.data = rt_http_dup_bytes(arena, kv.val.data, kv.val.len);
        req->params[i].val.len = kv.val.len;
    }
    for (size_t i = 0; i < stack_req->context_count && i < RT_HTTP_MAX_CONTEXT; i++) {
        rt_http_kv kv = stack_req->context[i];
        req->context[i].key.data = rt_http_dup_bytes(arena, kv.key.data, kv.key.len);
        req->context[i].key.len = kv.key.len;
        req->context[i].val.data = rt_http_dup_bytes(arena, kv.val.data, kv.val.len);
        req->context[i].val.len = kv.val.len;
    }
    for (size_t i = 0; i < stack_req->resp_header_count && i < RT_HTTP_MAX_RESP_HEADERS; i++) {
        rt_http_kv kv = stack_req->resp_headers[i];
        req->resp_headers[i].key.data = rt_http_dup_bytes(arena, kv.key.data, kv.key.len);
        req->resp_headers[i].key.len = kv.key.len;
        req->resp_headers[i].val.data = rt_http_dup_bytes(arena, kv.val.data, kv.val.len);
        req->resp_headers[i].val.len = kv.val.len;
    }
    park->req = req;
    park->cont = stack_req->park_cont;
    g_rt_http_parked = park;
    rt_http_arm_gpu_timeout(stack_req->ring, stack_req->need_submit);
}

static RT_UNUSED void rt_http_dispatch(rt_http_request* req) {
    rt_arena arena;
    rt_arena_init_frame(&arena, &g_rt_http_worker_ctx);
    g_rt_http_worker_ctx.arena = &arena;

    int match_idx = -1;
    for (size_t i = 0; i < g_trux_http_router.route_count; i++) {
        rt_http_route_entry* r = &g_trux_http_router.routes[i];
        if (r->method.len > 0 && r->method.data != NULL) {
            if (!rt_http_string_equal_ci(r->method, req->method, req->method_len)) continue;
        }
        if (rt_http_match_route(r->path, req->path, req->path_len, req->params, &req->param_count, RT_HTTP_MAX_PARAMS)) {
            match_idx = (int)i;
            break;
        }
    }

    bool aborted = false;
    for (size_t i = 0; i < g_trux_http_router.middleware_count; i++) {
        rt_http_middleware_entry* m = &g_trux_http_router.middlewares[i];
        if (rt_http_match_prefix(m->prefix, req->path, req->path_len)) {
            int64_t ret = m->handler(&g_rt_http_worker_ctx, &arena, (int64_t)(intptr_t)req);
            if (ret != 0 || req->responded) {
                aborted = true;
                break;
            }
        }
    }

    if (!aborted && !req->responded) {
        if (match_idx >= 0) {
            g_trux_http_router.routes[match_idx].handler(&g_rt_http_worker_ctx, &arena, (int64_t)(intptr_t)req);
        } else if (g_trux_http_router.not_found_handler != NULL) {
            g_trux_http_router.not_found_handler(&g_rt_http_worker_ctx, &arena, (int64_t)(intptr_t)req);
        } else {
            rt_http_send_simple(req->fd, 404, "not found\n", req->close_after_response);
            req->responded = true;
        }
    }

    if (req->parked) {
        if (req->responded) rt_runtime_fail("gpu.finish and a response cannot both happen in the same handler");
        rt_http_retain_parked(req, &arena, &g_rt_http_worker_ctx);
        return;
    }

    if (!req->responded) {
        rt_http_send_simple(req->fd, 500, "handler did not respond\n", req->close_after_response);
        req->responded = true;
    }

    rt_arena_deinit(&arena);
    rt_http_trim_worker_pool(&g_rt_http_worker_ctx);
}

typedef struct {
    char host[128];
    int port;
    int worker_index;
} rt_http_thread_args;

static RT_UNUSED void rt_http_drain_conn(rt_http_conn* conns, rt_http_conn* c, int client_fd, struct io_uring* ring, bool* need_submit) {
    if (c == NULL || !c->in_use) return;
    for (;;) {
        ssize_t hdr_end = rt_http_find_header_end(c->buffer, c->len, c->header_scan_start);
        if (hdr_end < 0) {
            if (c->len > 3) c->header_scan_start = c->len - 3;
            return;
        }

        rt_http_request req;
        memset(&req, 0, sizeof(req));
        req.fd = client_fd;
        req.ring = ring;
        req.need_submit = need_submit;
        req.conn = c;

        size_t content_length = 0;
        if (!rt_http_parse_request_headers(c->buffer, c->len, (size_t)hdr_end, &req, &content_length)) {
            rt_http_send_simple(client_fd, 400, "bad request\n", true);
            shutdown(client_fd, SHUT_RDWR);
            close(client_fd);
            c->in_use = false;
            c->len = 0;
            return;
        }

        if (content_length > RT_HTTP_MAX_BODY_BYTES) {
            rt_http_send_simple(client_fd, 413, "payload too large\n", true);
            shutdown(client_fd, SHUT_RDWR);
            close(client_fd);
            c->in_use = false;
            c->len = 0;
            return;
        }

        if (c->len < (size_t)hdr_end + content_length) return;

        if (g_rt_http_parked != NULL) {
            bool matches_gpu = false;
            for (size_t i = 0; i < g_trux_http_router.route_count; i++) {
                rt_http_route_entry* r = &g_trux_http_router.routes[i];
                if (r->uses_gpu && rt_http_token_equal_ci(r->method.data, r->method.len, req.method, req.method_len) &&
                    rt_http_match_route(r->path, req.path, req.path_len, req.params, &req.param_count, RT_HTTP_MAX_PARAMS)) {
                    matches_gpu = true;
                    break;
                }
            }
            if (matches_gpu) {
                rt_http_enqueue_ready(conns, client_fd);
                return;
            }
        }

        req.body = c->buffer + hdr_end;
        req.body_len = content_length;
        rt_http_dispatch(&req);

        size_t consumed = (size_t)hdr_end + content_length;
        if (consumed < c->len) {
            memmove(c->buffer, c->buffer + consumed, c->len - consumed);
            c->len -= consumed;
            c->header_scan_start = 0;
        } else {
            c->len = 0;
            c->header_scan_start = 0;
        }

        if (req.parked) {
            if (c->len > 0) {
                rt_http_enqueue_ready(conns, client_fd);
            }
            return;
        }

        if (req.close_after_response) {
            if (c->tx_in_flight) {
                c->close_after_tx = true;
                return;
            }
            shutdown(client_fd, SHUT_RDWR);
            close(client_fd);
            c->in_use = false;
            c->len = 0;
            return;
        }

        if (c->len == 0) return;
    }
}

static RT_UNUSED void rt_http_drain_ready(rt_http_conn* conns, struct io_uring* ring, bool* need_submit) {
    while (g_rt_http_ready_head >= 0 && g_rt_http_parked == NULL) {
        int fd = g_rt_http_ready_head;
        rt_http_conn* c = &conns[fd];
        g_rt_http_ready_head = c->next_ready_fd;
        if (g_rt_http_ready_head < 0) {
            g_rt_http_ready_tail = -1;
        }
        c->next_ready_fd = -1;
        c->in_ready_queue = false;
        if (!c->in_use || c->len == 0) continue;
        rt_http_drain_conn(conns, c, fd, ring, need_submit);
    }
}

static RT_UNUSED void rt_http_gpu_release(struct io_uring* ring, rt_http_conn* conns, bool* need_submit, int failed) {
    rt_http_gpu_park* park = g_rt_http_parked;
    if (park == NULL) {
        rt_http_arm_gpu_poll(ring, need_submit);
        return;
    }
    g_rt_http_parked = NULL;
    g_rt_http_gpu_gen++;
    g_rt_http_gpu_timer_armed = false;
    rt_gpu_clear_busy();
    rt_http_request* req = park->req;
    int fd = req->fd;
    int alive = fd >= 0 && fd < RT_MAX_CONNS && conns[fd].in_use;
    if (alive) {
        req->ring = ring;
        req->need_submit = need_submit;
        req->conn = &conns[fd];
        if (!failed && park->cont != NULL) {
            g_rt_http_in_continuation = 1;
            park->cont(park->slot.ctx, park->slot.arena, (int64_t)(intptr_t)req);
            g_rt_http_in_continuation = 0;
        }
        if (!req->responded) {
            const char* body = failed ? "gpu request failed\n" : "handler did not respond\n";
            rt_http_send_simple(fd, 500, body, req->close_after_response);
            req->responded = true;
        }
        if (req->close_after_response) {
            rt_http_conn* c = &conns[fd];
            if (c->tx_in_flight) {
                c->close_after_tx = true;
            } else if (c->in_use) {
                shutdown(fd, SHUT_RDWR);
                close(fd);
                c->in_use = false;
                c->len = 0;
            }
        }
    }
    if (park->slot.arena != NULL && park->slot.ctx != NULL) {
        rt_arena_deinit(park->slot.arena);
        rt_arena_chunk* c = park->slot.ctx->chunk_pool;
        while (c != NULL) {
            rt_arena_chunk* next = c->next;
            c->next = g_rt_http_worker_ctx.chunk_pool;
            g_rt_http_worker_ctx.chunk_pool = c;
            c = next;
        }
        park->slot.ctx->chunk_pool = NULL;
        rt_http_trim_worker_pool(&g_rt_http_worker_ctx);
    }
    rt_http_park_complete(&park->slot, 1);
    rt_http_arm_gpu_poll(ring, need_submit);
    rt_http_drain_ready(conns, ring, need_submit);
}

static RT_UNUSED void* rt_http_worker_thread(void* arg) {
    rt_http_thread_args* targs = (rt_http_thread_args*)arg;
    int worker_idx = targs->worker_index;
#if defined(__linux__)
    cpu_set_t cpuset;
    CPU_ZERO(&cpuset);
    long nproc = sysconf(_SC_NPROCESSORS_ONLN);
    if (nproc > 0) {
        CPU_SET((size_t)(worker_idx % nproc), &cpuset);
        (void)pthread_setaffinity_np(pthread_self(), sizeof(cpu_set_t), &cpuset);
    }
#endif

    int listen_fd = socket(AF_INET, SOCK_STREAM, 0);
    if (listen_fd < 0) rt_runtime_fail("worker socket creation failed");

    int one = 1;
    (void)setsockopt(listen_fd, SOL_SOCKET, SO_REUSEPORT, &one, sizeof(one));
    (void)setsockopt(listen_fd, SOL_SOCKET, SO_REUSEADDR, &one, sizeof(one));
    (void)setsockopt(listen_fd, IPPROTO_TCP, TCP_NODELAY, &one, sizeof(one));

    struct sockaddr_in addr;
    memset(&addr, 0, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_port = htons((uint16_t)targs->port);
    if (inet_pton(AF_INET, targs->host, &addr.sin_addr) != 1) {
        rt_runtime_fail("invalid http host IP address");
    }

    if (bind(listen_fd, (struct sockaddr*)&addr, sizeof(addr)) != 0) {
        rt_runtime_fail("worker bind failed");
    }
    if (listen(listen_fd, 4096) != 0) {
        rt_runtime_fail("worker listen failed");
    }

    struct io_uring ring;
    struct io_uring_params params;
    memset(&params, 0, sizeof(params));
#ifdef IORING_SETUP_COOP_TASKRUN
    params.flags |= IORING_SETUP_COOP_TASKRUN;
#endif
#ifdef IORING_SETUP_SINGLE_ISSUER
    params.flags |= IORING_SETUP_SINGLE_ISSUER;
#endif
#ifdef IORING_SETUP_DEFER_TASKRUN
    params.flags |= IORING_SETUP_DEFER_TASKRUN;
#endif
    int ret = io_uring_queue_init_params(RT_URING_ENTRIES, &ring, &params);
    if (ret < 0) {
        ret = io_uring_queue_init(RT_URING_ENTRIES, &ring, 0);
        if (ret < 0) rt_runtime_fail("io_uring_queue_init failed");
    }

    uint8_t* buf_storage = NULL;
    if (posix_memalign((void**)&buf_storage, 4096, RT_BUF_RING_SIZE * RT_BUF_SIZE) != 0) {
        rt_runtime_fail("posix_memalign for buf ring failed");
    }

    int err = 0;
    struct io_uring_buf_ring* br = io_uring_setup_buf_ring(&ring, RT_BUF_RING_SIZE, RT_BUF_GROUP_ID, 0, &err);
    if (!br) rt_runtime_fail("io_uring_setup_buf_ring failed");

    for (int i = 0; i < RT_BUF_RING_SIZE; i++) {
        io_uring_buf_ring_add(br, buf_storage + i * RT_BUF_SIZE, RT_BUF_SIZE, i, io_uring_buf_ring_mask(RT_BUF_RING_SIZE), i);
    }
    io_uring_buf_ring_advance(br, RT_BUF_RING_SIZE);

    struct io_uring_sqe* sqe = io_uring_get_sqe(&ring);
    io_uring_prep_multishot_accept(sqe, listen_fd, NULL, NULL, 0);
    sqe->user_data = (uint64_t)-1;
    io_uring_submit(&ring);

    rt_http_conn* conns = (rt_http_conn*)calloc(RT_MAX_CONNS, sizeof(rt_http_conn));
    if (!conns) rt_runtime_fail("failed to allocate connection pool");

    rt_gpu_worker_init();
    {
        bool arm_gpu = false;
        rt_http_arm_gpu_poll(&ring, &arm_gpu);
        if (arm_gpu) io_uring_submit(&ring);
    }

    for (;;) {
        if (g_rt_http_parked != NULL) {
            int query = rt_gpu_query_job();
            if (query == 2) {
                bool need_sub = false;
                rt_http_gpu_release(&ring, conns, &need_sub, 0);
                if (need_sub) io_uring_submit(&ring);
            } else if (query == 1) {
                rt_gpu_reset_stream();
                bool need_sub = false;
                rt_http_gpu_release(&ring, conns, &need_sub, 1);
                if (need_sub) io_uring_submit(&ring);
            }
        }

        struct io_uring_cqe* cqe;
        ret = io_uring_wait_cqe(&ring, &cqe);
        if (ret < 0) {
            if (errno == EINTR) continue;
            break;
        }

        unsigned head;
        bool need_submit = false;

        io_uring_for_each_cqe(&ring, head, cqe) {
            uint64_t udata = cqe->user_data;
            if (udata == RT_HTTP_UDATA_GPU) {
                io_uring_cq_advance(&ring, 1);
                rt_gpu_drain_eventfd();
                int query = rt_gpu_query_job();
                if (g_rt_http_parked == NULL) {
                    if (query == 1) rt_gpu_reset_stream();
                    rt_http_arm_gpu_poll(&ring, &need_submit);
                } else if (query == 0) {
                    rt_http_arm_gpu_poll(&ring, &need_submit);
                } else if (query == 1) {
                    rt_gpu_reset_stream();
                    rt_http_gpu_release(&ring, conns, &need_submit, 1);
                } else {
                    rt_http_gpu_release(&ring, conns, &need_submit, 0);
                }
            } else if ((udata & 0xFFFFFFFF00000000ULL) == RT_HTTP_UDATA_GPU_TIMEOUT) {
                io_uring_cq_advance(&ring, 1);
                g_rt_http_gpu_timer_armed = false;
                uint32_t gen = (uint32_t)udata;
                if (g_rt_http_parked != NULL && gen == g_rt_http_gpu_gen) {
                    int query = rt_gpu_query_job();
                    if (query == 0) {
                        rt_http_arm_gpu_timeout(&ring, &need_submit);
                    } else if (query == 1) {
                        rt_gpu_reset_stream();
                        rt_http_gpu_release(&ring, conns, &need_submit, 1);
                    } else {
                        rt_http_gpu_release(&ring, conns, &need_submit, 0);
                    }
                }
            } else if (udata == RT_HTTP_UDATA_ACCEPT) {
                int client_fd = cqe->res;
                io_uring_cq_advance(&ring, 1);
                if (client_fd >= 0 && client_fd < RT_MAX_CONNS) {
                    int nodelay = 1;
                    (void)setsockopt(client_fd, IPPROTO_TCP, TCP_NODELAY, &nodelay, sizeof(nodelay));

                    rt_http_conn* c = &conns[client_fd];
                    c->fd = client_fd;
                    c->len = 0;
                    c->header_scan_start = 0;
                    c->tx_in_flight = false;
                    c->close_after_tx = false;
                    c->in_use = true;
                    if (c->buffer == NULL) {
                        c->cap = 8192;
                        c->buffer = (uint8_t*)malloc(c->cap);
                    }

                    struct io_uring_sqe* rsqe = io_uring_get_sqe(&ring);
                    if (!rsqe) {
                        io_uring_submit(&ring);
                        rsqe = io_uring_get_sqe(&ring);
                    }
                    if (rsqe) {
                        io_uring_prep_recv_multishot(rsqe, client_fd, NULL, 0, 0);
                        rsqe->flags |= IOSQE_BUFFER_SELECT;
                        rsqe->buf_group = RT_BUF_GROUP_ID;
                        rsqe->user_data = (uint64_t)client_fd;
                        need_submit = true;
                    }
                }
            } else if ((udata & RT_HTTP_UDATA_SEND) != 0) {
                int client_fd = (int)(udata & 0xFFFFFFFFULL);
                int res = cqe->res;
                io_uring_cq_advance(&ring, 1);
                if (client_fd >= 0 && client_fd < RT_MAX_CONNS) {
                    rt_http_conn* c = &conns[client_fd];
                    c->tx_in_flight = false;
                    if (res <= 0 || c->close_after_tx) {
                        if (c->in_use) {
                            if (c->close_after_tx) shutdown(client_fd, SHUT_RDWR);
                            close(client_fd);
                            c->in_use = false;
                            c->len = 0;
                            c->close_after_tx = false;
                        }
                    }
                }
            } else {
                int client_fd = (int)udata;
                int res = cqe->res;
                unsigned flags = cqe->flags;
                if (client_fd >= 0 && client_fd < RT_MAX_CONNS) {
                    rt_http_conn* c = &conns[client_fd];
                    if (res <= 0) {
                        io_uring_cq_advance(&ring, 1);
                        if (res == -ENOBUFS && c->in_use) {
                            struct io_uring_sqe* rsqe = io_uring_get_sqe(&ring);
                            if (!rsqe) {
                                io_uring_submit(&ring);
                                rsqe = io_uring_get_sqe(&ring);
                            }
                            if (rsqe) {
                                io_uring_prep_recv_multishot(rsqe, client_fd, NULL, 0, 0);
                                rsqe->flags |= IOSQE_BUFFER_SELECT;
                                rsqe->buf_group = RT_BUF_GROUP_ID;
                                rsqe->user_data = (uint64_t)client_fd;
                                need_submit = true;
                            }
                        } else if (c->in_use) {
                            close(client_fd);
                            c->in_use = false;
                            c->len = 0;
                        }
                    } else {
                        int bid = flags >> IORING_CQE_BUFFER_SHIFT;
                        uint8_t* in_data = buf_storage + bid * RT_BUF_SIZE;
                        size_t bytes = (size_t)res;

                        if (c->len + bytes > c->cap) {
                            size_t ncap = c->cap * 2;
                            while (ncap < c->len + bytes) ncap *= 2;
                            uint8_t* nbuf = (uint8_t*)realloc(c->buffer, ncap);
                            if (nbuf) {
                                c->buffer = nbuf;
                                c->cap = ncap;
                            }
                        }

                        if (c->len + bytes <= c->cap) {
                            memcpy(c->buffer + c->len, in_data, bytes);
                            c->len += bytes;
                        }

                        io_uring_buf_ring_add(br, in_data, RT_BUF_SIZE, bid, io_uring_buf_ring_mask(RT_BUF_RING_SIZE), 0);
                        io_uring_buf_ring_advance(br, 1);
                        io_uring_cq_advance(&ring, 1);

                        rt_http_drain_conn(conns, c, client_fd, &ring, &need_submit);

                        if (c->in_use && !(flags & IORING_CQE_F_MORE)) {
                            struct io_uring_sqe* rsqe = io_uring_get_sqe(&ring);
                            if (!rsqe) {
                                io_uring_submit(&ring);
                                rsqe = io_uring_get_sqe(&ring);
                            }
                            if (rsqe) {
                                io_uring_prep_recv_multishot(rsqe, client_fd, NULL, 0, 0);
                                rsqe->flags |= IOSQE_BUFFER_SELECT;
                                rsqe->buf_group = RT_BUF_GROUP_ID;
                                rsqe->user_data = (uint64_t)client_fd;
                                need_submit = true;
                            }
                        }
                    }
                } else {
                    io_uring_cq_advance(&ring, 1);
                }
            }
        }
        if (need_submit) {
            io_uring_submit(&ring);
        }
    }

    free(conns);
    io_uring_free_buf_ring(&ring, br, RT_BUF_RING_SIZE, RT_BUF_GROUP_ID);
    free(buf_storage);
    io_uring_queue_exit(&ring);
    close(listen_fd);
    return NULL;
}

static RT_UNUSED void rt_http_serve(rt_string host, int64_t port, int64_t workers) {
    if (port < 1 || port > 65535) rt_runtime_fail("http port must be between 1 and 65535");
    if (workers <= 0) {
        cpu_set_t cpuset;
        CPU_ZERO(&cpuset);
        if (sched_getaffinity(0, sizeof(cpuset), &cpuset) == 0) {
            workers = CPU_COUNT(&cpuset);
        }
        if (workers <= 0) {
            workers = sysconf(_SC_NPROCESSORS_ONLN);
        }
        if (workers < 1) workers = 1;
    }
    if (workers > 1024) workers = 1024;

#ifdef SIGPIPE
    signal(SIGPIPE, SIG_IGN);
#endif

    rt_http_thread_args args;
    memset(&args, 0, sizeof(args));
    args.port = (int)port;
    if (host.data != NULL && host.len > 0) {
        size_t len = host.len < sizeof(args.host) - 1 ? host.len : sizeof(args.host) - 1;
        memcpy(args.host, host.data, len);
        args.host[len] = '\0';
    } else {
        strcpy(args.host, "0.0.0.0");
    }

    pthread_t* threads = (pthread_t*)malloc((size_t)workers * sizeof(pthread_t));
    if (!threads) rt_runtime_fail("allocation failed for worker threads");
    rt_http_thread_args* thread_args = (rt_http_thread_args*)malloc((size_t)workers * sizeof(rt_http_thread_args));
    if (!thread_args) rt_runtime_fail("allocation failed for worker thread args");

    rt_gpu_publish_workers();
    for (int64_t i = 0; i < workers; i++) {
        thread_args[i] = args;
        thread_args[i].worker_index = (int)i;
        if (pthread_create(&threads[i], NULL, rt_http_worker_thread, &thread_args[i]) != 0) {
            rt_runtime_fail("failed to spawn worker thread");
        }
    }

    for (int64_t i = 0; i < workers; i++) {
        (void)pthread_join(threads[i], NULL);
    }
    free(thread_args);
    free(threads);
}

static RT_UNUSED void rt_http_listen(int64_t port) {
    rt_http_serve(RT_LIT(rt_string, (const uint8_t*)"0.0.0.0", 7), port, 0);
}

#else

typedef int64_t (*rt_http_handler_fn)(rt_context* trux_ctx, rt_arena* trux_result_arena, int64_t request);

static RT_UNUSED void rt_http_register_route(rt_string method, rt_string path, rt_http_handler_fn handler) { (void)method; (void)path; (void)handler; rt_runtime_fail("http is supported on Linux only"); }
static RT_UNUSED void rt_http_register_route_gpu(rt_string method, rt_string path, rt_http_handler_fn handler) { (void)method; (void)path; (void)handler; rt_runtime_fail("http is supported on Linux only"); }
static RT_UNUSED void rt_http_register_middleware(rt_string prefix, rt_http_handler_fn handler) { (void)prefix; (void)handler; rt_runtime_fail("http is supported on Linux only"); }
static RT_UNUSED void rt_http_register_not_found(rt_http_handler_fn handler) { (void)handler; rt_runtime_fail("http is supported on Linux only"); }
static RT_UNUSED void rt_http_listen(int64_t port) { (void)port; rt_runtime_fail("http is supported on Linux only"); }
static RT_UNUSED void rt_http_gpu_finish(int64_t handle, rt_http_handler_fn handler) { (void)handle; (void)handler; rt_runtime_fail("http is supported on Linux only"); }
static RT_UNUSED void rt_http_serve(rt_string host, int64_t port, int64_t workers) { (void)host; (void)port; (void)workers; rt_runtime_fail("http is supported on Linux only"); }
static RT_UNUSED rt_string rt_http_param(rt_arena* arena, int64_t handle, rt_string name) { (void)arena; (void)handle; (void)name; rt_runtime_fail("http is supported on Linux only"); return RT_LIT(rt_string, NULL, 0); }
static RT_UNUSED void rt_http_set_context(int64_t handle, rt_string key, rt_string val) { (void)handle; (void)key; (void)val; rt_runtime_fail("http is supported on Linux only"); }
static RT_UNUSED rt_string rt_http_context(rt_arena* arena, int64_t handle, rt_string key) { (void)arena; (void)handle; (void)key; rt_runtime_fail("http is supported on Linux only"); return RT_LIT(rt_string, NULL, 0); }
static RT_UNUSED void rt_http_set_header(int64_t handle, rt_string name, rt_string val) { (void)handle; (void)name; (void)val; rt_runtime_fail("http is supported on Linux only"); }
static RT_UNUSED void rt_http_text(int64_t handle, int64_t status, rt_string body) { (void)handle; (void)status; (void)body; rt_runtime_fail("http is supported on Linux only"); }
static RT_UNUSED void rt_http_json(int64_t handle, int64_t status, rt_string body) { (void)handle; (void)status; (void)body; rt_runtime_fail("http is supported on Linux only"); }
static RT_UNUSED void rt_http_status(int64_t handle, int64_t status) { (void)handle; (void)status; rt_runtime_fail("http is supported on Linux only"); }
static RT_UNUSED rt_string rt_http_method(rt_arena* arena, int64_t handle) { (void)arena; (void)handle; rt_runtime_fail("http is supported on Linux only"); return RT_LIT(rt_string, NULL, 0); }
static RT_UNUSED rt_string rt_http_path(rt_arena* arena, int64_t handle) { (void)arena; (void)handle; rt_runtime_fail("http is supported on Linux only"); return RT_LIT(rt_string, NULL, 0); }
static RT_UNUSED rt_string rt_http_query(rt_arena* arena, int64_t handle) { (void)arena; (void)handle; rt_runtime_fail("http is supported on Linux only"); return RT_LIT(rt_string, NULL, 0); }
static RT_UNUSED rt_string rt_http_header(rt_arena* arena, int64_t handle, rt_string name) { (void)arena; (void)handle; (void)name; rt_runtime_fail("http is supported on Linux only"); return RT_LIT(rt_string, NULL, 0); }
static RT_UNUSED rt_string rt_http_body(rt_arena* arena, int64_t handle) { (void)arena; (void)handle; rt_runtime_fail("http is supported on Linux only"); return RT_LIT(rt_string, NULL, 0); }
static RT_UNUSED void rt_http_respond(int64_t handle, int64_t status, rt_string content_type, rt_string body) { (void)handle; (void)status; (void)content_type; (void)body; rt_runtime_fail("http is supported on Linux only"); }

#endif

#endif
`

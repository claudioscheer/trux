package c

const HTTPHeaderName = "trux_http_runtime.h"

const HTTPSource = `
#ifndef TRUX_HTTP_RUNTIME_H
#define TRUX_HTTP_RUNTIME_H

#ifndef _WIN32

#include <arpa/inet.h>
#include <ctype.h>
#include <errno.h>
#include <netinet/in.h>
#include <netinet/tcp.h>
#include <pthread.h>
#include <signal.h>
#include <sys/socket.h>
#include <sys/time.h>
#include <sys/uio.h>
#include <unistd.h>

#define RT_HTTP_MAX_HEADER_BYTES 65536
#define RT_HTTP_MAX_BODY_BYTES 1048576
#define RT_HTTP_QUEUE_CAP 1024
#define RT_HTTP_READ_TIMEOUT_SECONDS 10

typedef int64_t (*rt_http_handler_fn)(rt_context* trux_ctx, rt_arena* trux_result_arena, int64_t request);

typedef struct {
    const uint8_t* name;
    size_t name_len;
    const uint8_t* value;
    size_t value_len;
} rt_http_header_pair;

typedef struct {
    int fd;
    uint8_t* buffer;
    size_t len;
    size_t cap;
    size_t header_scan_start;
    const uint8_t* method;
    size_t method_len;
    const uint8_t* path;
    size_t path_len;
    const uint8_t* query;
    size_t query_len;
    const uint8_t* body;
    size_t body_len;
    rt_http_header_pair* headers;
    size_t header_count;
    bool responded;
    bool close_after_response;
} rt_http_request;

typedef struct {
    int fds[RT_HTTP_QUEUE_CAP];
    size_t head;
    size_t tail;
    size_t len;
    pthread_mutex_t mutex;
    pthread_cond_t not_empty;
    pthread_cond_t not_full;
} rt_http_fd_queue;

typedef struct {
    rt_http_fd_queue* queue;
    rt_http_handler_fn handler;
} rt_http_worker_args;

static RT_UNUSED void rt_http_close_fd(int fd) {
    while (close(fd) != 0 && errno == EINTR) {
    }
}

static RT_UNUSED void rt_http_configure_client_socket(int fd) {
    int one = 1;
    (void)setsockopt(fd, IPPROTO_TCP, TCP_NODELAY, &one, sizeof(one));

    struct timeval timeout;
    timeout.tv_sec = RT_HTTP_READ_TIMEOUT_SECONDS;
    timeout.tv_usec = 0;
    (void)setsockopt(fd, SOL_SOCKET, SO_RCVTIMEO, &timeout, sizeof(timeout));
}

static RT_UNUSED bool rt_http_send_all(int fd, const uint8_t* data, size_t len) {
    size_t sent = 0;
    while (sent < len) {
        ssize_t n = send(fd, data + sent, len - sent, 0);
        if (n < 0) {
            if (errno == EINTR) {
                continue;
            }
            return false;
        }
        if (n == 0) {
            return false;
        }
        sent += (size_t)n;
    }
    return true;
}

static RT_UNUSED bool rt_http_send_cstr(int fd, const char* text) {
    return rt_http_send_all(fd, (const uint8_t*)text, strlen(text));
}

static RT_UNUSED bool rt_http_send_all_iov(int fd, struct iovec* iov, int iovcnt) {
    while (iovcnt > 0) {
        ssize_t n = writev(fd, iov, iovcnt);
        if (n < 0) {
            if (errno == EINTR) {
                continue;
            }
            return false;
        }
        if (n == 0) {
            return false;
        }
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

static RT_UNUSED const char* rt_http_reason(int64_t status) {
    switch (status) {
    case 200: return "OK";
    case 201: return "Created";
    case 204: return "No Content";
    case 400: return "Bad Request";
    case 404: return "Not Found";
    case 405: return "Method Not Allowed";
    case 413: return "Payload Too Large";
    case 500: return "Internal Server Error";
    default: return "OK";
    }
}

static RT_UNUSED void rt_http_send_simple(int fd, int status, const char* body, bool close_conn) {
    char header[256];
    size_t body_len = strlen(body);
    int n = snprintf(header, sizeof(header),
        "HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nContent-Length: %zu\r\nConnection: %s\r\n\r\n",
        status, rt_http_reason(status), body_len, close_conn ? "close" : "keep-alive");
    if (n < 0 || (size_t)n >= sizeof(header)) {
        return;
    }
    struct iovec iov[2] = {
        {(void*)header, (size_t)n},
        {(void*)body, body_len},
    };
    (void)rt_http_send_all_iov(fd, iov, body_len == 0 ? 1 : 2);
}

static RT_UNUSED int rt_http_ascii_lower(int c) {
    if (c >= 'A' && c <= 'Z') {
        return c + ('a' - 'A');
    }
    return c;
}

static RT_UNUSED bool rt_http_token_equal_ci(const uint8_t* left, size_t left_len, const uint8_t* right, size_t right_len) {
    if (left_len != right_len) {
        return false;
    }
    for (size_t i = 0; i < left_len; i++) {
        if (rt_http_ascii_lower(left[i]) != rt_http_ascii_lower(right[i])) {
            return false;
        }
    }
    return true;
}

static RT_UNUSED bool rt_http_string_equal_ci(rt_string left, const uint8_t* right, size_t right_len) {
    return rt_http_token_equal_ci(left.data, left.len, right, right_len);
}

static RT_UNUSED void rt_http_queue_init(rt_http_fd_queue* queue) {
    memset(queue, 0, sizeof(*queue));
    if (pthread_mutex_init(&queue->mutex, NULL) != 0) {
        rt_runtime_fail("pthread mutex init failed");
    }
    if (pthread_cond_init(&queue->not_empty, NULL) != 0) {
        rt_runtime_fail("pthread cond init failed");
    }
    if (pthread_cond_init(&queue->not_full, NULL) != 0) {
        rt_runtime_fail("pthread cond init failed");
    }
}

static RT_UNUSED void rt_http_queue_push(rt_http_fd_queue* queue, int fd) {
    if (pthread_mutex_lock(&queue->mutex) != 0) {
        rt_runtime_fail("pthread mutex lock failed");
    }
    while (queue->len == RT_HTTP_QUEUE_CAP) {
        if (pthread_cond_wait(&queue->not_full, &queue->mutex) != 0) {
            rt_runtime_fail("pthread cond wait failed");
        }
    }
    queue->fds[queue->tail] = fd;
    queue->tail = (queue->tail + 1) % RT_HTTP_QUEUE_CAP;
    queue->len++;
    if (pthread_cond_signal(&queue->not_empty) != 0) {
        rt_runtime_fail("pthread cond signal failed");
    }
    if (pthread_mutex_unlock(&queue->mutex) != 0) {
        rt_runtime_fail("pthread mutex unlock failed");
    }
}

static RT_UNUSED int rt_http_queue_pop(rt_http_fd_queue* queue) {
    if (pthread_mutex_lock(&queue->mutex) != 0) {
        rt_runtime_fail("pthread mutex lock failed");
    }
    while (queue->len == 0) {
        if (pthread_cond_wait(&queue->not_empty, &queue->mutex) != 0) {
            rt_runtime_fail("pthread cond wait failed");
        }
    }
    int fd = queue->fds[queue->head];
    queue->head = (queue->head + 1) % RT_HTTP_QUEUE_CAP;
    queue->len--;
    if (pthread_cond_signal(&queue->not_full) != 0) {
        rt_runtime_fail("pthread cond signal failed");
    }
    if (pthread_mutex_unlock(&queue->mutex) != 0) {
        rt_runtime_fail("pthread mutex unlock failed");
    }
    return fd;
}

static RT_UNUSED void rt_http_request_reset(rt_http_request* req, int fd) {
    req->fd = fd;
    req->len = 0;
    req->header_scan_start = 0;
    req->method = NULL;
    req->method_len = 0;
    req->path = NULL;
    req->path_len = 0;
    req->query = NULL;
    req->query_len = 0;
    req->body = NULL;
    req->body_len = 0;
    req->header_count = 0;
    req->responded = false;
    req->close_after_response = false;
}

static RT_UNUSED void rt_http_request_deinit(rt_http_request* req) {
    free(req->buffer);
    free(req->headers);
}

static RT_UNUSED void rt_http_request_reserve(rt_http_request* req, size_t want) {
    if (want <= req->cap) {
        return;
    }
    size_t cap = req->cap == 0 ? 4096 : req->cap;
    while (cap < want) {
        if (cap > SIZE_MAX / 2) {
            rt_runtime_fail("http request too large");
        }
        cap *= 2;
    }
    uint8_t* data = (uint8_t*)realloc(req->buffer, cap);
    if (data == NULL) {
        rt_runtime_fail("allocation failed");
    }
    req->buffer = data;
    req->cap = cap;
}

static RT_UNUSED ssize_t rt_http_find_header_end(rt_http_request* req) {
    if (req->len < 4) {
        return -1;
    }
    size_t start = req->header_scan_start;
    if (start > 3) {
        start -= 3;
    }
    for (size_t i = start; i + 3 < req->len; i++) {
        if (req->buffer[i] == '\r' && req->buffer[i + 1] == '\n' && req->buffer[i + 2] == '\r' && req->buffer[i + 3] == '\n') {
            return (ssize_t)(i + 4);
        }
    }
    req->header_scan_start = req->len - 3;
    return -1;
}

static RT_UNUSED bool rt_http_read_more(rt_http_request* req) {
    rt_http_request_reserve(req, req->len + 4096);
    for (;;) {
        ssize_t n = recv(req->fd, req->buffer + req->len, req->cap - req->len, 0);
        if (n < 0) {
            if (errno == EINTR) {
                continue;
            }
            return false;
        }
        if (n == 0) {
            return false;
        }
        req->len += (size_t)n;
        return true;
    }
}

static RT_UNUSED bool rt_http_read_headers(rt_http_request* req, size_t* header_end) {
    for (;;) {
        ssize_t end = rt_http_find_header_end(req);
        if (end >= 0) {
            *header_end = (size_t)end;
            return true;
        }
        if (req->len >= RT_HTTP_MAX_HEADER_BYTES) {
            return false;
        }
        if (!rt_http_read_more(req)) {
            return false;
        }
    }
}

static RT_UNUSED uint8_t* rt_http_memchr(uint8_t* data, size_t len, uint8_t needle) {
    return (uint8_t*)memchr(data, needle, len);
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
    if (len == 0) {
        return false;
    }
    for (size_t i = 0; i < len; i++) {
        if (data[i] < '0' || data[i] > '9') {
            return false;
        }
        size_t digit = (size_t)(data[i] - '0');
        if (value > (SIZE_MAX - digit) / 10) {
            return false;
        }
        value = value * 10 + digit;
    }
    *out = value;
    return true;
}

static RT_UNUSED bool rt_http_contains_token_ci(const uint8_t* data, size_t len, const uint8_t* token, size_t token_len) {
    const uint8_t* cursor = data;
    size_t remaining = len;
    for (;;) {
        const uint8_t* comma = memchr(cursor, ',', remaining);
        size_t part_len = comma == NULL ? remaining : (size_t)(comma - cursor);
        uint8_t* part = (uint8_t*)cursor;
        rt_http_trim_ows(&part, &part_len);
        if (rt_http_token_equal_ci(part, part_len, token, token_len)) {
            return true;
        }
        if (comma == NULL) {
            return false;
        }
        size_t consumed = (size_t)(comma + 1 - cursor);
        cursor = comma + 1;
        remaining -= consumed;
    }
}

static RT_UNUSED bool rt_http_parse_request(rt_http_request* req, size_t header_end, size_t* consumed) {
    size_t header_len = header_end - 2;
    uint8_t* start = req->buffer;
    uint8_t* end = req->buffer + header_len;
    uint8_t* line_end = rt_http_memchr(start, (size_t)(end - start), '\r');
    if (line_end == NULL || line_end + 1 >= end || line_end[1] != '\n') {
        return false;
    }

    uint8_t* sp1 = rt_http_memchr(start, (size_t)(line_end - start), ' ');
    if (sp1 == NULL) {
        return false;
    }
    uint8_t* sp2 = rt_http_memchr(sp1 + 1, (size_t)(line_end - (sp1 + 1)), ' ');
    if (sp2 == NULL) {
        return false;
    }
    req->method = start;
    req->method_len = (size_t)(sp1 - start);
    uint8_t* target = sp1 + 1;
    size_t target_len = (size_t)(sp2 - target);
    uint8_t* question = rt_http_memchr(target, target_len, '?');
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
    req->close_after_response = version[7] == '0';

    size_t header_cap = 16;
    if (req->headers == NULL) {
        req->headers = (rt_http_header_pair*)malloc(header_cap * sizeof(rt_http_header_pair));
        if (req->headers == NULL) {
            rt_runtime_fail("allocation failed");
        }
    }
    size_t content_length = 0;
    bool has_content_length = false;
    bool has_transfer_encoding = false;
    uint8_t* line = line_end + 2;
    while (line < end) {
        uint8_t* next = rt_http_memchr(line, (size_t)(end - line), '\r');
        if (next == NULL || next + 1 > end || next[1] != '\n') {
            return false;
        }
        uint8_t* colon = rt_http_memchr(line, (size_t)(next - line), ':');
        if (colon == NULL) {
            return false;
        }
        uint8_t* name = line;
        size_t name_len = (size_t)(colon - line);
        uint8_t* value = colon + 1;
        size_t value_len = (size_t)(next - value);
        rt_http_trim_ows(&value, &value_len);
        if (req->header_count == header_cap) {
            header_cap *= 2;
            rt_http_header_pair* headers = (rt_http_header_pair*)realloc(req->headers, header_cap * sizeof(rt_http_header_pair));
            if (headers == NULL) {
                rt_runtime_fail("allocation failed");
            }
            req->headers = headers;
        }
        req->headers[req->header_count++] = (rt_http_header_pair){name, name_len, value, value_len};
        if (rt_http_token_equal_ci(name, name_len, (const uint8_t*)"Content-Length", 14)) {
            if (has_content_length || has_transfer_encoding) {
                return false;
            }
            if (!rt_http_parse_content_length(value, value_len, &content_length)) {
                return false;
            }
            has_content_length = true;
        } else if (rt_http_token_equal_ci(name, name_len, (const uint8_t*)"Transfer-Encoding", 17)) {
            if (has_transfer_encoding || has_content_length) {
                return false;
            }
            has_transfer_encoding = true;
            return false;
        } else if (rt_http_token_equal_ci(name, name_len, (const uint8_t*)"Connection", 10)) {
            if (rt_http_contains_token_ci(value, value_len, (const uint8_t*)"close", 5)) {
                req->close_after_response = true;
            }
        }
        line = next + 2;
    }

    if (!has_content_length) {
        content_length = 0;
    }
    if (content_length > RT_HTTP_MAX_BODY_BYTES) {
        rt_http_send_simple(req->fd, 413, "payload too large\n", true);
        req->responded = true;
        req->close_after_response = true;
        return false;
    }
    while (req->len < header_end + content_length) {
        if (!rt_http_read_more(req)) {
            return false;
        }
    }
    req->body = req->buffer + header_end;
    req->body_len = content_length;
    *consumed = header_end + content_length;
    return true;
}

static RT_UNUSED rt_string rt_http_clone_string(rt_arena* arena, const uint8_t* data, size_t len) {
    if (len == 0) {
        return (rt_string){NULL, 0};
    }
    uint8_t* out = (uint8_t*)rt_arena_alloc(arena, len);
    memcpy(out, data, len);
    return (rt_string){out, len};
}

static RT_UNUSED rt_http_request* rt_http_request_from_handle(int64_t handle) {
    rt_http_request* req = (rt_http_request*)(intptr_t)handle;
    if (req == NULL) {
        rt_runtime_fail("invalid http request");
    }
    return req;
}

static RT_UNUSED rt_string rt_http_method(rt_arena* arena, int64_t handle) {
    rt_http_request* req = rt_http_request_from_handle(handle);
    return rt_http_clone_string(arena, req->method, req->method_len);
}

static RT_UNUSED rt_string rt_http_path(rt_arena* arena, int64_t handle) {
    rt_http_request* req = rt_http_request_from_handle(handle);
    return rt_http_clone_string(arena, req->path, req->path_len);
}

static RT_UNUSED rt_string rt_http_query(rt_arena* arena, int64_t handle) {
    rt_http_request* req = rt_http_request_from_handle(handle);
    return rt_http_clone_string(arena, req->query, req->query_len);
}

static RT_UNUSED rt_string rt_http_body(rt_arena* arena, int64_t handle) {
    rt_http_request* req = rt_http_request_from_handle(handle);
    return rt_http_clone_string(arena, req->body, req->body_len);
}

static RT_UNUSED rt_string rt_http_header(rt_arena* arena, int64_t handle, rt_string name) {
    rt_http_request* req = rt_http_request_from_handle(handle);
    for (size_t i = 0; i < req->header_count; i++) {
        rt_http_header_pair header = req->headers[i];
        if (rt_http_string_equal_ci(name, header.name, header.name_len)) {
            return rt_http_clone_string(arena, header.value, header.value_len);
        }
    }
    return (rt_string){NULL, 0};
}

static RT_UNUSED void rt_http_respond(int64_t handle, int64_t status, rt_string content_type, rt_string body) {
    rt_http_request* req = rt_http_request_from_handle(handle);
    if (req->responded) {
        return;
    }
    if (status < 100 || status > 999) {
        status = 500;
    }
    char header[512];
    int n = snprintf(header, sizeof(header),
        "HTTP/1.1 %lld %s\r\nContent-Type: %.*s\r\nContent-Length: %zu\r\nConnection: %s\r\n\r\n",
        (long long)status,
        rt_http_reason(status),
        (int)content_type.len,
        content_type.data == NULL ? (const uint8_t*)"" : content_type.data,
        body.len,
        req->close_after_response ? "close" : "keep-alive");
    if (n < 0 || (size_t)n >= sizeof(header)) {
        req->close_after_response = true;
        req->responded = true;
        return;
    }
    struct iovec iov[2] = {
        {(void*)header, (size_t)n},
        {(void*)body.data, body.len},
    };
    int iovcnt = body.len > 0 && body.data != NULL ? 2 : 1;
    if (!rt_http_send_all_iov(req->fd, iov, iovcnt)) {
        req->close_after_response = true;
    }
    req->responded = true;
}

static RT_UNUSED void rt_http_compact_buffer(rt_http_request* req, size_t consumed) {
    if (consumed >= req->len) {
        req->len = 0;
        req->header_scan_start = 0;
        return;
    }
    memmove(req->buffer, req->buffer + consumed, req->len - consumed);
    req->len -= consumed;
    req->header_scan_start = 0;
}

static RT_UNUSED void rt_http_handle_connection(int fd, rt_http_handler_fn handler) {
    rt_http_request req;
    memset(&req, 0, sizeof(req));
    rt_http_request_reset(&req, fd);
    rt_arena arena;
    rt_arena_init(&arena);

    for (;;) {
        rt_arena_reset(&arena);
        size_t header_end = 0;
        if (!rt_http_read_headers(&req, &header_end)) {
            if (req.len > 0) {
                rt_http_send_simple(fd, 400, "bad request\n", true);
            }
            break;
        }

        size_t consumed = 0;
        bool parsed = rt_http_parse_request(&req, header_end, &consumed);
        if (!parsed) {
            if (!req.responded) {
                rt_http_send_simple(fd, 400, "bad request\n", true);
            }
            break;
        }

        rt_context ctx = {&arena};
        handler(&ctx, &arena, (int64_t)(intptr_t)&req);
        if (!req.responded) {
            rt_http_send_simple(fd, 500, "handler did not respond\n", req.close_after_response);
        }

        bool close_conn = req.close_after_response;
        rt_http_compact_buffer(&req, consumed);
        if (close_conn) {
            break;
        }
        req.method = NULL;
        req.method_len = 0;
        req.path = NULL;
        req.path_len = 0;
        req.query = NULL;
        req.query_len = 0;
        req.body = NULL;
        req.body_len = 0;
        req.header_count = 0;
        req.responded = false;
        req.close_after_response = false;
    }

    rt_arena_deinit(&arena);
    rt_http_request_deinit(&req);
    rt_http_close_fd(fd);
}

static RT_UNUSED void* rt_http_worker_main(void* arg) {
    rt_http_worker_args* args = (rt_http_worker_args*)arg;
    for (;;) {
        int fd = rt_http_queue_pop(args->queue);
        rt_http_handle_connection(fd, args->handler);
    }
    return NULL;
}

static RT_UNUSED void rt_http_serve(rt_string host, int64_t port_value, int64_t workers_value, rt_http_handler_fn handler) {
    if (port_value < 1 || port_value > 65535) {
        rt_runtime_fail("http port must be between 1 and 65535");
    }
    if (workers_value < 1 || workers_value > 1024) {
        rt_runtime_fail("http workers must be between 1 and 1024");
    }
#ifdef SIGPIPE
    signal(SIGPIPE, SIG_IGN);
#endif
    char* host_text = rt_string_to_c_string(host, "http host");
    int listen_fd = socket(AF_INET, SOCK_STREAM, 0);
    if (listen_fd < 0) {
        rt_runtime_fail("socket failed");
    }
    int one = 1;
    (void)setsockopt(listen_fd, SOL_SOCKET, SO_REUSEADDR, &one, sizeof(one));

    struct sockaddr_in addr;
    memset(&addr, 0, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_port = htons((uint16_t)port_value);
    if (inet_pton(AF_INET, host_text, &addr.sin_addr) != 1) {
        rt_runtime_fail("http host must be an IPv4 address");
    }
    free(host_text);

    if (bind(listen_fd, (struct sockaddr*)&addr, sizeof(addr)) != 0) {
        rt_runtime_fail("bind failed");
    }
    if (listen(listen_fd, 1024) != 0) {
        rt_runtime_fail("listen failed");
    }

    rt_http_fd_queue queue;
    rt_http_queue_init(&queue);
    rt_http_worker_args args = {&queue, handler};
    pthread_t* workers = (pthread_t*)malloc((size_t)workers_value * sizeof(pthread_t));
    if (workers == NULL) {
        rt_runtime_fail("allocation failed");
    }
    for (int64_t i = 0; i < workers_value; i++) {
        if (pthread_create(&workers[i], NULL, rt_http_worker_main, &args) != 0) {
            rt_runtime_fail("pthread create failed");
        }
        (void)pthread_detach(workers[i]);
    }

    for (;;) {
        int fd = accept(listen_fd, NULL, NULL);
        if (fd < 0) {
            if (errno == EINTR) {
                continue;
            }
            rt_runtime_fail("accept failed");
        }
        rt_http_configure_client_socket(fd);
        rt_http_queue_push(&queue, fd);
    }
}

#else

typedef int64_t (*rt_http_handler_fn)(rt_context* trux_ctx, rt_arena* trux_result_arena, int64_t request);

static RT_UNUSED rt_string rt_http_method(rt_arena* arena, int64_t handle) { (void)arena; (void)handle; rt_runtime_fail("http is not supported on Windows"); return (rt_string){NULL, 0}; }
static RT_UNUSED rt_string rt_http_path(rt_arena* arena, int64_t handle) { (void)arena; (void)handle; rt_runtime_fail("http is not supported on Windows"); return (rt_string){NULL, 0}; }
static RT_UNUSED rt_string rt_http_query(rt_arena* arena, int64_t handle) { (void)arena; (void)handle; rt_runtime_fail("http is not supported on Windows"); return (rt_string){NULL, 0}; }
static RT_UNUSED rt_string rt_http_header(rt_arena* arena, int64_t handle, rt_string name) { (void)arena; (void)handle; (void)name; rt_runtime_fail("http is not supported on Windows"); return (rt_string){NULL, 0}; }
static RT_UNUSED rt_string rt_http_body(rt_arena* arena, int64_t handle) { (void)arena; (void)handle; rt_runtime_fail("http is not supported on Windows"); return (rt_string){NULL, 0}; }
static RT_UNUSED void rt_http_respond(int64_t handle, int64_t status, rt_string content_type, rt_string body) { (void)handle; (void)status; (void)content_type; (void)body; rt_runtime_fail("http is not supported on Windows"); }
static RT_UNUSED void rt_http_serve(rt_string host, int64_t port, int64_t workers, rt_http_handler_fn handler) { (void)host; (void)port; (void)workers; (void)handler; rt_runtime_fail("http is not supported on Windows"); }

#endif

#endif
`

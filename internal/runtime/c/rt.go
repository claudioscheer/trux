package c

const HeaderName = "trux_runtime.h"

const Source = `#ifndef TRUX_RUNTIME_H
#define TRUX_RUNTIME_H

#define _POSIX_C_SOURCE 200809L
#include <ctype.h>
#include <errno.h>
#include <stdbool.h>
#include <stdint.h>
#include <stddef.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <inttypes.h>
#include <time.h>

#ifdef TRUX_RUNTIME_CUDA
#include <cuda_runtime.h>
#endif

#if defined(__GNUC__) || defined(__clang__)
#define RT_UNUSED __attribute__((unused))
#else
#define RT_UNUSED
#endif

#if defined(__cplusplus)
#define RT_ALIGNOF(T) alignof(T)
#else
#define RT_ALIGNOF(T) _Alignof(T)
#endif

#define RT_ARENA_DEFAULT_CHUNK_SIZE ((size_t)4096)

typedef struct {
    const uint8_t* data;
    size_t len;
} rt_string;

typedef struct rt_arena_chunk {
    struct rt_arena_chunk* next;
    size_t cap;
    size_t used;
    max_align_t data[];
} rt_arena_chunk;

typedef struct rt_list_allocation {
    void* list;
    void (*free_list)(void*);
    struct rt_list_allocation* next;
} rt_list_allocation;

typedef struct {
    rt_arena_chunk* chunks;
    rt_list_allocation* lists;
} rt_arena;

typedef struct {
    rt_arena_chunk* chunk;
    size_t used;
    rt_list_allocation* lists;
} rt_arena_mark;

typedef struct {
    rt_arena* arena;
} rt_context;

typedef struct {
    size_t start;
    size_t end;
} rt_range;

typedef struct {
    uint8_t* data;
    size_t len;
    size_t cap;
} rt_byte_buffer;

static RT_UNUSED void rt_runtime_fail(const char* message) {
    fprintf(stderr, "trux runtime error: %s\n", message);
    exit(1);
}

static RT_UNUSED void rt_arithmetic_fail(const char* operation, const char* file, int line) {
    fprintf(stderr, "trux runtime error: integer %s\n  at %s:%d\n", operation, file, line);
    exit(1);
}

static RT_UNUSED int64_t rt_add_i64_at(int64_t left, int64_t right, const char* file, int line) {
    if ((right > 0 && left > INT64_MAX - right) || (right < 0 && left < INT64_MIN - right)) {
        rt_arithmetic_fail("overflow in +", file, line);
    }
    return left + right;
}

static RT_UNUSED int64_t rt_sub_i64_at(int64_t left, int64_t right, const char* file, int line) {
    if ((right < 0 && left > INT64_MAX + right) || (right > 0 && left < INT64_MIN + right)) {
        rt_arithmetic_fail("overflow in -", file, line);
    }
    return left - right;
}

static RT_UNUSED int64_t rt_mul_i64_at(int64_t left, int64_t right, const char* file, int line) {
    if (left != 0 && right != 0) {
        if (left == INT64_MIN && right == -1) {
            rt_arithmetic_fail("overflow in *", file, line);
        }
        if (right == INT64_MIN && left == -1) {
            rt_arithmetic_fail("overflow in *", file, line);
        }
        if (left > 0) {
            if ((right > 0 && left > INT64_MAX / right) || (right < 0 && right < INT64_MIN / left)) {
                rt_arithmetic_fail("overflow in *", file, line);
            }
        } else {
            if ((right > 0 && left < INT64_MIN / right) || (right < 0 && left < INT64_MAX / right)) {
                rt_arithmetic_fail("overflow in *", file, line);
            }
        }
    }
    return left * right;
}

static RT_UNUSED int64_t rt_div_i64_at(int64_t left, int64_t right, const char* file, int line) {
    if (right == 0) {
        rt_arithmetic_fail("division by zero", file, line);
    }
    if (left == INT64_MIN && right == -1) {
        rt_arithmetic_fail("overflow in /", file, line);
    }
    return left / right;
}

#ifdef TRUX_RUNTIME_CUDA
__device__ static int64_t rt_add_i64_device(int64_t left, int64_t right) {
    if ((right > 0 && left > INT64_MAX - right) || (right < 0 && left < INT64_MIN - right)) {
        asm("trap;");
    }
    return left + right;
}

__device__ static int64_t rt_sub_i64_device(int64_t left, int64_t right) {
    if ((right < 0 && left > INT64_MAX + right) || (right > 0 && left < INT64_MIN + right)) {
        asm("trap;");
    }
    return left - right;
}

__device__ static int64_t rt_mul_i64_device(int64_t left, int64_t right) {
    if (left != 0 && right != 0) {
        if ((left == INT64_MIN && right == -1) || (right == INT64_MIN && left == -1)) {
            asm("trap;");
        }
        if (left > 0) {
            if ((right > 0 && left > INT64_MAX / right) || (right < 0 && right < INT64_MIN / left)) {
                asm("trap;");
            }
        } else {
            if ((right > 0 && left < INT64_MIN / right) || (right < 0 && left < INT64_MAX / right)) {
                asm("trap;");
            }
        }
    }
    return left * right;
}

__device__ static int64_t rt_div_i64_device(int64_t left, int64_t right) {
    if (right == 0 || (left == INT64_MIN && right == -1)) {
        asm("trap;");
    }
    return left / right;
}
#endif

#if defined(TRUX_RUNTIME_CUDA) && defined(__CUDA_ARCH__)
#define rt_add_i64(LEFT, RIGHT) rt_add_i64_device((LEFT), (RIGHT))
#define rt_sub_i64(LEFT, RIGHT) rt_sub_i64_device((LEFT), (RIGHT))
#define rt_mul_i64(LEFT, RIGHT) rt_mul_i64_device((LEFT), (RIGHT))
#define rt_div_i64(LEFT, RIGHT) rt_div_i64_device((LEFT), (RIGHT))
#else
#define rt_add_i64(LEFT, RIGHT) rt_add_i64_at((LEFT), (RIGHT), __FILE__, __LINE__)
#define rt_sub_i64(LEFT, RIGHT) rt_sub_i64_at((LEFT), (RIGHT), __FILE__, __LINE__)
#define rt_mul_i64(LEFT, RIGHT) rt_mul_i64_at((LEFT), (RIGHT), __FILE__, __LINE__)
#define rt_div_i64(LEFT, RIGHT) rt_div_i64_at((LEFT), (RIGHT), __FILE__, __LINE__)
#endif

static RT_UNUSED size_t rt_arena_align_up(size_t size) {
    size_t align = RT_ALIGNOF(max_align_t);
    size_t remainder = size % align;
    if (remainder == 0) {
        return size;
    }
    size_t padding = align - remainder;
    if (size > SIZE_MAX - padding) {
        rt_runtime_fail("allocation size overflow");
    }
    return size + padding;
}

static RT_UNUSED void rt_arena_free_lists_until(rt_arena* arena, rt_list_allocation* stop) {
    rt_list_allocation* list = arena->lists;
    while (list != stop) {
        rt_list_allocation* next = list->next;
        list->free_list(list->list);
        free(list);
        list = next;
    }
    arena->lists = stop;
}

static RT_UNUSED void rt_arena_free_lists(rt_arena* arena) {
    rt_arena_free_lists_until(arena, NULL);
}

static RT_UNUSED rt_arena_chunk* rt_arena_new_chunk(size_t min_cap) {
    size_t cap = RT_ARENA_DEFAULT_CHUNK_SIZE;
    if (min_cap > cap) {
        cap = min_cap;
    }
    if (cap > SIZE_MAX - sizeof(rt_arena_chunk)) {
        rt_runtime_fail("allocation size overflow");
    }

    rt_arena_chunk* chunk = (rt_arena_chunk*)malloc(sizeof(rt_arena_chunk) + cap);
    if (chunk == NULL) {
        rt_runtime_fail("allocation failed");
    }
    chunk->next = NULL;
    chunk->cap = cap;
    chunk->used = 0;
    return chunk;
}

static RT_UNUSED void rt_arena_init(rt_arena* arena) {
    arena->chunks = NULL;
    arena->lists = NULL;
}

static RT_UNUSED void rt_arena_reset(rt_arena* arena) {
    rt_arena_free_lists(arena);

    rt_arena_chunk* chunk = arena->chunks;
    while (chunk != NULL) {
        chunk->used = 0;
        chunk = chunk->next;
    }
}

static RT_UNUSED void rt_arena_deinit(rt_arena* arena) {
    rt_arena_free_lists(arena);

    rt_arena_chunk* chunk = arena->chunks;
    while (chunk != NULL) {
        rt_arena_chunk* next = chunk->next;
        free(chunk);
        chunk = next;
    }
    arena->chunks = NULL;
}

static RT_UNUSED rt_arena_mark rt_arena_mark_current(rt_arena* arena) {
    return (rt_arena_mark){
        arena->chunks,
        arena->chunks == NULL ? 0 : arena->chunks->used,
        arena->lists,
    };
}

static RT_UNUSED void rt_arena_rewind(rt_arena* arena, rt_arena_mark mark) {
    rt_arena_free_lists_until(arena, mark.lists);

    rt_arena_chunk* chunk = arena->chunks;
    while (chunk != NULL && chunk != mark.chunk) {
        chunk->used = 0;
        chunk = chunk->next;
    }
    if (mark.chunk != NULL) {
        mark.chunk->used = mark.used;
    }
}

static RT_UNUSED void rt_arena_register_list(rt_arena* arena, void* list, void (*free_list)(void*)) {
    rt_list_allocation* node = (rt_list_allocation*)malloc(sizeof(rt_list_allocation));
    if (node == NULL) {
        rt_runtime_fail("allocation failed");
    }
    node->list = list;
    node->free_list = free_list;
    node->next = arena->lists;
    arena->lists = node;
}

static RT_UNUSED void* rt_arena_alloc(rt_arena* arena, size_t size) {
    if (size == 0) {
        return NULL;
    }
    size_t aligned_size = rt_arena_align_up(size);

    rt_arena_chunk* chunk = arena->chunks;
    if (chunk == NULL || aligned_size > chunk->cap - chunk->used) {
        chunk = rt_arena_new_chunk(aligned_size);
        chunk->next = arena->chunks;
        arena->chunks = chunk;
    }

    void* data = (uint8_t*)chunk->data + chunk->used;
    chunk->used += aligned_size;
    return data;
}

static RT_UNUSED size_t rt_checked_count(int64_t count, const char* what) {
    if (count < 0) {
        fprintf(stderr, "trux runtime error: %s must be non-negative, got %" PRId64 "\n", what, count);
        exit(1);
    }
    return (size_t)count;
}

static RT_UNUSED size_t rt_checked_positive_count(int64_t count, const char* what) {
    if (count <= 0) {
        fprintf(stderr, "trux runtime error: %s must be positive, got %" PRId64 "\n", what, count);
        exit(1);
    }
    return (size_t)count;
}

static RT_UNUSED int64_t rt_checked_len_i64(size_t len) {
    if (len > (size_t)INT64_MAX) {
        rt_runtime_fail("length too large");
    }
    return (int64_t)len;
}

static RT_UNUSED size_t rt_checked_bytes(size_t count, size_t elem_size) {
    if (elem_size != 0 && count > SIZE_MAX / elem_size) {
        rt_runtime_fail("allocation size overflow");
    }
    return count * elem_size;
}

static RT_UNUSED void* rt_arena_alloc_count(rt_arena* arena, size_t count, size_t elem_size, bool zeroed) {
    size_t bytes = rt_checked_bytes(count, elem_size);
    void* data = rt_arena_alloc(arena, bytes);
    if (zeroed && data != NULL) {
        memset(data, 0, bytes);
    }
    return data;
}

static RT_UNUSED size_t rt_check_index(size_t len, int64_t index) {
    if (index < 0 || (uint64_t)index >= len) {
        fprintf(stderr, "trux runtime error: index %" PRId64 " out of bounds for length %zu\n", index, len);
        exit(1);
    }
    return (size_t)index;
}

static RT_UNUSED rt_range rt_check_slice(size_t len, bool has_start, int64_t start_value, bool has_end, int64_t end_value) {
    int64_t start = has_start ? start_value : 0;
    int64_t end = has_end ? end_value : rt_checked_len_i64(len);
    if (start < 0 || end < 0 || start > end || (uint64_t)end > len) {
        fprintf(stderr, "trux runtime error: slice %" PRId64 ":%" PRId64 " out of bounds for length %zu\n", start, end, len);
        exit(1);
    }
    return (rt_range){(size_t)start, (size_t)end};
}

static RT_UNUSED void rt_check_string(rt_string value) {
    if (value.len > 0 && value.data == NULL) {
        rt_runtime_fail("invalid string: non-zero length with NULL data");
    }
}

static RT_UNUSED void rt_assert_fail(rt_string message) {
    rt_check_string(message);
    fprintf(stderr, "trux runtime error: assertion failed: ");
    if (message.len > 0 && fwrite(message.data, 1, message.len, stderr) != message.len) {
        rt_runtime_fail("write failed");
    }
    fputc('\n', stderr);
    exit(1);
}

static RT_UNUSED void rt_assert_fail_at(rt_string message, const char* file, int line, int column) {
    rt_check_string(message);
    fprintf(stderr, "trux runtime error: assertion failed: ");
    if (message.len > 0 && fwrite(message.data, 1, message.len, stderr) != message.len) {
        rt_runtime_fail("write failed");
    }
    fprintf(stderr, "\n  at %s:%d:%d\n", file, line, column);
    exit(1);
}

static RT_UNUSED void rt_check_array_like(const void* data, size_t len, const char* what) {
    if (len > 0 && data == NULL) {
        fprintf(stderr, "trux runtime error: invalid %s: non-zero length with NULL data\n", what);
        exit(1);
    }
}

static RT_UNUSED void rt_byte_buffer_init(rt_byte_buffer* buffer) {
    buffer->data = NULL;
    buffer->len = 0;
    buffer->cap = 0;
}

static RT_UNUSED void rt_byte_buffer_free(rt_byte_buffer* buffer) {
    free(buffer->data);
    buffer->data = NULL;
    buffer->len = 0;
    buffer->cap = 0;
}

static RT_UNUSED void rt_byte_buffer_reserve(rt_byte_buffer* buffer, size_t needed) {
    if (needed <= buffer->cap) {
        return;
    }

    size_t cap = buffer->cap == 0 ? (size_t)128 : buffer->cap;
    while (cap < needed) {
        if (cap > SIZE_MAX / 2) {
            rt_runtime_fail("buffer size overflow");
        }
        cap *= 2;
    }

    uint8_t* data = (uint8_t*)realloc(buffer->data, cap);
    if (data == NULL) {
        rt_runtime_fail("allocation failed");
    }
    buffer->data = data;
    buffer->cap = cap;
}

static RT_UNUSED void rt_byte_buffer_append_byte(rt_byte_buffer* buffer, uint8_t value) {
    if (buffer->len == SIZE_MAX) {
        rt_runtime_fail("buffer size overflow");
    }
    rt_byte_buffer_reserve(buffer, buffer->len + 1);
    buffer->data[buffer->len] = value;
    buffer->len++;
}

static RT_UNUSED void rt_byte_buffer_append_bytes(rt_byte_buffer* buffer, const uint8_t* data, size_t len) {
    if (len == 0) {
        return;
    }
    if (data == NULL) {
        rt_runtime_fail("invalid bytes: non-zero length with NULL data");
    }
    if (buffer->len > SIZE_MAX - len) {
        rt_runtime_fail("buffer size overflow");
    }
    rt_byte_buffer_reserve(buffer, buffer->len + len);
    memcpy(buffer->data + buffer->len, data, len);
    buffer->len += len;
}

static RT_UNUSED rt_string rt_byte_buffer_to_string(rt_arena* arena, const rt_byte_buffer* buffer) {
    if (buffer->len == 0) {
        return (rt_string){NULL, 0};
    }
    uint8_t* data = (uint8_t*)rt_arena_alloc(arena, buffer->len);
    memcpy(data, buffer->data, buffer->len);
    return (rt_string){data, buffer->len};
}

static RT_UNUSED rt_string rt_string_trim_ascii(rt_string value) {
    rt_check_string(value);
    size_t start = 0;
    size_t end = value.len;
    while (start < end && isspace((unsigned char)value.data[start])) {
        start++;
    }
    while (end > start && isspace((unsigned char)value.data[end - 1])) {
        end--;
    }
    return (rt_string){value.data == NULL ? NULL : value.data + start, end - start};
}

static RT_UNUSED bool rt_string_equal_c(rt_string value, const char* text) {
    size_t len = strlen(text);
    rt_check_string(value);
    return value.len == len && (len == 0 || memcmp(value.data, text, len) == 0);
}

static RT_UNUSED char* rt_string_to_c_string(rt_string value, const char* what) {
    rt_check_string(value);
    if (value.len == SIZE_MAX) {
        rt_runtime_fail("string length overflow");
    }
    for (size_t i = 0; i < value.len; i++) {
        if (value.data[i] == 0) {
            fprintf(stderr, "trux runtime error: %s contains NUL byte\n", what);
            exit(1);
        }
    }
    char* out = (char*)malloc(value.len + 1);
    if (out == NULL) {
        rt_runtime_fail("allocation failed");
    }
    if (value.len > 0) {
        memcpy(out, value.data, value.len);
    }
    out[value.len] = 0;
    return out;
}

static RT_UNUSED void rt_file_fail(const char* operation, const char* path) {
    fprintf(stderr, "trux runtime error: cannot %s %s: %s\n", operation, path, strerror(errno));
    exit(1);
}

static RT_UNUSED void rt_file_write_byte(FILE* file, uint8_t value) {
    if (fputc(value, file) == EOF) {
        rt_runtime_fail("write failed");
    }
}

static RT_UNUSED void rt_file_write_string(FILE* file, rt_string value) {
    if (value.len == 0) {
        return;
    }
    rt_check_string(value);
    if (fwrite(value.data, 1, value.len, file) != value.len) {
        rt_runtime_fail("write failed");
    }
}

static RT_UNUSED rt_string rt_read_line(rt_arena* arena) {
    rt_byte_buffer buffer;
    rt_byte_buffer_init(&buffer);

    for (;;) {
        int ch = fgetc(stdin);
        if (ch == EOF) {
            if (ferror(stdin)) {
                rt_runtime_fail("read failed");
            }
            break;
        }
        if (ch == '\n') {
            break;
        }
        rt_byte_buffer_append_byte(&buffer, (uint8_t)ch);
    }

    if (buffer.len > 0 && buffer.data[buffer.len - 1] == '\r') {
        buffer.len--;
    }

    rt_string line = rt_byte_buffer_to_string(arena, &buffer);
    rt_byte_buffer_free(&buffer);
    return line;
}

static RT_UNUSED int64_t rt_read_int(rt_arena* arena) {
    rt_string line = rt_string_trim_ascii(rt_read_line(arena));
    char* text = rt_string_to_c_string(line, "integer input");
    errno = 0;
    char* end = NULL;
    long long value = strtoll(text, &end, 10);
    if (errno == ERANGE || end == text || *end != 0) {
        free(text);
        rt_runtime_fail("invalid int input");
    }
    free(text);
    return (int64_t)value;
}

static RT_UNUSED double rt_read_float(rt_arena* arena) {
    rt_string line = rt_string_trim_ascii(rt_read_line(arena));
    char* text = rt_string_to_c_string(line, "float input");
    errno = 0;
    char* end = NULL;
    double value = strtod(text, &end);
    if (errno == ERANGE || end == text || *end != 0) {
        free(text);
        rt_runtime_fail("invalid float input");
    }
    free(text);
    return value;
}

static RT_UNUSED bool rt_read_bool(rt_arena* arena) {
    rt_string line = rt_string_trim_ascii(rt_read_line(arena));
    if (rt_string_equal_c(line, "true")) {
        return true;
    }
    if (rt_string_equal_c(line, "false")) {
        return false;
    }
    rt_runtime_fail("invalid bool input");
    return false;
}

static RT_UNUSED int64_t rt_time_from_timespec_millis(struct timespec ts) {
    return (int64_t)ts.tv_sec * 1000 + (int64_t)(ts.tv_nsec / 1000000);
}

static RT_UNUSED int64_t rt_time_from_timespec_nanos(struct timespec ts) {
    return (int64_t)ts.tv_sec * 1000000000 + (int64_t)ts.tv_nsec;
}

static RT_UNUSED int64_t rt_time_now_unix_millis(void) {
    struct timespec ts;
    if (clock_gettime(CLOCK_REALTIME, &ts) != 0) {
        rt_runtime_fail("clock_gettime failed");
    }
    return rt_time_from_timespec_millis(ts);
}

static RT_UNUSED int64_t rt_time_monotonic_nanos(void) {
    struct timespec ts;
    if (clock_gettime(CLOCK_MONOTONIC, &ts) != 0) {
        rt_runtime_fail("clock_gettime failed");
    }
    return rt_time_from_timespec_nanos(ts);
}

static RT_UNUSED void rt_time_sleep_millis(int64_t millis) {
    size_t checked = rt_checked_count(millis, "sleep milliseconds");
    struct timespec remaining;
    remaining.tv_sec = (time_t)(checked / 1000);
    remaining.tv_nsec = (long)((checked % 1000) * 1000000);
    while (nanosleep(&remaining, &remaining) != 0) {
        if (errno != EINTR) {
            rt_runtime_fail("sleep failed");
        }
    }
}

static RT_UNUSED rt_string rt_read_file(rt_arena* arena, rt_string path) {
    char* c_path = rt_string_to_c_string(path, "file path");
    FILE* file = fopen(c_path, "rb");
    if (file == NULL) {
        rt_file_fail("open file for reading", c_path);
    }

    rt_byte_buffer buffer;
    rt_byte_buffer_init(&buffer);
    uint8_t chunk[4096];
    for (;;) {
        size_t read = fread(chunk, 1, sizeof(chunk), file);
        rt_byte_buffer_append_bytes(&buffer, chunk, read);
        if (read < sizeof(chunk)) {
            if (ferror(file)) {
                rt_runtime_fail("read failed");
            }
            break;
        }
    }
    if (fclose(file) != 0) {
        rt_file_fail("close file after reading", c_path);
    }

    rt_string contents = rt_byte_buffer_to_string(arena, &buffer);
    rt_byte_buffer_free(&buffer);
    free(c_path);
    return contents;
}

static RT_UNUSED void rt_write_file(rt_string path, rt_string contents) {
    char* c_path = rt_string_to_c_string(path, "file path");
    FILE* file = fopen(c_path, "wb");
    if (file == NULL) {
        rt_file_fail("open file for writing", c_path);
    }

    rt_file_write_string(file, contents);
    if (fclose(file) != 0) {
        rt_file_fail("close file after writing", c_path);
    }
    free(c_path);
}

static RT_UNUSED void rt_print_int(int64_t value) {
    printf("%" PRId64, value);
}

static RT_UNUSED void rt_print_float(double value) {
    printf("%.15g", value);
}

static RT_UNUSED void rt_print_string(rt_string value) {
    if (value.len == 0) {
        return;
    }
    rt_check_string(value);
    if (fwrite(value.data, 1, value.len, stdout) != value.len) {
        rt_runtime_fail("write failed");
    }
}

static RT_UNUSED void rt_print_bool(bool value) {
    printf("%s", value ? "true" : "false");
}

static RT_UNUSED void rt_print_newline(void) {
    putchar('\n');
}

static RT_UNUSED bool rt_string_equal(rt_string left, rt_string right) {
    rt_check_string(left);
    rt_check_string(right);
    if (left.len != right.len) {
        return false;
    }
    if (left.len == 0) {
        return true;
    }
    return memcmp(left.data, right.data, left.len) == 0;
}

static RT_UNUSED rt_string rt_string_concat(rt_arena* arena, rt_string left, rt_string right) {
    if (left.len > SIZE_MAX - right.len) {
        rt_runtime_fail("string length overflow");
    }

    size_t len = left.len + right.len;
    if (len == 0) {
        return (rt_string){NULL, 0};
    }

    uint8_t* data = (uint8_t*)rt_arena_alloc(arena, len);
    if (left.len > 0) {
        rt_check_string(left);
        memcpy(data, left.data, left.len);
    }
    if (right.len > 0) {
        rt_check_string(right);
        memcpy(data + left.len, right.data, right.len);
    }
    return (rt_string){data, len};
}

static RT_UNUSED rt_string rt_string_clone(rt_arena* arena, rt_string value) {
    if (value.len == 0) {
        return (rt_string){NULL, 0};
    }
    rt_check_string(value);
    uint8_t* data = (uint8_t*)rt_arena_alloc(arena, value.len);
    memcpy(data, value.data, value.len);
    return (rt_string){data, value.len};
}

static RT_UNUSED rt_string rt_string_index(rt_string value, int64_t index) {
    rt_check_string(value);
    size_t checked = rt_check_index(value.len, index);
    return (rt_string){value.data + checked, 1};
}

static RT_UNUSED rt_string rt_string_slice(rt_string value, bool has_start, int64_t start, bool has_end, int64_t end) {
    rt_check_string(value);
    rt_range range = rt_check_slice(value.len, has_start, start, has_end, end);
    const uint8_t* data = value.data == NULL ? NULL : value.data + range.start;
    return (rt_string){data, range.end - range.start};
}

static RT_UNUSED bool rt_string_contains(rt_string needle, rt_string haystack) {
    rt_check_string(needle);
    rt_check_string(haystack);
    if (needle.len == 0) {
        return true;
    }
    if (needle.len > haystack.len) {
        return false;
    }

    const uint8_t* cursor = haystack.data;
    size_t remaining = haystack.len;
    while (remaining >= needle.len) {
        const void* found = memchr(cursor, needle.data[0], remaining - needle.len + 1);
        if (found == NULL) {
            return false;
        }
        cursor = (const uint8_t*)found;
        if (memcmp(cursor, needle.data, needle.len) == 0) {
            return true;
        }
        cursor++;
        remaining = haystack.len - (size_t)(cursor - haystack.data);
    }

    return false;
}

#define RT_CLONE_VALUE_int(ARENA, VALUE) (VALUE)
#define RT_CLONE_VALUE_float(ARENA, VALUE) (VALUE)
#define RT_CLONE_VALUE_bool(ARENA, VALUE) (VALUE)
#define RT_CLONE_VALUE_string(ARENA, VALUE) rt_string_clone((ARENA), (VALUE))

#define RT_DEFINE_COLLECTIONS(NAME, CTYPE) \
typedef struct { \
    CTYPE* data; \
    size_t len; \
} rt_array_##NAME; \
typedef struct { \
    CTYPE* data; \
    size_t len; \
} rt_slice_##NAME; \
typedef struct { \
    CTYPE* data; \
    size_t len; \
    size_t cap; \
} rt_list_##NAME; \
static RT_UNUSED rt_array_##NAME rt_array_##NAME##_from_values(rt_arena* arena, CTYPE const* values, size_t len) { \
    rt_check_array_like(values, len, "array values"); \
    CTYPE* data = (CTYPE*)rt_arena_alloc_count(arena, len, sizeof(CTYPE), false); \
    if (len > 0) { \
        memcpy(data, values, len * sizeof(CTYPE)); \
    } \
    return (rt_array_##NAME){data, len}; \
} \
static RT_UNUSED rt_array_##NAME rt_array_##NAME##_clone(rt_arena* arena, rt_array_##NAME value) { \
    rt_check_array_like(value.data, value.len, "array"); \
    CTYPE* data = (CTYPE*)rt_arena_alloc_count(arena, value.len, sizeof(CTYPE), false); \
    for (size_t i = 0; i < value.len; i++) { \
        data[i] = RT_CLONE_VALUE_##NAME(arena, value.data[i]); \
    } \
    return (rt_array_##NAME){data, value.len}; \
} \
static RT_UNUSED rt_slice_##NAME rt_make_slice_##NAME(rt_arena* arena, int64_t count) { \
    size_t len = rt_checked_count(count, "slice length"); \
    CTYPE* data = (CTYPE*)rt_arena_alloc_count(arena, len, sizeof(CTYPE), true); \
    return (rt_slice_##NAME){data, len}; \
} \
static RT_UNUSED rt_slice_##NAME rt_slice_##NAME##_clone(rt_arena* arena, rt_slice_##NAME value) { \
    rt_check_array_like(value.data, value.len, "slice"); \
    CTYPE* data = (CTYPE*)rt_arena_alloc_count(arena, value.len, sizeof(CTYPE), false); \
    for (size_t i = 0; i < value.len; i++) { \
        data[i] = RT_CLONE_VALUE_##NAME(arena, value.data[i]); \
    } \
    return (rt_slice_##NAME){data, value.len}; \
} \
static RT_UNUSED CTYPE rt_array_##NAME##_get(rt_array_##NAME value, int64_t index) { \
    rt_check_array_like(value.data, value.len, "array"); \
    return value.data[rt_check_index(value.len, index)]; \
} \
static RT_UNUSED CTYPE rt_slice_##NAME##_get(rt_slice_##NAME value, int64_t index) { \
    rt_check_array_like(value.data, value.len, "slice"); \
    return value.data[rt_check_index(value.len, index)]; \
} \
static RT_UNUSED CTYPE rt_list_##NAME##_get(rt_list_##NAME* value, int64_t index) { \
    rt_check_array_like(value->data, value->len, "list"); \
    return value->data[rt_check_index(value->len, index)]; \
} \
static RT_UNUSED void rt_array_##NAME##_set(rt_array_##NAME value, int64_t index, CTYPE elem) { \
    rt_check_array_like(value.data, value.len, "array"); \
    value.data[rt_check_index(value.len, index)] = elem; \
} \
static RT_UNUSED void rt_slice_##NAME##_set(rt_slice_##NAME value, int64_t index, CTYPE elem) { \
    rt_check_array_like(value.data, value.len, "slice"); \
    value.data[rt_check_index(value.len, index)] = elem; \
} \
static RT_UNUSED void rt_list_##NAME##_set(rt_list_##NAME* value, int64_t index, CTYPE elem) { \
    rt_check_array_like(value->data, value->len, "list"); \
    value->data[rt_check_index(value->len, index)] = elem; \
} \
static RT_UNUSED rt_slice_##NAME rt_array_##NAME##_slice(rt_array_##NAME value, bool has_start, int64_t start, bool has_end, int64_t end) { \
    rt_check_array_like(value.data, value.len, "array"); \
    rt_range range = rt_check_slice(value.len, has_start, start, has_end, end); \
    CTYPE* data = value.data == NULL ? NULL : value.data + range.start; \
    return (rt_slice_##NAME){data, range.end - range.start}; \
} \
static RT_UNUSED rt_slice_##NAME rt_slice_##NAME##_slice(rt_slice_##NAME value, bool has_start, int64_t start, bool has_end, int64_t end) { \
    rt_check_array_like(value.data, value.len, "slice"); \
    rt_range range = rt_check_slice(value.len, has_start, start, has_end, end); \
    CTYPE* data = value.data == NULL ? NULL : value.data + range.start; \
    return (rt_slice_##NAME){data, range.end - range.start}; \
} \
static RT_UNUSED rt_slice_##NAME rt_list_##NAME##_slice(rt_list_##NAME* value, bool has_start, int64_t start, bool has_end, int64_t end) { \
    rt_check_array_like(value->data, value->len, "list"); \
    rt_range range = rt_check_slice(value->len, has_start, start, has_end, end); \
    CTYPE* data = value->data == NULL ? NULL : value->data + range.start; \
    return (rt_slice_##NAME){data, range.end - range.start}; \
} \
static RT_UNUSED void rt_list_##NAME##_free(void* ptr) { \
    rt_list_##NAME* list = (rt_list_##NAME*)ptr; \
    free(list->data); \
    free(list); \
} \
static RT_UNUSED rt_list_##NAME* rt_list_##NAME##_new(rt_arena* arena, size_t cap) { \
    rt_list_##NAME* list = (rt_list_##NAME*)malloc(sizeof(rt_list_##NAME)); \
    if (list == NULL) { \
        rt_runtime_fail("allocation failed"); \
    } \
    list->data = NULL; \
    list->len = 0; \
    list->cap = cap; \
    if (cap > 0) { \
        list->data = (CTYPE*)malloc(rt_checked_bytes(cap, sizeof(CTYPE))); \
        if (list->data == NULL) { \
            free(list); \
            rt_runtime_fail("allocation failed"); \
        } \
    } \
    rt_arena_register_list(arena, list, rt_list_##NAME##_free); \
    return list; \
} \
static RT_UNUSED rt_list_##NAME* rt_list_##NAME##_from_values(rt_arena* arena, CTYPE const* values, size_t len) { \
    rt_check_array_like(values, len, "list values"); \
    rt_list_##NAME* list = rt_list_##NAME##_new(arena, len); \
    if (len > 0) { \
        memcpy(list->data, values, len * sizeof(CTYPE)); \
    } \
    list->len = len; \
    return list; \
} \
static RT_UNUSED rt_list_##NAME* rt_list_##NAME##_clone(rt_arena* arena, rt_list_##NAME* value) { \
    rt_check_array_like(value->data, value->len, "list"); \
    rt_list_##NAME* list = rt_list_##NAME##_new(arena, value->len); \
    for (size_t i = 0; i < value->len; i++) { \
        list->data[i] = RT_CLONE_VALUE_##NAME(arena, value->data[i]); \
    } \
    list->len = value->len; \
    return list; \
} \
static RT_UNUSED void rt_list_##NAME##_append(rt_list_##NAME* list, CTYPE elem) { \
    rt_check_array_like(list->data, list->len, "list"); \
    if (list->len == list->cap) { \
        size_t new_cap = list->cap == 0 ? 4 : list->cap * 2; \
        if (new_cap < list->cap || new_cap > SIZE_MAX / sizeof(CTYPE)) { \
            rt_runtime_fail("list capacity overflow"); \
        } \
        CTYPE* data = (CTYPE*)realloc(list->data, new_cap * sizeof(CTYPE)); \
        if (data == NULL) { \
            rt_runtime_fail("allocation failed"); \
        } \
        list->data = data; \
        list->cap = new_cap; \
    } \
    list->data[list->len] = elem; \
    list->len++; \
}

RT_DEFINE_COLLECTIONS(int, int64_t)
RT_DEFINE_COLLECTIONS(float, double)
RT_DEFINE_COLLECTIONS(bool, bool)
RT_DEFINE_COLLECTIONS(string, rt_string)

#ifdef TRUX_RUNTIME_CUDA
typedef struct { int64_t* data; size_t len; } rt_gpu_buffer_int;
typedef struct { double* data; size_t len; } rt_gpu_buffer_float;

static RT_UNUSED void rt_cuda_check(cudaError_t err, const char* operation) {
    if (err != cudaSuccess) {
        fprintf(stderr, "trux runtime error: CUDA %s failed: %s\n", operation, cudaGetErrorString(err));
        exit(1);
    }
}

static RT_UNUSED rt_gpu_buffer_int rt_gpu_alloc_int(int64_t count) {
    size_t len = rt_checked_count(count, "gpu buffer length");
    int64_t* data = NULL;
    rt_cuda_check(cudaMalloc((void**)&data, rt_checked_bytes(len, sizeof(int64_t))), "alloc");
    return (rt_gpu_buffer_int){data, len};
}

static RT_UNUSED rt_gpu_buffer_float rt_gpu_alloc_float(int64_t count) {
    size_t len = rt_checked_count(count, "gpu buffer length");
    double* data = NULL;
    rt_cuda_check(cudaMalloc((void**)&data, rt_checked_bytes(len, sizeof(double))), "alloc");
    return (rt_gpu_buffer_float){data, len};
}

static RT_UNUSED void rt_gpu_copy_to_device_int(rt_slice_int host, rt_gpu_buffer_int device) {
    if (host.len > device.len) rt_runtime_fail("gpu copyToDevice host slice is larger than device buffer");
    rt_cuda_check(cudaMemcpy(device.data, host.data, host.len * sizeof(int64_t), cudaMemcpyHostToDevice), "copyToDevice");
}

static RT_UNUSED void rt_gpu_copy_to_device_float(rt_slice_float host, rt_gpu_buffer_float device) {
    if (host.len > device.len) rt_runtime_fail("gpu copyToDevice host slice is larger than device buffer");
    rt_cuda_check(cudaMemcpy(device.data, host.data, host.len * sizeof(double), cudaMemcpyHostToDevice), "copyToDevice");
}

static RT_UNUSED void rt_gpu_copy_to_host_int(rt_gpu_buffer_int device, rt_slice_int host) {
    if (host.len > device.len) rt_runtime_fail("gpu copyToHost host slice is larger than device buffer");
    rt_cuda_check(cudaMemcpy(host.data, device.data, host.len * sizeof(int64_t), cudaMemcpyDeviceToHost), "copyToHost");
}

static RT_UNUSED void rt_gpu_copy_to_host_float(rt_gpu_buffer_float device, rt_slice_float host) {
    if (host.len > device.len) rt_runtime_fail("gpu copyToHost host slice is larger than device buffer");
    rt_cuda_check(cudaMemcpy(host.data, device.data, host.len * sizeof(double), cudaMemcpyDeviceToHost), "copyToHost");
}

static RT_UNUSED void rt_gpu_free_int(rt_gpu_buffer_int buffer) {
    rt_cuda_check(cudaFree(buffer.data), "free");
}

static RT_UNUSED void rt_gpu_free_float(rt_gpu_buffer_float buffer) {
    rt_cuda_check(cudaFree(buffer.data), "free");
}

static RT_UNUSED void rt_gpu_sync(void) {
    rt_cuda_check(cudaDeviceSynchronize(), "sync");
}
#endif

typedef struct {
    int64_t width;
    int64_t height;
    int64_t max_value;
    size_t pixel_start;
} rt_ppm_header;

static RT_UNUSED void rt_ppm_skip_space_and_comments(rt_string contents, size_t* index) {
    rt_check_string(contents);
    for (;;) {
        while (*index < contents.len && isspace((unsigned char)contents.data[*index])) {
            (*index)++;
        }
        if (*index >= contents.len || contents.data[*index] != '#') {
            return;
        }
        while (*index < contents.len && contents.data[*index] != '\n' && contents.data[*index] != '\r') {
            (*index)++;
        }
    }
}

static RT_UNUSED int64_t rt_ppm_read_int_token(rt_string contents, size_t* index, const char* what) {
    rt_ppm_skip_space_and_comments(contents, index);
    if (*index >= contents.len || !isdigit((unsigned char)contents.data[*index])) {
        fprintf(stderr, "trux runtime error: expected ppm %s\n", what);
        exit(1);
    }

    int64_t value = 0;
    while (*index < contents.len && isdigit((unsigned char)contents.data[*index])) {
        int digit = contents.data[*index] - '0';
        if (value > (INT64_MAX - digit) / 10) {
            rt_runtime_fail("ppm integer overflow");
        }
        value = value * 10 + digit;
        (*index)++;
    }
    return value;
}

static RT_UNUSED rt_ppm_header rt_ppm_read_header(rt_string contents) {
    rt_check_string(contents);
    size_t index = 0;
    rt_ppm_skip_space_and_comments(contents, &index);
    if (index + 2 > contents.len || contents.data[index] != 'P' || contents.data[index + 1] != '3') {
        rt_runtime_fail("expected P3 ppm image");
    }
    index += 2;

    int64_t width = rt_ppm_read_int_token(contents, &index, "width");
    int64_t height = rt_ppm_read_int_token(contents, &index, "height");
    int64_t max_value = rt_ppm_read_int_token(contents, &index, "max value");
    if (width <= 0 || height <= 0) {
        rt_runtime_fail("ppm width and height must be positive");
    }
    if (max_value != 255) {
        rt_runtime_fail("only 8-bit P3 ppm images with max value 255 are supported");
    }
    rt_ppm_skip_space_and_comments(contents, &index);
    return (rt_ppm_header){width, height, max_value, index};
}

static RT_UNUSED size_t rt_ppm_pixel_count(rt_ppm_header header) {
    size_t width = rt_checked_positive_count(header.width, "ppm width");
    size_t height = rt_checked_positive_count(header.height, "ppm height");
    size_t pixels = rt_checked_bytes(width, height);
    return rt_checked_bytes(pixels, 3);
}

static RT_UNUSED int64_t rt_image_width(rt_string path) {
    rt_arena arena;
    rt_arena_init(&arena);
    rt_ppm_header header = rt_ppm_read_header(rt_read_file(&arena, path));
    int64_t width = header.width;
    rt_arena_deinit(&arena);
    return width;
}

static RT_UNUSED int64_t rt_image_height(rt_string path) {
    rt_arena arena;
    rt_arena_init(&arena);
    rt_ppm_header header = rt_ppm_read_header(rt_read_file(&arena, path));
    int64_t height = header.height;
    rt_arena_deinit(&arena);
    return height;
}

static RT_UNUSED rt_slice_int rt_read_ppm(rt_arena* arena, rt_string path) {
    rt_string contents = rt_read_file(arena, path);
    rt_ppm_header header = rt_ppm_read_header(contents);
    size_t count = rt_ppm_pixel_count(header);
    if (count > (size_t)INT64_MAX) {
        rt_runtime_fail("ppm image too large");
    }

    rt_slice_int pixels = rt_make_slice_int(arena, (int64_t)count);
    size_t index = header.pixel_start;
    for (size_t i = 0; i < count; i++) {
        int64_t value = rt_ppm_read_int_token(contents, &index, "pixel value");
        if (value < 0 || value > 255) {
            rt_runtime_fail("ppm pixel value must be between 0 and 255");
        }
        pixels.data[i] = value;
    }

    rt_ppm_skip_space_and_comments(contents, &index);
    if (index < contents.len) {
        rt_runtime_fail("unexpected trailing data in ppm image");
    }
    return pixels;
}

static RT_UNUSED void rt_write_ppm(rt_string path, rt_slice_int pixels, int64_t width_value, int64_t height_value) {
    size_t width = rt_checked_positive_count(width_value, "ppm width");
    size_t height = rt_checked_positive_count(height_value, "ppm height");
    size_t expected = rt_checked_bytes(rt_checked_bytes(width, height), 3);
    rt_check_array_like(pixels.data, pixels.len, "ppm pixels");
    if (pixels.len != expected) {
        fprintf(stderr, "trux runtime error: ppm pixel count %zu does not match %zux%zux3\n", pixels.len, width, height);
        exit(1);
    }

    char* c_path = rt_string_to_c_string(path, "file path");
    FILE* file = fopen(c_path, "wb");
    if (file == NULL) {
        rt_file_fail("open ppm for writing", c_path);
    }
    if (fprintf(file, "P3\n%zu %zu\n255\n", width, height) < 0) {
        rt_runtime_fail("write failed");
    }

    for (size_t i = 0; i < pixels.len; i++) {
        int64_t value = pixels.data[i];
        if (value < 0 || value > 255) {
            rt_runtime_fail("ppm pixel value must be between 0 and 255");
        }
        if (fprintf(file, "%" PRId64, value) < 0) {
            rt_runtime_fail("write failed");
        }
        if ((i + 1) % 12 == 0 || i + 1 == pixels.len) {
            rt_file_write_byte(file, '\n');
        } else {
            rt_file_write_byte(file, ' ');
        }
    }

    if (fclose(file) != 0) {
        rt_file_fail("close ppm after writing", c_path);
    }
    free(c_path);
}

static RT_UNUSED bool rt_csv_is_newline(uint8_t ch) {
    return ch == '\n' || ch == '\r';
}

static RT_UNUSED void rt_csv_consume_newline(rt_string contents, size_t* index) {
    uint8_t ch = contents.data[*index];
    (*index)++;
    if (ch == '\r' && *index < contents.len && contents.data[*index] == '\n') {
        (*index)++;
    }
}

static RT_UNUSED void rt_csv_check_row_columns(size_t got, size_t want) {
    if (got != want) {
        fprintf(stderr, "trux runtime error: csv row has %zu columns, want %zu\n", got, want);
        exit(1);
    }
}

static RT_UNUSED rt_string rt_csv_finish_cell(rt_arena* arena, rt_byte_buffer* field) {
    rt_string cell = rt_byte_buffer_to_string(arena, field);
    rt_byte_buffer_free(field);
    return cell;
}

static RT_UNUSED rt_list_string* rt_read_csv(rt_arena* arena, rt_string path, int64_t columns_value) {
    size_t columns = rt_checked_positive_count(columns_value, "csv columns");
    rt_string contents = rt_read_file(arena, path);
    rt_list_string* cells = rt_list_string_new(arena, 0);
    if (contents.len == 0) {
        return cells;
    }

    size_t index = 0;
    size_t row_columns = 0;
    bool pending_empty = false;

    while (index < contents.len || pending_empty) {
        pending_empty = false;
        rt_byte_buffer field;
        rt_byte_buffer_init(&field);

        if (index < contents.len && contents.data[index] == '"') {
            bool closed = false;
            index++;
            while (index < contents.len) {
                uint8_t ch = contents.data[index];
                index++;
                if (ch != '"') {
                    rt_byte_buffer_append_byte(&field, ch);
                    continue;
                }
                if (index < contents.len && contents.data[index] == '"') {
                    rt_byte_buffer_append_byte(&field, '"');
                    index++;
                    continue;
                }
                closed = true;
                break;
            }
            if (!closed) {
                rt_runtime_fail("unterminated csv quoted field");
            }
            if (index < contents.len && contents.data[index] != ',' && !rt_csv_is_newline(contents.data[index])) {
                rt_runtime_fail("expected csv delimiter after quoted field");
            }
        } else {
            while (index < contents.len && contents.data[index] != ',' && !rt_csv_is_newline(contents.data[index])) {
                if (contents.data[index] == '"') {
                    rt_runtime_fail("unexpected quote in csv field");
                }
                rt_byte_buffer_append_byte(&field, contents.data[index]);
                index++;
            }
        }

        rt_list_string_append(cells, rt_csv_finish_cell(arena, &field));
        row_columns++;

        if (index >= contents.len) {
            rt_csv_check_row_columns(row_columns, columns);
            break;
        }
        if (contents.data[index] == ',') {
            index++;
            if (index >= contents.len) {
                pending_empty = true;
            }
            continue;
        }
        if (rt_csv_is_newline(contents.data[index])) {
            rt_csv_consume_newline(contents, &index);
            rt_csv_check_row_columns(row_columns, columns);
            row_columns = 0;
            continue;
        }

        rt_runtime_fail("invalid csv parser state");
    }

    return cells;
}

static RT_UNUSED bool rt_csv_cell_needs_quotes(rt_string cell) {
    rt_check_string(cell);
    for (size_t i = 0; i < cell.len; i++) {
        uint8_t ch = cell.data[i];
        if (ch == ',' || ch == '"' || ch == '\n' || ch == '\r') {
            return true;
        }
    }
    return false;
}

static RT_UNUSED void rt_csv_write_cell(FILE* file, rt_string cell) {
    if (!rt_csv_cell_needs_quotes(cell)) {
        rt_file_write_string(file, cell);
        return;
    }

    rt_file_write_byte(file, '"');
    for (size_t i = 0; i < cell.len; i++) {
        if (cell.data[i] == '"') {
            rt_file_write_byte(file, '"');
        }
        rt_file_write_byte(file, cell.data[i]);
    }
    rt_file_write_byte(file, '"');
}

static RT_UNUSED void rt_write_csv(rt_string path, rt_list_string* cells, int64_t columns_value) {
    size_t columns = rt_checked_positive_count(columns_value, "csv columns");
    rt_check_array_like(cells->data, cells->len, "csv cells");
    if (cells->len % columns != 0) {
        fprintf(stderr, "trux runtime error: csv cell count %zu is not divisible by columns %zu\n", cells->len, columns);
        exit(1);
    }

    char* c_path = rt_string_to_c_string(path, "file path");
    FILE* file = fopen(c_path, "wb");
    if (file == NULL) {
        rt_file_fail("open csv for writing", c_path);
    }

    for (size_t i = 0; i < cells->len; i++) {
        if (i > 0) {
            rt_file_write_byte(file, i % columns == 0 ? '\n' : ',');
        }
        rt_csv_write_cell(file, cells->data[i]);
    }
    if (cells->len > 0) {
        rt_file_write_byte(file, '\n');
    }

    if (fclose(file) != 0) {
        rt_file_fail("close csv after writing", c_path);
    }
    free(c_path);
}

#endif
`

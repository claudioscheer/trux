package c

const HeaderName = "trux_runtime.h"

const Source = `#ifndef TRUX_RUNTIME_H
#define TRUX_RUNTIME_H

#define _POSIX_C_SOURCE 200809L
#include <ctype.h>
#include <errno.h>
#include <math.h>
#include <stdbool.h>
#include <stdint.h>
#include <stddef.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <inttypes.h>
#include <time.h>

#include <sys/types.h>
#include <sys/stat.h>
#include <dirent.h>
#include <unistd.h>

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
#define RT_LIT(TYPE, ...) TYPE{__VA_ARGS__}
template <typename T, size_t N>
struct rt_c_array {
    T values[N];
#if defined(__CUDACC__)
    __host__ __device__
#endif
    operator T*() { return values; }
};
#define RT_ARRAY(T, N, ...) (rt_c_array<T, N>{{__VA_ARGS__}})
#else
#define RT_ALIGNOF(T) _Alignof(T)
#define RT_LIT(TYPE, ...) ((TYPE){__VA_ARGS__})
#define RT_ARRAY(T, N, ...) ((T[]){__VA_ARGS__})
#endif

#define RT_ARENA_DEFAULT_CHUNK_SIZE ((size_t)4096)

static RT_UNUSED void rt_stdio_binary(void) {
}

typedef struct {
    const uint8_t* data;
    size_t len;
} rt_string;

typedef struct rt_error_data {
    rt_string message;
} rt_error_data;

typedef rt_error_data* rt_error;

typedef struct {
    rt_string f0;
    rt_error f1;
} trux_tuple_string_error;

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

typedef struct rt_context rt_context;

typedef struct {
    rt_arena_chunk* chunks;
    rt_list_allocation* lists;
    rt_context* owner;
} rt_arena;

typedef struct {
    rt_arena_chunk* chunk;
    size_t used;
    rt_list_allocation* lists;
} rt_arena_mark;

static RT_UNUSED rt_error rt_error_new(rt_arena* arena, rt_string message);
static RT_UNUSED rt_string rt_error_message(rt_error err);
static RT_UNUSED rt_error rt_error_clone(rt_arena* arena, rt_error err);
static RT_UNUSED void rt_print_error(rt_error err);

struct rt_context {
    rt_arena* arena;
    rt_arena_chunk* chunk_pool;
};

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

static RT_UNUSED int64_t rt_rem_i64_at(int64_t left, int64_t right, const char* file, int line) {
    if (right == 0) {
        rt_arithmetic_fail("remainder by zero", file, line);
    }
    if (left == INT64_MIN && right == -1) {
        rt_arithmetic_fail("overflow in %", file, line);
    }
    return left % right;
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

__device__ static int64_t rt_rem_i64_device(int64_t left, int64_t right) {
    if (right == 0 || (left == INT64_MIN && right == -1)) {
        asm("trap;");
    }
    return left % right;
}
#endif

#if defined(TRUX_RUNTIME_CUDA) && defined(__CUDA_ARCH__)
#define rt_add_i64(LEFT, RIGHT) rt_add_i64_device((LEFT), (RIGHT))
#define rt_sub_i64(LEFT, RIGHT) rt_sub_i64_device((LEFT), (RIGHT))
#define rt_mul_i64(LEFT, RIGHT) rt_mul_i64_device((LEFT), (RIGHT))
#define rt_div_i64(LEFT, RIGHT) rt_div_i64_device((LEFT), (RIGHT))
#define rt_rem_i64(LEFT, RIGHT) rt_rem_i64_device((LEFT), (RIGHT))
#else
#define rt_add_i64(LEFT, RIGHT) rt_add_i64_at((LEFT), (RIGHT), __FILE__, __LINE__)
#define rt_sub_i64(LEFT, RIGHT) rt_sub_i64_at((LEFT), (RIGHT), __FILE__, __LINE__)
#define rt_mul_i64(LEFT, RIGHT) rt_mul_i64_at((LEFT), (RIGHT), __FILE__, __LINE__)
#define rt_div_i64(LEFT, RIGHT) rt_div_i64_at((LEFT), (RIGHT), __FILE__, __LINE__)
#define rt_rem_i64(LEFT, RIGHT) rt_rem_i64_at((LEFT), (RIGHT), __FILE__, __LINE__)
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

static RT_UNUSED size_t rt_arena_next_chunk_cap(const rt_arena* arena, size_t min_cap) {
    size_t cap = RT_ARENA_DEFAULT_CHUNK_SIZE;
    if (arena->chunks != NULL && arena->chunks->cap <= SIZE_MAX / 2) {
        size_t doubled = arena->chunks->cap * 2;
        if (doubled > cap) {
            cap = doubled;
        }
    }
    if (min_cap > cap) {
        cap = min_cap;
    }
    return cap;
}

static RT_UNUSED rt_arena_chunk* rt_arena_new_chunk(rt_arena* arena, size_t min_cap) {
    size_t cap = rt_arena_next_chunk_cap(arena, min_cap);
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
    arena->owner = NULL;
}

static RT_UNUSED void rt_arena_init_frame(rt_arena* arena, rt_context* ctx) {
    rt_arena_init(arena);
    arena->owner = ctx;
}

static RT_UNUSED void rt_context_free_chunks(rt_context* ctx) {
    if (ctx == NULL) {
        return;
    }
    rt_arena_chunk* chunk = ctx->chunk_pool;
    while (chunk != NULL) {
        rt_arena_chunk* next = chunk->next;
        free(chunk);
        chunk = next;
    }
    ctx->chunk_pool = NULL;
}

static RT_UNUSED rt_arena_chunk* rt_arena_take_chunk(rt_arena* arena, size_t min_cap) {
    if (arena->owner != NULL) {
        rt_arena_chunk** link = &arena->owner->chunk_pool;
        rt_arena_chunk* chunk = arena->owner->chunk_pool;
        while (chunk != NULL) {
            if (chunk->cap >= min_cap) {
                *link = chunk->next;
                chunk->next = NULL;
                chunk->used = 0;
                return chunk;
            }
            link = &chunk->next;
            chunk = chunk->next;
        }
    }
    return rt_arena_new_chunk(arena, min_cap);
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
        chunk->used = 0;
        if (arena->owner != NULL) {
            chunk->next = arena->owner->chunk_pool;
            arena->owner->chunk_pool = chunk;
        } else {
            free(chunk);
        }
        chunk = next;
    }
    arena->chunks = NULL;
}

typedef struct {
    rt_arena* arena;
    rt_context* ctx;
    int live;
    int completions;
    int sends;
} rt_http_park_slot;

static RT_UNUSED int rt_http_park_begin(rt_http_park_slot* slot, rt_arena* stack_arena, rt_context* stack_ctx) {
    if (slot == NULL || stack_arena == NULL || stack_ctx == NULL || slot->live) return 0;
    rt_arena* arena = (rt_arena*)malloc(sizeof(rt_arena));
    if (arena == NULL) return 0;
    rt_context* ctx = (rt_context*)malloc(sizeof(rt_context));
    if (ctx == NULL) {
        free(arena);
        return 0;
    }
    *arena = *stack_arena;
    stack_arena->chunks = NULL;
    stack_arena->lists = NULL;
    stack_arena->owner = NULL;
    *ctx = *stack_ctx;
    ctx->arena = arena;
    stack_ctx->chunk_pool = NULL;
    slot->arena = arena;
    slot->ctx = ctx;
    slot->live = 1;
    return 1;
}

static RT_UNUSED int rt_http_park_complete(rt_http_park_slot* slot, int responded) {
    (void)responded;
    if (slot == NULL || !slot->live) return 0;
    slot->live = 0;
    slot->completions++;
    slot->sends++;
    rt_context_free_chunks(slot->ctx);
    rt_arena_deinit(slot->arena);
    free(slot->ctx);
    free(slot->arena);
    slot->arena = NULL;
    slot->ctx = NULL;
    return 1;
}

static RT_UNUSED rt_arena_mark rt_arena_mark_current(rt_arena* arena) {
    return RT_LIT(rt_arena_mark, 
        arena->chunks,
        arena->chunks == NULL ? 0 : arena->chunks->used,
        arena->lists,
    );
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

static RT_UNUSED void rt_arena_track_list(rt_arena* arena, rt_list_allocation* node, void* list, void (*free_list)(void*)) {
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
        chunk = rt_arena_take_chunk(arena, aligned_size);
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
    return RT_LIT(rt_range, (size_t)start, (size_t)end);
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

static RT_UNUSED void* rt_check_not_null_at(const void* ptr, const char* file, int line, int column) {
    if (ptr == NULL) {
        if (file != NULL && file[0] != '\0' && line > 0) {
            fprintf(stderr, "trux runtime error: nil pointer dereference\n  at %s:%d:%d\n", file, line, column);
        } else {
            fprintf(stderr, "trux runtime error: nil pointer dereference\n");
        }
        exit(1);
    }
    return (void*)ptr;
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
        return RT_LIT(rt_string, NULL, 0);
    }
    uint8_t* data = (uint8_t*)rt_arena_alloc(arena, buffer->len);
    memcpy(data, buffer->data, buffer->len);
    return RT_LIT(rt_string, data, buffer->len);
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
    return RT_LIT(rt_string, value.data == NULL ? NULL : value.data + start, end - start);
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

static RT_UNUSED int rt_clock_realtime(struct timespec* ts) {
#if defined(__CUDA_ARCH__)
    (void)ts;
    return -1;
#else
    return clock_gettime(CLOCK_REALTIME, ts);
#endif
}

static RT_UNUSED int rt_clock_monotonic(struct timespec* ts) {
#if defined(__CUDA_ARCH__)
    (void)ts;
    return -1;
#else
    return clock_gettime(CLOCK_MONOTONIC, ts);
#endif
}

static RT_UNUSED int64_t rt_time_now_unix_millis(void) {
    struct timespec ts;
    if (rt_clock_realtime(&ts) != 0) {
        rt_runtime_fail("clock_gettime failed");
    }
    return rt_time_from_timespec_millis(ts);
}

static RT_UNUSED int64_t rt_time_monotonic_nanos(void) {
    struct timespec ts;
    if (rt_clock_monotonic(&ts) != 0) {
        rt_runtime_fail("clock_gettime failed");
    }
    return rt_time_from_timespec_nanos(ts);
}

static RT_UNUSED void rt_time_sleep_millis(int64_t millis) {
    size_t checked = rt_checked_count(millis, "sleep milliseconds");
#if defined(__CUDA_ARCH__)
    (void)checked;
#else
    struct timespec remaining;
    remaining.tv_sec = (time_t)(checked / 1000);
    remaining.tv_nsec = (long)((checked % 1000) * 1000000);
    while (nanosleep(&remaining, &remaining) != 0) {
        if (errno != EINTR) {
            rt_runtime_fail("sleep failed");
        }
    }
#endif
}

static RT_UNUSED trux_tuple_string_error rt_read_file(rt_arena* arena, rt_string path) {
    char* c_path = rt_string_to_c_string(path, "file path");
    FILE* file = fopen(c_path, "rb");
    if (file == NULL) {
        free(c_path);
        return RT_LIT(trux_tuple_string_error, RT_LIT(rt_string, NULL, 0), rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"failed to open file for reading", 31)));
    }
    if (fseek(file, 0, SEEK_END) != 0) {
        fclose(file);
        free(c_path);
        return RT_LIT(trux_tuple_string_error, RT_LIT(rt_string, NULL, 0), rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"failed to seek file", 19)));
    }
    long file_size = ftell(file);
    if (file_size < 0) {
        fclose(file);
        free(c_path);
        return RT_LIT(trux_tuple_string_error, RT_LIT(rt_string, NULL, 0), rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"failed to tell file size", 24)));
    }
    if (fseek(file, 0, SEEK_SET) != 0) {
        fclose(file);
        free(c_path);
        return RT_LIT(trux_tuple_string_error, RT_LIT(rt_string, NULL, 0), rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"failed to rewind file", 21)));
    }

    size_t size = (size_t)file_size;
    uint8_t* data = NULL;
    if (size > 0) {
        data = (uint8_t*)rt_arena_alloc(arena, size);
        if (fread(data, 1, size, file) != size) {
            fclose(file);
            free(c_path);
            return RT_LIT(trux_tuple_string_error, RT_LIT(rt_string, NULL, 0), rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"read failed", 11)));
        }
    }
    if (fclose(file) != 0) {
        free(c_path);
        return RT_LIT(trux_tuple_string_error, RT_LIT(rt_string, NULL, 0), rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"failed to close file", 20)));
    }
    free(c_path);
    return RT_LIT(trux_tuple_string_error, RT_LIT(rt_string, data, size), NULL);
}

static RT_UNUSED rt_error rt_write_file(rt_arena* arena, rt_string path, rt_string contents) {
    char* c_path = rt_string_to_c_string(path, "file path");
    FILE* file = fopen(c_path, "wb");
    if (file == NULL) {
        free(c_path);
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"failed to open file for writing", 31));
    }

    if (contents.len > 0) {
        if (fwrite(contents.data, 1, contents.len, file) != contents.len) {
            fclose(file);
            free(c_path);
            return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"failed to write file contents", 29));
        }
    }
    if (fclose(file) != 0) {
        free(c_path);
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"failed to close file after writing", 34));
    }
    free(c_path);
    return NULL;
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
        return RT_LIT(rt_string, NULL, 0);
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
    return RT_LIT(rt_string, data, len);
}

static RT_UNUSED rt_string rt_string_clone(rt_arena* arena, rt_string value) {
    if (value.len == 0) {
        return RT_LIT(rt_string, NULL, 0);
    }
    rt_check_string(value);
    uint8_t* data = (uint8_t*)rt_arena_alloc(arena, value.len);
    memcpy(data, value.data, value.len);
    return RT_LIT(rt_string, data, value.len);
}

static RT_UNUSED rt_string rt_string_index(rt_string value, int64_t index) {
    rt_check_string(value);
    size_t checked = rt_check_index(value.len, index);
    return RT_LIT(rt_string, value.data + checked, 1);
}

static RT_UNUSED rt_string rt_string_slice(rt_string value, bool has_start, int64_t start, bool has_end, int64_t end) {
    rt_check_string(value);
    rt_range range = rt_check_slice(value.len, has_start, start, has_end, end);
    const uint8_t* data = value.data == NULL ? NULL : value.data + range.start;
    return RT_LIT(rt_string, data, range.end - range.start);
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

static RT_UNUSED rt_error rt_error_new(rt_arena* arena, rt_string message) {
    rt_error err = (rt_error)rt_arena_alloc(arena, sizeof(rt_error_data));
    err->message = message;
    return err;
}

static RT_UNUSED rt_string rt_error_message(rt_error err) {
    if (err == NULL) {
        return RT_LIT(rt_string, (const uint8_t*)"", 0);
    }
    return err->message;
}

static RT_UNUSED rt_error rt_error_clone(rt_arena* arena, rt_error err) {
    if (err == NULL) {
        return NULL;
    }
    rt_error copied = (rt_error)rt_arena_alloc(arena, sizeof(rt_error_data));
    copied->message = rt_string_clone(arena, err->message);
    return copied;
}

#define rt_clone_error(ARENA, VALUE) rt_error_clone((ARENA), (VALUE))

static RT_UNUSED void rt_print_error(rt_error err) {
    if (err == NULL) {
        printf("<nil>");
    } else {
        rt_print_string(err->message);
    }
}

#define RT_CLONE_VALUE_int(ARENA, VALUE) (VALUE)
#define RT_CLONE_VALUE_float(ARENA, VALUE) (VALUE)
#define RT_CLONE_VALUE_bool(ARENA, VALUE) (VALUE)
#define RT_CLONE_VALUE_string(ARENA, VALUE) rt_string_clone((ARENA), (VALUE))
#define RT_CLONE_VALUE_error(ARENA, VALUE) rt_error_clone((ARENA), (VALUE))

#define RT_DEFINE_COLLECTIONS(NAME, CTYPE) \
typedef struct { \
    CTYPE* data; \
    size_t len; \
} rt_array_##NAME; \
typedef struct { \
    CTYPE* data; \
    size_t len; \
} rt_slice_##NAME; \
typedef struct rt_list_##NAME { \
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
    return RT_LIT(rt_array_##NAME, data, len); \
} \
static RT_UNUSED rt_array_##NAME rt_array_##NAME##_clone(rt_arena* arena, rt_array_##NAME value) { \
    rt_check_array_like(value.data, value.len, "array"); \
    CTYPE* data = (CTYPE*)rt_arena_alloc_count(arena, value.len, sizeof(CTYPE), false); \
    for (size_t i = 0; i < value.len; i++) { \
        data[i] = RT_CLONE_VALUE_##NAME(arena, value.data[i]); \
    } \
    return RT_LIT(rt_array_##NAME, data, value.len); \
} \
static RT_UNUSED rt_slice_##NAME rt_make_slice_##NAME(rt_arena* arena, int64_t count) { \
    size_t len = rt_checked_count(count, "slice length"); \
    CTYPE* data = (CTYPE*)rt_arena_alloc_count(arena, len, sizeof(CTYPE), true); \
    return RT_LIT(rt_slice_##NAME, data, len); \
} \
static RT_UNUSED rt_slice_##NAME rt_slice_##NAME##_clone(rt_arena* arena, rt_slice_##NAME value) { \
    rt_check_array_like(value.data, value.len, "slice"); \
    CTYPE* data = (CTYPE*)rt_arena_alloc_count(arena, value.len, sizeof(CTYPE), false); \
    for (size_t i = 0; i < value.len; i++) { \
        data[i] = RT_CLONE_VALUE_##NAME(arena, value.data[i]); \
    } \
    return RT_LIT(rt_slice_##NAME, data, value.len); \
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
static RT_UNUSED CTYPE* rt_array_##NAME##_addr(rt_array_##NAME value, int64_t index) { \
    rt_check_array_like(value.data, value.len, "array"); \
    return &value.data[rt_check_index(value.len, index)]; \
} \
static RT_UNUSED CTYPE* rt_slice_##NAME##_addr(rt_slice_##NAME value, int64_t index) { \
    rt_check_array_like(value.data, value.len, "slice"); \
    return &value.data[rt_check_index(value.len, index)]; \
} \
static RT_UNUSED CTYPE* rt_list_##NAME##_addr(rt_list_##NAME* value, int64_t index) { \
    rt_check_array_like(value->data, value->len, "list"); \
    return &value->data[rt_check_index(value->len, index)]; \
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
    return RT_LIT(rt_slice_##NAME, data, range.end - range.start); \
} \
static RT_UNUSED rt_slice_##NAME rt_slice_##NAME##_slice(rt_slice_##NAME value, bool has_start, int64_t start, bool has_end, int64_t end) { \
    rt_check_array_like(value.data, value.len, "slice"); \
    rt_range range = rt_check_slice(value.len, has_start, start, has_end, end); \
    CTYPE* data = value.data == NULL ? NULL : value.data + range.start; \
    return RT_LIT(rt_slice_##NAME, data, range.end - range.start); \
} \
static RT_UNUSED rt_slice_##NAME rt_list_##NAME##_slice(rt_list_##NAME* value, bool has_start, int64_t start, bool has_end, int64_t end) { \
    rt_check_array_like(value->data, value->len, "list"); \
    rt_range range = rt_check_slice(value->len, has_start, start, has_end, end); \
    CTYPE* data = value->data == NULL ? NULL : value->data + range.start; \
    return RT_LIT(rt_slice_##NAME, data, range.end - range.start); \
} \
static RT_UNUSED void rt_list_##NAME##_free(void* ptr) { \
    rt_list_##NAME* list = (rt_list_##NAME*)ptr; \
    if (list->data != NULL) { \
        free(list->data); \
    } \
} \
static RT_UNUSED rt_list_##NAME* rt_list_##NAME##_new(rt_arena* arena, size_t cap) { \
    size_t offset = sizeof(rt_list_allocation); \
    size_t align = RT_ALIGNOF(rt_list_##NAME); \
    if (align > 1 && offset % align != 0) { \
        size_t padding = align - (offset % align); \
        if (offset > SIZE_MAX - padding) { \
            rt_runtime_fail("allocation size overflow"); \
        } \
        offset += padding; \
    } \
    if (offset > SIZE_MAX - sizeof(rt_list_##NAME)) { \
        rt_runtime_fail("allocation size overflow"); \
    } \
    uint8_t* block = (uint8_t*)malloc(offset + sizeof(rt_list_##NAME)); \
    if (block == NULL) { \
        rt_runtime_fail("allocation failed"); \
    } \
    rt_list_allocation* node = (rt_list_allocation*)block; \
    rt_list_##NAME* list = (rt_list_##NAME*)(block + offset); \
    list->data = NULL; \
    list->len = 0; \
    list->cap = cap; \
    if (cap > 0) { \
        list->data = (CTYPE*)malloc(rt_checked_bytes(cap, sizeof(CTYPE))); \
        if (list->data == NULL) { \
            free(block); \
            rt_runtime_fail("allocation failed"); \
        } \
    } \
    rt_arena_track_list(arena, node, list, rt_list_##NAME##_free); \
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

static RT_UNUSED uint64_t rt_hash_int(int64_t key) {
    uint64_t x = (uint64_t)key;
    x = (x ^ (x >> 30)) * UINT64_C(0xbf58476d1ce4e5b9);
    x = (x ^ (x >> 27)) * UINT64_C(0x94d049bb133111eb);
    return x ^ (x >> 31);
}

static RT_UNUSED bool rt_eq_int(int64_t a, int64_t b) {
    return a == b;
}

static RT_UNUSED uint64_t rt_hash_bool(bool key) {
    return key ? UINT64_C(0x9e3779b97f4a7c15) : UINT64_C(0xcbf29ce484222325);
}

static RT_UNUSED bool rt_eq_bool(bool a, bool b) {
    return a == b;
}

static RT_UNUSED uint64_t rt_hash_string(rt_string key) {
    rt_check_string(key);
    uint64_t hash = UINT64_C(14695981039346656037);
    for (size_t i = 0; i < key.len; i++) {
        hash ^= (uint64_t)key.data[i];
        hash *= UINT64_C(1099511628211);
    }
    return hash;
}

static RT_UNUSED bool rt_eq_string(rt_string a, rt_string b) {
    return rt_string_equal(a, b);
}

static RT_UNUSED uint64_t rt_hash_ptr(const void* key) {
    uint64_t x = (uint64_t)(uintptr_t)key;
    x = (x ^ (x >> 30)) * UINT64_C(0xbf58476d1ce4e5b9);
    x = (x ^ (x >> 27)) * UINT64_C(0x94d049bb133111eb);
    return x ^ (x >> 31);
}

static RT_UNUSED bool rt_eq_ptr(const void* a, const void* b) {
    return a == b;
}

#define RT_DEFINE_MAP(NAME, KEY_NAME, KEY_CTYPE, VAL_NAME, VAL_CTYPE, KEY_EQ_FN, KEY_HASH_FN, ZERO_VAL) \
typedef struct { \
    uint64_t hash; \
    KEY_CTYPE key; \
    VAL_CTYPE value; \
    bool deleted; \
} rt_map_entry_##NAME; \
typedef struct rt_map_##NAME { \
    rt_map_entry_##NAME* entries; \
    size_t* buckets; \
    size_t len; \
    size_t entries_len; \
    size_t entries_cap; \
    size_t bucket_cap; \
} rt_map_##NAME; \
static RT_UNUSED void rt_map_##NAME##_free(void* ptr) { \
    rt_map_##NAME* m = (rt_map_##NAME*)ptr; \
    if (m->entries != NULL) { \
        free(m->entries); \
    } \
    if (m->buckets != NULL) { \
        free(m->buckets); \
    } \
} \
static RT_UNUSED void rt_map_##NAME##_rehash(rt_map_##NAME* m, size_t min_bucket_cap) { \
    size_t new_bucket_cap = m->bucket_cap == 0 ? 8 : m->bucket_cap * 2; \
    while (new_bucket_cap < min_bucket_cap) { \
        if (new_bucket_cap > SIZE_MAX / 2) { \
            rt_runtime_fail("map capacity overflow"); \
        } \
        new_bucket_cap *= 2; \
    } \
    size_t* new_buckets = (size_t*)malloc(rt_checked_bytes(new_bucket_cap, sizeof(size_t))); \
    if (new_buckets == NULL) { \
        rt_runtime_fail("allocation failed"); \
    } \
    memset(new_buckets, 0, new_bucket_cap * sizeof(size_t)); \
    size_t write_idx = 0; \
    size_t mask = new_bucket_cap - 1; \
    for (size_t i = 0; i < m->entries_len; i++) { \
        if (m->entries[i].deleted) { \
            continue; \
        } \
        if (write_idx != i) { \
            m->entries[write_idx] = m->entries[i]; \
        } \
        size_t b = (size_t)m->entries[write_idx].hash & mask; \
        while (new_buckets[b] != 0) { \
            b = (b + 1) & mask; \
        } \
        new_buckets[b] = write_idx + 1; \
        write_idx++; \
    } \
    m->entries_len = write_idx; \
    if (m->buckets != NULL) { \
        free(m->buckets); \
    } \
    m->buckets = new_buckets; \
    m->bucket_cap = new_bucket_cap; \
} \
static RT_UNUSED rt_map_##NAME* rt_map_##NAME##_new(rt_arena* arena, int64_t initial_cap) { \
    size_t cap = rt_checked_count(initial_cap, "map capacity"); \
    size_t offset = sizeof(rt_list_allocation); \
    size_t align = RT_ALIGNOF(rt_map_##NAME); \
    if (align > 1 && offset % align != 0) { \
        size_t padding = align - (offset % align); \
        if (offset > SIZE_MAX - padding) { \
            rt_runtime_fail("allocation size overflow"); \
        } \
        offset += padding; \
    } \
    if (offset > SIZE_MAX - sizeof(rt_map_##NAME)) { \
        rt_runtime_fail("allocation size overflow"); \
    } \
    uint8_t* block = (uint8_t*)malloc(offset + sizeof(rt_map_##NAME)); \
    if (block == NULL) { \
        rt_runtime_fail("allocation failed"); \
    } \
    rt_list_allocation* node = (rt_list_allocation*)block; \
    rt_map_##NAME* m = (rt_map_##NAME*)(block + offset); \
    m->entries = NULL; \
    m->buckets = NULL; \
    m->len = 0; \
    m->entries_len = 0; \
    m->entries_cap = 0; \
    m->bucket_cap = 0; \
    rt_arena_track_list(arena, node, m, rt_map_##NAME##_free); \
    if (cap > 0) { \
        m->entries = (rt_map_entry_##NAME*)malloc(rt_checked_bytes(cap, sizeof(rt_map_entry_##NAME))); \
        if (m->entries == NULL) { \
            rt_runtime_fail("allocation failed"); \
        } \
        m->entries_cap = cap; \
        if (cap > SIZE_MAX / 2) { \
            rt_runtime_fail("map capacity overflow"); \
        } \
        rt_map_##NAME##_rehash(m, cap * 2); \
    } \
    return m; \
} \
static RT_UNUSED bool rt_map_##NAME##_find_bucket(rt_map_##NAME* m, KEY_CTYPE key, uint64_t hash, size_t* out_bucket) { \
    if (m == NULL || m->bucket_cap == 0) { \
        return false; \
    } \
    size_t mask = m->bucket_cap - 1; \
    size_t b = (size_t)hash & mask; \
    size_t first_tombstone = SIZE_MAX; \
    for (;;) { \
        size_t slot = m->buckets[b]; \
        if (slot == 0) { \
            if (out_bucket != NULL) { \
                *out_bucket = (first_tombstone != SIZE_MAX) ? first_tombstone : b; \
            } \
            return false; \
        } \
        if (slot == SIZE_MAX) { \
            if (first_tombstone == SIZE_MAX) { \
                first_tombstone = b; \
            } \
        } else { \
            rt_map_entry_##NAME* e = &m->entries[slot - 1]; \
            if (!e->deleted && e->hash == hash && KEY_EQ_FN(e->key, key)) { \
                if (out_bucket != NULL) { \
                    *out_bucket = b; \
                } \
                return true; \
            } \
        } \
        b = (b + 1) & mask; \
    } \
} \
static RT_UNUSED int64_t rt_map_##NAME##_len(rt_map_##NAME* m) { \
    if (m == NULL) { \
        return 0; \
    } \
    return rt_checked_len_i64(m->len); \
} \
static RT_UNUSED VAL_CTYPE rt_map_##NAME##_get(rt_map_##NAME* m, KEY_CTYPE key) { \
    uint64_t hash = KEY_HASH_FN(key); \
    size_t b = 0; \
    if (rt_map_##NAME##_find_bucket(m, key, hash, &b)) { \
        return m->entries[m->buckets[b] - 1].value; \
    } \
    return ZERO_VAL; \
} \
static RT_UNUSED VAL_CTYPE rt_map_##NAME##_get_ok(rt_map_##NAME* m, KEY_CTYPE key, bool* out_ok) { \
    uint64_t hash = KEY_HASH_FN(key); \
    size_t b = 0; \
    if (rt_map_##NAME##_find_bucket(m, key, hash, &b)) { \
        if (out_ok != NULL) { \
            *out_ok = true; \
        } \
        return m->entries[m->buckets[b] - 1].value; \
    } \
    if (out_ok != NULL) { \
        *out_ok = false; \
    } \
    return ZERO_VAL; \
} \
static RT_UNUSED void rt_map_##NAME##_set(rt_map_##NAME* m, KEY_CTYPE key, VAL_CTYPE value) { \
    if (m == NULL) { \
        rt_runtime_fail("assignment to entry in nil map"); \
    } \
    if ((m->entries_len + 1) * 2 > m->bucket_cap) { \
        rt_map_##NAME##_rehash(m, m->bucket_cap == 0 ? 8 : m->bucket_cap * 2); \
    } \
    uint64_t hash = KEY_HASH_FN(key); \
    size_t b = 0; \
    if (rt_map_##NAME##_find_bucket(m, key, hash, &b)) { \
        m->entries[m->buckets[b] - 1].value = value; \
        return; \
    } \
    if (m->entries_len == m->entries_cap) { \
        size_t new_cap = m->entries_cap == 0 ? 4 : m->entries_cap * 2; \
        if (new_cap < m->entries_cap || new_cap > SIZE_MAX / sizeof(rt_map_entry_##NAME)) { \
            rt_runtime_fail("map capacity overflow"); \
        } \
        rt_map_entry_##NAME* data = (rt_map_entry_##NAME*)realloc(m->entries, new_cap * sizeof(rt_map_entry_##NAME)); \
        if (data == NULL) { \
            rt_runtime_fail("allocation failed"); \
        } \
        m->entries = data; \
        m->entries_cap = new_cap; \
    } \
    size_t idx = m->entries_len++; \
    m->entries[idx] = RT_LIT(rt_map_entry_##NAME, hash, key, value, false); \
    m->buckets[b] = idx + 1; \
    m->len++; \
} \
static RT_UNUSED void rt_map_##NAME##_delete(rt_map_##NAME* m, KEY_CTYPE key) { \
    if (m == NULL || m->len == 0) { \
        return; \
    } \
    uint64_t hash = KEY_HASH_FN(key); \
    size_t b = 0; \
    if (rt_map_##NAME##_find_bucket(m, key, hash, &b)) { \
        size_t idx = m->buckets[b] - 1; \
        m->entries[idx].deleted = true; \
        m->buckets[b] = SIZE_MAX; \
        m->len--; \
    } \
} \
static RT_UNUSED rt_map_##NAME* rt_map_##NAME##_clone(rt_arena* arena, rt_map_##NAME* value) { \
    if (value == NULL) { \
        return rt_map_##NAME##_new(arena, 0); \
    } \
    rt_map_##NAME* copy = rt_map_##NAME##_new(arena, rt_checked_len_i64(value->len)); \
    for (size_t i = 0; i < value->entries_len; i++) { \
        if (!value->entries[i].deleted) { \
            KEY_CTYPE k = RT_CLONE_VALUE_##KEY_NAME(arena, value->entries[i].key); \
            VAL_CTYPE v = RT_CLONE_VALUE_##VAL_NAME(arena, value->entries[i].value); \
            rt_map_##NAME##_set(copy, k, v); \
        } \
    } \
    return copy; \
}

#define RT_DEFINE_MAP_LISTS(NAME, KEY_NAME, KEY_CTYPE, VAL_NAME, VAL_CTYPE) \
static RT_UNUSED rt_list_##KEY_NAME* rt_map_##NAME##_keys(rt_arena* arena, rt_map_##NAME* m) { \
    rt_list_##KEY_NAME* out = rt_list_##KEY_NAME##_new(arena, m == NULL ? 0 : m->len); \
    if (m != NULL) { \
        for (size_t i = 0; i < m->entries_len; i++) { \
            if (!m->entries[i].deleted) { \
                rt_list_##KEY_NAME##_append(out, m->entries[i].key); \
            } \
        } \
    } \
    return out; \
} \
static RT_UNUSED rt_list_##VAL_NAME* rt_map_##NAME##_values(rt_arena* arena, rt_map_##NAME* m) { \
    rt_list_##VAL_NAME* out = rt_list_##VAL_NAME##_new(arena, m == NULL ? 0 : m->len); \
    if (m != NULL) { \
        for (size_t i = 0; i < m->entries_len; i++) { \
            if (!m->entries[i].deleted) { \
                rt_list_##VAL_NAME##_append(out, m->entries[i].value); \
            } \
        } \
    } \
    return out; \
}

typedef struct {
    rt_list_string* f0;
    rt_error f1;
} trux_tuple_list_string_error;

typedef struct {
    rt_slice_int f0;
    rt_error f1;
} trux_tuple_slice_int_error;

#define RT_TXM_MAX_TENSORS 256
#define RT_TXM_MAX_NAME 64
#define RT_GPU_MAX_WORKSPACES 16
#define RT_GPU_MAX_OWNED 64
#define RT_TXM_DTYPE_INT 1u
#define RT_TXM_DTYPE_FLOAT 2u

typedef struct {
    char name[RT_TXM_MAX_NAME];
    uint32_t name_len;
    uint32_t dtype;
    uint32_t rank;
    uint64_t dims[4];
    uint64_t count;
    void* host;
    double scalar;
    void* device;
} rt_txm_tensor;

typedef struct {
    int loaded;
    uint32_t count;
    rt_txm_tensor tensors[RT_TXM_MAX_TENSORS];
} rt_txm_model;

static rt_txm_model g_rt_txm;

typedef struct {
    char name[RT_TXM_MAX_NAME];
    uint32_t name_len;
    int is_float;
    uint64_t count;
} rt_gpu_ws_decl;

static int g_rt_gpu_workers_started;

static RT_UNUSED void rt_gpu_publish_workers(void) {
    g_rt_gpu_workers_started = 1;
}

static void* g_rt_gpu_owned[RT_GPU_MAX_OWNED];
static int g_rt_gpu_owned_count;

static RT_UNUSED int rt_txm_read_u32(const uint8_t* data, size_t len, size_t* index, uint32_t* out) {
    if (*index + 4 > len) return 0;
    const uint8_t* p = data + *index;
    *out = (uint32_t)p[0] | ((uint32_t)p[1] << 8) | ((uint32_t)p[2] << 16) | ((uint32_t)p[3] << 24);
    *index += 4;
    return 1;
}

static RT_UNUSED int rt_txm_read_u64(const uint8_t* data, size_t len, size_t* index, uint64_t* out) {
    if (*index + 8 > len) return 0;
    const uint8_t* p = data + *index;
    uint64_t value = 0;
    for (int i = 0; i < 8; i++) {
        value |= ((uint64_t)p[i]) << (8 * i);
    }
    *out = value;
    *index += 8;
    return 1;
}

static RT_UNUSED void rt_txm_reset(void) {
    for (uint32_t i = 0; i < g_rt_txm.count; i++) {
        free(g_rt_txm.tensors[i].host);
        g_rt_txm.tensors[i].host = NULL;
    }
    memset(&g_rt_txm, 0, sizeof(g_rt_txm));
}

static RT_UNUSED const char* rt_txm_parse(const uint8_t* data, size_t len) {
    rt_txm_reset();
    if (data == NULL || len < 12) return "model file is truncated";
    if (memcmp(data, "TXM1", 4) != 0) return "model file has a bad magic";
    size_t index = 4;
    uint32_t version = 0;
    uint32_t count = 0;
    if (!rt_txm_read_u32(data, len, &index, &version)) return "model file is truncated";
    if (version != 1) return "model file has a bad version";
    if (!rt_txm_read_u32(data, len, &index, &count)) return "model file is truncated";
    if (count == 0 || count > RT_TXM_MAX_TENSORS) return "model file has a bad tensor count";
    for (uint32_t n = 0; n < count; n++) {
        uint32_t name_len = 0;
        uint32_t dtype = 0;
        uint32_t rank = 0;
        if (!rt_txm_read_u32(data, len, &index, &name_len)) return "model file is truncated";
        if (name_len == 0 || name_len > RT_TXM_MAX_NAME) return "model tensor name has a bad length";
        if (index + name_len > len) return "model file is truncated";
        rt_txm_tensor* tensor = &g_rt_txm.tensors[n];
        memcpy(tensor->name, data + index, name_len);
        tensor->name_len = name_len;
        index += name_len;
        for (uint32_t prev = 0; prev < n; prev++) {
            rt_txm_tensor* other = &g_rt_txm.tensors[prev];
            if (other->name_len == name_len && memcmp(other->name, tensor->name, name_len) == 0) {
                return "model file has a duplicate tensor name";
            }
        }
        if (!rt_txm_read_u32(data, len, &index, &dtype)) return "model file is truncated";
        if (dtype != RT_TXM_DTYPE_INT && dtype != RT_TXM_DTYPE_FLOAT) return "model tensor has a bad dtype";
        if (!rt_txm_read_u32(data, len, &index, &rank)) return "model file is truncated";
        if (rank < 1 || rank > 4) return "model tensor rank must be from 1 to 4";
        tensor->dtype = dtype;
        tensor->rank = rank;
        uint64_t elements = 1;
        for (uint32_t axis = 0; axis < rank; axis++) {
            uint64_t dim = 0;
            if (!rt_txm_read_u64(data, len, &index, &dim)) return "model file is truncated";
            if (dim == 0) return "model tensor dimension must be at least 1";
            if (elements > UINT64_MAX / dim) return "model tensor dimension product overflows";
            elements *= dim;
            tensor->dims[axis] = dim;
        }
        size_t elem_size = dtype == RT_TXM_DTYPE_FLOAT ? sizeof(double) : sizeof(int64_t);
        if (elements > SIZE_MAX / elem_size) return "model tensor dimension product overflows";
        size_t bytes = (size_t)elements * elem_size;
        if (index + bytes < index || index + bytes > len) return "model file is truncated";
        void* host = malloc(bytes == 0 ? 1 : bytes);
        if (host == NULL) return "model tensor allocation failed";
        memcpy(host, data + index, bytes);
        index += bytes;
        tensor->count = elements;
        tensor->host = host;
        g_rt_txm.count = n + 1;
        if (dtype == RT_TXM_DTYPE_FLOAT && elements == 1) {
            memcpy(&tensor->scalar, host, sizeof(double));
        }
    }
    if (index != len) return "model file has trailing bytes";
    g_rt_txm.count = count;
    return NULL;
}

static RT_UNUSED rt_txm_tensor* rt_txm_find(rt_string name) {
    if (!g_rt_txm.loaded && g_rt_txm.count == 0) return NULL;
    for (uint32_t i = 0; i < g_rt_txm.count; i++) {
        rt_txm_tensor* tensor = &g_rt_txm.tensors[i];
        if (tensor->name_len == name.len && name.len > 0 && memcmp(tensor->name, name.data, name.len) == 0) {
            return tensor;
        }
    }
    return NULL;
}

static RT_UNUSED int rt_gpu_is_owned(void* pointer) {
    if (pointer == NULL) return 0;
    for (uint32_t i = 0; i < g_rt_txm.count; i++) {
        if (g_rt_txm.tensors[i].device == pointer) return 1;
    }
    for (int i = 0; i < g_rt_gpu_owned_count; i++) {
        if (g_rt_gpu_owned[i] == pointer) return 1;
    }
    return 0;
}

static RT_UNUSED void rt_gpu_mark_owned(void* pointer) {
    if (pointer == NULL) return;
    if (rt_gpu_is_owned(pointer)) return;
    if (g_rt_gpu_owned_count >= RT_GPU_MAX_OWNED) rt_runtime_fail("too many protected GPU buffers");
    g_rt_gpu_owned[g_rt_gpu_owned_count++] = pointer;
}

static RT_UNUSED void rt_gpu_refuse_owned(void* pointer) {
    if (rt_gpu_is_owned(pointer)) rt_runtime_fail("gpu.free cannot free a mounted tensor or workspace");
}

static RT_UNUSED int64_t rt_gpu_dim(rt_string name, int64_t axis) {
    if (!g_rt_txm.loaded) rt_runtime_fail("gpu.mount has not been called");
    rt_txm_tensor* tensor = rt_txm_find(name);
    if (tensor == NULL) rt_runtime_fail("unknown model tensor");
    if (axis < 0 || (uint32_t)axis >= tensor->rank) rt_runtime_fail("model tensor axis is out of range");
    return (int64_t)tensor->dims[axis];
}

static RT_UNUSED double rt_gpu_scalar(rt_string name) {
    if (!g_rt_txm.loaded) rt_runtime_fail("gpu.mount has not been called");
    rt_txm_tensor* tensor = rt_txm_find(name);
    if (tensor == NULL || tensor->dtype != RT_TXM_DTYPE_FLOAT || tensor->count != 1) {
        rt_runtime_fail("gpu.scalar expects a one-element float tensor");
    }
    return tensor->scalar;
}

static RT_UNUSED rt_slice_float rt_gpu_host_floats(rt_string name) {
    if (!g_rt_txm.loaded) rt_runtime_fail("gpu.mount has not been called");
    rt_txm_tensor* tensor = rt_txm_find(name);
    if (tensor == NULL || tensor->dtype != RT_TXM_DTYPE_FLOAT) rt_runtime_fail("gpu.hostFloats expects a float tensor");
    return RT_LIT(rt_slice_float, (double*)tensor->host, (size_t)tensor->count);
}

static RT_UNUSED rt_slice_int rt_gpu_host_ints(rt_string name) {
    if (!g_rt_txm.loaded) rt_runtime_fail("gpu.mount has not been called");
    rt_txm_tensor* tensor = rt_txm_find(name);
    if (tensor == NULL || tensor->dtype != RT_TXM_DTYPE_INT) rt_runtime_fail("gpu.hostInts expects an int tensor");
    return RT_LIT(rt_slice_int, (int64_t*)tensor->host, (size_t)tensor->count);
}

#ifdef TRUX_RUNTIME_CUDA
typedef struct { int64_t* data; size_t len; } rt_gpu_buffer_int;
typedef struct { double* data; size_t len; } rt_gpu_buffer_float;

static RT_UNUSED void rt_cuda_check(cudaError_t err, const char* operation) {
    if (err != cudaSuccess) {
        fprintf(stderr, "trux runtime error: CUDA %s failed: %s\n", operation, cudaGetErrorString(err));
        exit(1);
    }
}

typedef struct {
    char name[RT_TXM_MAX_NAME];
    uint32_t name_len;
    int is_float;
    uint64_t count;
    void* device;
    void* pinned;
} rt_gpu_ws_slot;

static __thread rt_gpu_ws_slot g_rt_gpu_ws[RT_GPU_MAX_WORKSPACES];
static __thread int g_rt_gpu_ws_ready;
static __thread cudaStream_t g_rt_gpu_stream;
static __thread int g_rt_gpu_stream_ready;
static __thread int g_rt_gpu_fd;
static __thread int g_rt_gpu_fd_ready;
static __thread int g_rt_gpu_job_busy;
static rt_gpu_ws_decl g_rt_gpu_ws_decl[RT_GPU_MAX_WORKSPACES];
static int g_rt_gpu_ws_decl_count;

static RT_UNUSED rt_gpu_ws_slot* rt_gpu_ws_by_device(void* pointer) {
    if (!g_rt_gpu_ws_ready || pointer == NULL) return NULL;
    for (int i = 0; i < g_rt_gpu_ws_decl_count; i++) {
        if (g_rt_gpu_ws[i].device == pointer) return &g_rt_gpu_ws[i];
    }
    return NULL;
}

static RT_UNUSED cudaStream_t rt_gpu_launch_stream(void) {
    if (g_rt_gpu_stream_ready) return g_rt_gpu_stream;
    return 0;
}

static RT_UNUSED rt_gpu_buffer_int rt_gpu_alloc_int(int64_t count) {
    size_t len = rt_checked_count(count, "gpu buffer length");
    int64_t* data = NULL;
    rt_cuda_check(cudaMalloc((void**)&data, rt_checked_bytes(len, sizeof(int64_t))), "alloc");
    return RT_LIT(rt_gpu_buffer_int, data, len);
}

static RT_UNUSED rt_gpu_buffer_float rt_gpu_alloc_float(int64_t count) {
    size_t len = rt_checked_count(count, "gpu buffer length");
    double* data = NULL;
    rt_cuda_check(cudaMalloc((void**)&data, rt_checked_bytes(len, sizeof(double))), "alloc");
    return RT_LIT(rt_gpu_buffer_float, data, len);
}

static RT_UNUSED void rt_gpu_copy_to_device_int(rt_slice_int host, rt_gpu_buffer_int device) {
    if (host.len > device.len) rt_runtime_fail("gpu copyToDevice host slice is larger than device buffer");
    rt_gpu_ws_slot* slot = rt_gpu_ws_by_device(device.data);
    if (slot != NULL && g_rt_gpu_stream_ready) {
        memcpy(slot->pinned, host.data, host.len * sizeof(int64_t));
        rt_cuda_check(cudaMemcpyAsync(device.data, slot->pinned, host.len * sizeof(int64_t), cudaMemcpyHostToDevice, g_rt_gpu_stream), "copyToDevice");
        return;
    }
    rt_cuda_check(cudaMemcpy(device.data, host.data, host.len * sizeof(int64_t), cudaMemcpyHostToDevice), "copyToDevice");
}

static RT_UNUSED void rt_gpu_copy_to_device_float(rt_slice_float host, rt_gpu_buffer_float device) {
    if (host.len > device.len) rt_runtime_fail("gpu copyToDevice host slice is larger than device buffer");
    rt_gpu_ws_slot* slot = rt_gpu_ws_by_device(device.data);
    if (slot != NULL && g_rt_gpu_stream_ready) {
        memcpy(slot->pinned, host.data, host.len * sizeof(double));
        rt_cuda_check(cudaMemcpyAsync(device.data, slot->pinned, host.len * sizeof(double), cudaMemcpyHostToDevice, g_rt_gpu_stream), "copyToDevice");
        return;
    }
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
    if (rt_gpu_ws_by_device(buffer.data) != NULL) rt_runtime_fail("gpu.free cannot free a mounted tensor or workspace");
    rt_gpu_refuse_owned(buffer.data);
    rt_cuda_check(cudaFree(buffer.data), "free");
}

static RT_UNUSED void rt_gpu_free_float(rt_gpu_buffer_float buffer) {
    if (rt_gpu_ws_by_device(buffer.data) != NULL) rt_runtime_fail("gpu.free cannot free a mounted tensor or workspace");
    rt_gpu_refuse_owned(buffer.data);
    rt_cuda_check(cudaFree(buffer.data), "free");
}

static RT_UNUSED void rt_gpu_sync(void) {
    rt_cuda_check(cudaDeviceSynchronize(), "sync");
}

static RT_UNUSED void rt_gpu_upload_all(void) {
    rt_cuda_check(cudaSetDevice(0), "setDevice");
    for (uint32_t i = 0; i < g_rt_txm.count; i++) {
        rt_txm_tensor* tensor = &g_rt_txm.tensors[i];
        size_t elem = tensor->dtype == RT_TXM_DTYPE_FLOAT ? sizeof(double) : sizeof(int64_t);
        size_t bytes = (size_t)tensor->count * elem;
        void* device = NULL;
        rt_cuda_check(cudaMalloc(&device, bytes), "alloc");
        rt_cuda_check(cudaMemcpy(device, tensor->host, bytes, cudaMemcpyHostToDevice), "copyToDevice");
        tensor->device = device;
    }
    g_rt_txm.loaded = 1;
}

static RT_UNUSED void rt_gpu_mount(rt_string path) {
    if (g_rt_txm.loaded) rt_runtime_fail("gpu.mount can only be called once");
    rt_arena arena;
    rt_arena_init(&arena);
    trux_tuple_string_error file = rt_read_file(&arena, path);
    if (file.f1 != NULL) {
        rt_arena_deinit(&arena);
        rt_runtime_fail("failed to read model file");
    }
    const char* err = rt_txm_parse(file.f0.data, file.f0.len);
    rt_arena_deinit(&arena);
    if (err != NULL) rt_runtime_fail(err);
    rt_gpu_upload_all();
}

static RT_UNUSED rt_gpu_buffer_float rt_gpu_floats(rt_string name) {
    if (!g_rt_txm.loaded) rt_runtime_fail("gpu.mount has not been called");
    rt_txm_tensor* tensor = rt_txm_find(name);
    if (tensor == NULL || tensor->dtype != RT_TXM_DTYPE_FLOAT || tensor->device == NULL) {
        rt_runtime_fail("gpu.floats expects a mounted float tensor");
    }
    return RT_LIT(rt_gpu_buffer_float, (double*)tensor->device, (size_t)tensor->count);
}

static RT_UNUSED rt_gpu_buffer_int rt_gpu_ints(rt_string name) {
    if (!g_rt_txm.loaded) rt_runtime_fail("gpu.mount has not been called");
    rt_txm_tensor* tensor = rt_txm_find(name);
    if (tensor == NULL || tensor->dtype != RT_TXM_DTYPE_INT || tensor->device == NULL) {
        rt_runtime_fail("gpu.ints expects a mounted int tensor");
    }
    return RT_LIT(rt_gpu_buffer_int, (int64_t*)tensor->device, (size_t)tensor->count);
}

static RT_UNUSED void rt_gpu_alloc_workspace_slot(int index) {
    rt_gpu_ws_decl* decl = &g_rt_gpu_ws_decl[index];
    size_t elem = decl->is_float ? sizeof(double) : sizeof(int64_t);
    size_t bytes = (size_t)decl->count * elem;
    void* device = NULL;
    void* pinned = NULL;
    rt_cuda_check(cudaMalloc(&device, bytes), "workspace");
    rt_cuda_check(cudaHostAlloc(&pinned, bytes, cudaHostAllocPortable), "workspace");
    memset(pinned, 0, bytes);
    g_rt_gpu_ws[index].name_len = decl->name_len;
    memcpy(g_rt_gpu_ws[index].name, decl->name, decl->name_len);
    g_rt_gpu_ws[index].is_float = decl->is_float;
    g_rt_gpu_ws[index].count = decl->count;
    g_rt_gpu_ws[index].device = device;
    g_rt_gpu_ws[index].pinned = pinned;
}

static RT_UNUSED void rt_gpu_workspace_declare(rt_string name, int64_t count, int is_float) {
    if (g_rt_gpu_workers_started) rt_runtime_fail("gpu.workspace must run before http.listen");
    if (name.len == 0 || name.len > RT_TXM_MAX_NAME) rt_runtime_fail("gpu.workspace name has a bad length");
    if (count <= 0) rt_runtime_fail("gpu.workspace length must be positive");
    for (int i = 0; i < g_rt_gpu_ws_decl_count; i++) {
        if (g_rt_gpu_ws_decl[i].name_len == name.len && memcmp(g_rt_gpu_ws_decl[i].name, name.data, name.len) == 0) {
            rt_runtime_fail("gpu.workspace name is already declared");
        }
    }
    if (g_rt_gpu_ws_decl_count >= RT_GPU_MAX_WORKSPACES) rt_runtime_fail("too many GPU workspaces");
    int index = g_rt_gpu_ws_decl_count++;
    g_rt_gpu_ws_decl[index].name_len = (uint32_t)name.len;
    memcpy(g_rt_gpu_ws_decl[index].name, name.data, name.len);
    g_rt_gpu_ws_decl[index].is_float = is_float;
    g_rt_gpu_ws_decl[index].count = (uint64_t)count;
    rt_cuda_check(cudaSetDevice(0), "setDevice");
    rt_gpu_alloc_workspace_slot(index);
    g_rt_gpu_ws_ready = 1;
}

static RT_UNUSED void rt_gpu_workspace_float(rt_string name, int64_t count) {
    rt_gpu_workspace_declare(name, count, 1);
}

static RT_UNUSED void rt_gpu_workspace_int(rt_string name, int64_t count) {
    rt_gpu_workspace_declare(name, count, 0);
}

static RT_UNUSED rt_gpu_ws_slot* rt_gpu_ws_by_name(rt_string name, int is_float) {
    if (!g_rt_gpu_ws_ready) rt_runtime_fail("gpu.workspace has not been called on this thread");
    for (int i = 0; i < g_rt_gpu_ws_decl_count; i++) {
        rt_gpu_ws_slot* slot = &g_rt_gpu_ws[i];
        if (slot->name_len == name.len && name.len > 0 && memcmp(slot->name, name.data, name.len) == 0) {
            if (slot->is_float != is_float) rt_runtime_fail("GPU workspace has a different element type");
            return slot;
        }
    }
    rt_runtime_fail("unknown GPU workspace");
    return NULL;
}

static RT_UNUSED rt_gpu_buffer_float rt_gpu_borrow_float(rt_string name) {
    rt_gpu_ws_slot* slot = rt_gpu_ws_by_name(name, 1);
    return RT_LIT(rt_gpu_buffer_float, (double*)slot->device, (size_t)slot->count);
}

static RT_UNUSED rt_gpu_buffer_int rt_gpu_borrow_int(rt_string name) {
    rt_gpu_ws_slot* slot = rt_gpu_ws_by_name(name, 0);
    return RT_LIT(rt_gpu_buffer_int, (int64_t*)slot->device, (size_t)slot->count);
}

static RT_UNUSED rt_slice_float rt_gpu_ready(rt_string name) {
    rt_gpu_ws_slot* slot = rt_gpu_ws_by_name(name, 1);
    return RT_LIT(rt_slice_float, (double*)slot->pinned, (size_t)slot->count);
}

#if defined(__linux__)
#include <sys/eventfd.h>
#if defined(__cplusplus)
extern "C"
#endif
void trux_rt_gpu_host_signal(void* user) {
    int fd = (int)(intptr_t)user;
    if (fd < 0) return;
    uint64_t one = 1;
    ssize_t wrote = write(fd, &one, sizeof(one));
    (void)wrote;
}
#else
static void trux_rt_gpu_host_signal(void* user) { (void)user; }
#endif

static RT_UNUSED void rt_gpu_worker_init(void) {
#if defined(__linux__)
    rt_cuda_check(cudaSetDevice(0), "setDevice");
    if (!g_rt_gpu_stream_ready) {
        rt_cuda_check(cudaStreamCreateWithFlags(&g_rt_gpu_stream, cudaStreamNonBlocking), "stream");
        g_rt_gpu_stream_ready = 1;
    }
    if (!g_rt_gpu_fd_ready) {
        g_rt_gpu_fd = eventfd(0, EFD_NONBLOCK | EFD_CLOEXEC);
        if (g_rt_gpu_fd < 0) rt_runtime_fail("eventfd failed");
        g_rt_gpu_fd_ready = 1;
    }
    for (int i = 0; i < g_rt_gpu_ws_decl_count; i++) {
        if (g_rt_gpu_ws[i].device == NULL) rt_gpu_alloc_workspace_slot(i);
    }
    g_rt_gpu_ws_ready = g_rt_gpu_ws_decl_count > 0;
#else
    rt_runtime_fail("the HTTP GPU worker requires Linux");
#endif
}

static RT_UNUSED int rt_gpu_event_fd(void) { return g_rt_gpu_fd_ready ? g_rt_gpu_fd : -1; }

static RT_UNUSED int rt_gpu_busy(void) { return g_rt_gpu_job_busy; }

static RT_UNUSED void rt_gpu_clear_busy(void) { g_rt_gpu_job_busy = 0; }

static RT_UNUSED void rt_gpu_drain_eventfd(void) {
#if defined(__linux__)
    if (!g_rt_gpu_fd_ready) return;
    uint64_t value = 0;
    while (read(g_rt_gpu_fd, &value, sizeof(value)) > 0) {
    }
#else
    return;
#endif
}

static RT_UNUSED void rt_gpu_submit_signal(void) {
    if (!g_rt_gpu_stream_ready || rt_gpu_event_fd() < 0) rt_runtime_fail("gpu.finish requires an HTTP worker");
    if (g_rt_gpu_job_busy) rt_runtime_fail("a GPU request is already in flight on this worker");
    for (int i = 0; i < g_rt_gpu_ws_decl_count; i++) {
        rt_gpu_ws_slot* slot = &g_rt_gpu_ws[i];
        if (slot->device == NULL || slot->pinned == NULL) continue;
        size_t elem = slot->is_float ? sizeof(double) : sizeof(int64_t);
        size_t bytes = (size_t)slot->count * elem;
        rt_cuda_check(cudaMemcpyAsync(slot->pinned, slot->device, bytes, cudaMemcpyDeviceToHost, g_rt_gpu_stream), "ready");
    }
    int fd = rt_gpu_event_fd();
    if (fd < 0) rt_runtime_fail("gpu.finish requires an HTTP worker");
    rt_cuda_check(cudaLaunchHostFunc(g_rt_gpu_stream, trux_rt_gpu_host_signal, (void*)(intptr_t)fd), "finish");
    g_rt_gpu_job_busy = 1;
}

static RT_UNUSED int rt_gpu_query_job(void) {
    if (!g_rt_gpu_stream_ready) return 0;
    cudaError_t err = cudaStreamQuery(g_rt_gpu_stream);
    if (err == cudaErrorNotReady) return 0;
    if (err != cudaSuccess) {
        cudaGetLastError();
        return 1;
    }
    return 2;
}

static RT_UNUSED void rt_gpu_reset_stream(void) {
    if (g_rt_gpu_stream_ready) {
        cudaStreamDestroy(g_rt_gpu_stream);
        cudaGetLastError();
        rt_cuda_check(cudaStreamCreateWithFlags(&g_rt_gpu_stream, cudaStreamNonBlocking), "stream");
    }
    g_rt_gpu_job_busy = 0;
}
#else
static RT_UNUSED void rt_gpu_worker_init(void) {}
static RT_UNUSED int rt_gpu_event_fd(void) { return -1; }
static RT_UNUSED int rt_gpu_busy(void) { return 0; }
static RT_UNUSED void rt_gpu_clear_busy(void) {}
static RT_UNUSED void rt_gpu_drain_eventfd(void) {}
static RT_UNUSED void rt_gpu_submit_signal(void) { rt_runtime_fail("gpu.finish requires a CUDA build"); }
static RT_UNUSED int rt_gpu_query_job(void) { return 0; }
static RT_UNUSED void rt_gpu_reset_stream(void) {}
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

static RT_UNUSED bool rt_ppm_try_read_int_token(rt_string contents, size_t* index, int64_t* out_value) {
    rt_ppm_skip_space_and_comments(contents, index);
    if (*index >= contents.len || !isdigit((unsigned char)contents.data[*index])) {
        return false;
    }

    int64_t value = 0;
    while (*index < contents.len && isdigit((unsigned char)contents.data[*index])) {
        int digit = contents.data[*index] - '0';
        if (value > (INT64_MAX - digit) / 10) {
            return false;
        }
        value = value * 10 + digit;
        (*index)++;
    }
    *out_value = value;
    return true;
}

static RT_UNUSED int64_t rt_ppm_read_int_token(rt_string contents, size_t* index, const char* what) {
    int64_t value = 0;
    if (!rt_ppm_try_read_int_token(contents, index, &value)) {
        fprintf(stderr, "trux runtime error: expected ppm %s\n", what);
        exit(1);
    }
    return value;
}

static RT_UNUSED bool rt_ppm_try_read_header(rt_arena* arena, rt_string contents, rt_ppm_header* out_header, rt_error* out_err) {
    rt_check_string(contents);
    size_t index = 0;
    rt_ppm_skip_space_and_comments(contents, &index);
    if (index + 2 > contents.len || contents.data[index] != 'P' || contents.data[index + 1] != '3') {
        *out_err = rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"expected P3 ppm image", 21));
        return false;
    }
    index += 2;

    int64_t width = 0;
    int64_t height = 0;
    int64_t max_value = 0;
    if (!rt_ppm_try_read_int_token(contents, &index, &width) ||
        !rt_ppm_try_read_int_token(contents, &index, &height) ||
        !rt_ppm_try_read_int_token(contents, &index, &max_value)) {
        *out_err = rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"invalid ppm header", 18));
        return false;
    }
    if (width <= 0 || height <= 0) {
        *out_err = rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"ppm width and height must be positive", 37));
        return false;
    }
    if (max_value != 255) {
        *out_err = rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"only 8-bit P3 ppm images with max value 255 are supported", 57));
        return false;
    }
    rt_ppm_skip_space_and_comments(contents, &index);
    *out_header = RT_LIT(rt_ppm_header, width, height, max_value, index);
    return true;
}

static RT_UNUSED rt_ppm_header rt_ppm_read_header(rt_string contents) {
    rt_arena arena;
    rt_arena_init(&arena);
    rt_ppm_header header;
    rt_error err = NULL;
    if (!rt_ppm_try_read_header(&arena, contents, &header, &err)) {
        char* c_msg = rt_string_to_c_string(err->message, "ppm error");
        rt_arena_deinit(&arena);
        fprintf(stderr, "trux runtime error: %s\n", c_msg);
        exit(1);
    }
    rt_arena_deinit(&arena);
    return header;
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
    trux_tuple_string_error res = rt_read_file(&arena, path);
    if (res.f1 != NULL) {
        rt_runtime_fail("failed to read ppm image file");
    }
    rt_ppm_header header = rt_ppm_read_header(res.f0);
    int64_t width = header.width;
    rt_arena_deinit(&arena);
    return width;
}

static RT_UNUSED int64_t rt_image_height(rt_string path) {
    rt_arena arena;
    rt_arena_init(&arena);
    trux_tuple_string_error res = rt_read_file(&arena, path);
    if (res.f1 != NULL) {
        rt_runtime_fail("failed to read ppm image file");
    }
    rt_ppm_header header = rt_ppm_read_header(res.f0);
    int64_t height = header.height;
    rt_arena_deinit(&arena);
    return height;
}

static RT_UNUSED trux_tuple_slice_int_error rt_read_ppm(rt_arena* arena, rt_string path) {
    trux_tuple_string_error res = rt_read_file(arena, path);
    if (res.f1 != NULL) {
        return RT_LIT(trux_tuple_slice_int_error, rt_make_slice_int(arena, 0), res.f1);
    }
    rt_string contents = res.f0;
    rt_ppm_header header;
    rt_error header_err = NULL;
    if (!rt_ppm_try_read_header(arena, contents, &header, &header_err)) {
        return RT_LIT(trux_tuple_slice_int_error, rt_make_slice_int(arena, 0), header_err);
    }
    size_t count = rt_ppm_pixel_count(header);
    if (count > (size_t)INT64_MAX) {
        return RT_LIT(trux_tuple_slice_int_error, rt_make_slice_int(arena, 0), rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"ppm image too large", 19)));
    }

    rt_slice_int pixels = rt_make_slice_int(arena, (int64_t)count);
    size_t index = header.pixel_start;
    for (size_t i = 0; i < count; i++) {
        int64_t value = 0;
        if (!rt_ppm_try_read_int_token(contents, &index, &value)) {
            return RT_LIT(trux_tuple_slice_int_error, pixels, rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"expected ppm pixel value", 24)));
        }
        if (value < 0 || value > 255) {
            return RT_LIT(trux_tuple_slice_int_error, pixels, rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"ppm pixel value must be between 0 and 255", 41)));
        }
        pixels.data[i] = value;
    }

    rt_ppm_skip_space_and_comments(contents, &index);
    if (index < contents.len) {
        return RT_LIT(trux_tuple_slice_int_error, pixels, rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"unexpected trailing data in ppm image", 37)));
    }
    return RT_LIT(trux_tuple_slice_int_error, pixels, NULL);
}

static RT_UNUSED rt_error rt_write_ppm(rt_arena* arena, rt_string path, rt_slice_int pixels, int64_t width_value, int64_t height_value) {
    if (width_value <= 0 || height_value <= 0) {
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"ppm dimensions must be positive", 31));
    }
    size_t width = (size_t)width_value;
    size_t height = (size_t)height_value;
    if (width > SIZE_MAX / height || width * height > SIZE_MAX / 3) {
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"ppm image dimensions overflow", 29));
    }
    size_t expected = width * height * 3;
    rt_check_array_like(pixels.data, pixels.len, "ppm pixels");
    if (pixels.len != expected) {
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"ppm pixel count does not match dimensions", 41));
    }

    char* c_path = rt_string_to_c_string(path, "file path");
    FILE* file = fopen(c_path, "wb");
    if (file == NULL) {
        free(c_path);
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"failed to open ppm for writing", 30));
    }
    if (fprintf(file, "P3\n%zu %zu\n255\n", width, height) < 0) {
        fclose(file);
        free(c_path);
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"write failed", 12));
    }

    for (size_t i = 0; i < pixels.len; i++) {
        int64_t value = pixels.data[i];
        if (value < 0 || value > 255) {
            fclose(file);
            free(c_path);
            return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"ppm pixel value must be between 0 and 255", 41));
        }
        if (fprintf(file, "%" PRId64, value) < 0) {
            fclose(file);
            free(c_path);
            return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"write failed", 12));
        }
        if ((i + 1) % 12 == 0 || i + 1 == pixels.len) {
            rt_file_write_byte(file, '\n');
        } else {
            rt_file_write_byte(file, ' ');
        }
    }

    if (fclose(file) != 0) {
        free(c_path);
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"close ppm after writing failed", 30));
    }
    free(c_path);
    return NULL;
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

static RT_UNUSED trux_tuple_list_string_error rt_read_csv(rt_arena* arena, rt_string path, int64_t columns_value) {
    if (columns_value <= 0) {
        return RT_LIT(trux_tuple_list_string_error, rt_list_string_new(arena, 0), rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"csv columns must be positive", 28)));
    }
    size_t columns = (size_t)columns_value;
    trux_tuple_string_error file_res = rt_read_file(arena, path);
    if (file_res.f1 != NULL) {
        return RT_LIT(trux_tuple_list_string_error, rt_list_string_new(arena, 0), file_res.f1);
    }
    rt_string contents = file_res.f0;
    rt_list_string* cells = rt_list_string_new(arena, 0);
    if (contents.len == 0) {
        return RT_LIT(trux_tuple_list_string_error, cells, NULL);
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
                return RT_LIT(trux_tuple_list_string_error, cells, rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"unterminated csv quoted field", 29)));
            }
            if (index < contents.len && contents.data[index] != ',' && !rt_csv_is_newline(contents.data[index])) {
                return RT_LIT(trux_tuple_list_string_error, cells, rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"expected csv delimiter after quoted field", 41)));
            }
        } else {
            while (index < contents.len && contents.data[index] != ',' && !rt_csv_is_newline(contents.data[index])) {
                if (contents.data[index] == '"') {
                    return RT_LIT(trux_tuple_list_string_error, cells, rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"unexpected quote in csv field", 29)));
                }
                rt_byte_buffer_append_byte(&field, contents.data[index]);
                index++;
            }
        }

        rt_list_string_append(cells, rt_csv_finish_cell(arena, &field));
        row_columns++;

        if (index >= contents.len) {
            if (row_columns != columns) {
                return RT_LIT(trux_tuple_list_string_error, cells, rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"csv row column count mismatch", 29)));
            }
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
            if (row_columns != columns) {
                return RT_LIT(trux_tuple_list_string_error, cells, rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"csv row column count mismatch", 29)));
            }
            row_columns = 0;
            continue;
        }

        return RT_LIT(trux_tuple_list_string_error, cells, rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"invalid csv parser state", 24)));
    }

    return RT_LIT(trux_tuple_list_string_error, cells, NULL);
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

static RT_UNUSED rt_error rt_write_csv(rt_arena* arena, rt_string path, rt_list_string* cells, int64_t columns_value) {
    if (columns_value <= 0) {
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"csv columns must be positive", 28));
    }
    size_t columns = (size_t)columns_value;
    rt_check_array_like(cells->data, cells->len, "csv cells");
    if (cells->len % columns != 0) {
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"csv cell count is not divisible by columns", 42));
    }

    char* c_path = rt_string_to_c_string(path, "file path");
    FILE* file = fopen(c_path, "wb");
    if (file == NULL) {
        free(c_path);
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"failed to open csv for writing", 30));
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
        free(c_path);
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"failed to close csv after writing", 33));
    }
    free(c_path);
    return NULL;
}

static RT_UNUSED rt_string rt_string_from_c(rt_arena* arena, const char* s) {
    if (s == NULL) {
        return RT_LIT(rt_string, NULL, 0);
    }
    size_t len = strlen(s);
    if (len == 0) {
        return RT_LIT(rt_string, NULL, 0);
    }
    uint8_t* data = (uint8_t*)rt_arena_alloc(arena, len);
    memcpy(data, s, len);
    return RT_LIT(rt_string, data, len);
}

static int rt_compare_strings(const void* a, const void* b) {
    const rt_string* sa = (const rt_string*)a;
    const rt_string* sb = (const rt_string*)b;
    size_t min_len = sa->len < sb->len ? sa->len : sb->len;
    if (min_len > 0) {
        int cmp = memcmp(sa->data, sb->data, min_len);
        if (cmp != 0) {
            return cmp;
        }
    }
    if (sa->len < sb->len) return -1;
    if (sa->len > sb->len) return 1;
    return 0;
}

static RT_UNUSED rt_error rt_create_dir(rt_arena* arena, rt_string path) {
    char* c_path = rt_string_to_c_string(path, "dir path");
    int res = mkdir(c_path, 0777);
    free(c_path);
    if (res != 0) {
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"failed to create directory", 26));
    }
    return NULL;
}

static RT_UNUSED rt_error rt_remove(rt_arena* arena, rt_string path) {
    char* c_path = rt_string_to_c_string(path, "path");
    int res = remove(c_path);
    free(c_path);
    if (res != 0) {
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"failed to remove", 16));
    }
    return NULL;
}

static RT_UNUSED bool rt_exists(rt_string path) {
    char* c_path = rt_string_to_c_string(path, "path");
    struct stat st;
    int res = stat(c_path, &st);
    free(c_path);
    return res == 0;
}

static RT_UNUSED bool rt_is_file(rt_string path) {
    char* c_path = rt_string_to_c_string(path, "path");
    struct stat st;
    int res = stat(c_path, &st);
    free(c_path);
    return res == 0 && S_ISREG(st.st_mode);
}

static RT_UNUSED bool rt_is_dir(rt_string path) {
    char* c_path = rt_string_to_c_string(path, "path");
    struct stat st;
    int res = stat(c_path, &st);
    free(c_path);
    return res == 0 && S_ISDIR(st.st_mode);
}

static RT_UNUSED trux_tuple_list_string_error rt_read_dir(rt_arena* arena, rt_string path) {
    rt_list_string* entries = rt_list_string_new(arena, 0);
    char* c_path = rt_string_to_c_string(path, "dir path");
    DIR* dir = opendir(c_path);
    if (dir == NULL) {
        free(c_path);
        return RT_LIT(trux_tuple_list_string_error, entries, rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"failed to open directory", 25)));
    }
    struct dirent* entry;
    while ((entry = readdir(dir)) != NULL) {
        if (strcmp(entry->d_name, ".") == 0 || strcmp(entry->d_name, "..") == 0) {
            continue;
        }
        rt_string name = rt_string_from_c(arena, entry->d_name);
        rt_list_string_append(entries, name);
    }
    closedir(dir);
    free(c_path);
    if (entries->len > 1) {
        qsort(entries->data, entries->len, sizeof(rt_string), rt_compare_strings);
    }
    return RT_LIT(trux_tuple_list_string_error, entries, NULL);
}

#if defined(__CUDA_ARCH__)
static RT_UNUSED void rt_env_load(rt_arena* arena) {
    (void)arena;
}

static RT_UNUSED int64_t rt_env_len(void) {
    return 0;
}

static RT_UNUSED rt_string rt_env_key(int64_t index) {
    (void)index;
    return RT_LIT(rt_string, NULL, 0);
}

static RT_UNUSED rt_string rt_env_value(int64_t index) {
    (void)index;
    return RT_LIT(rt_string, NULL, 0);
}
#else
typedef struct {
    rt_string* keys;
    rt_string* values;
    int64_t len;
    int64_t cap;
} rt_env_table;

static rt_env_table trux_env_table;

static RT_UNUSED void rt_env_fail(const char* message) {
    fprintf(stderr, "trux runtime error: %s\n", message);
    exit(1);
}

static RT_UNUSED void rt_env_fail_line(int64_t line, const char* message) {
    fprintf(stderr, "trux runtime error: .env:%" PRId64 ": %s\n", line, message);
    exit(1);
}

static RT_UNUSED rt_string rt_env_copy(rt_arena* arena, const uint8_t* data, size_t len) {
    if (len == 0) {
        return RT_LIT(rt_string, NULL, 0);
    }
    uint8_t* copy = (uint8_t*)rt_arena_alloc(arena, len);
    memcpy(copy, data, len);
    return RT_LIT(rt_string, copy, len);
}

static RT_UNUSED void rt_env_grow(void) {
    if (trux_env_table.len < trux_env_table.cap) {
        return;
    }
    int64_t cap = trux_env_table.cap == 0 ? 8 : trux_env_table.cap * 2;
    if (cap < trux_env_table.cap) {
        rt_runtime_fail("allocation failed");
    }
    rt_string* keys = (rt_string*)realloc(trux_env_table.keys, (size_t)cap * sizeof(rt_string));
    if (keys == NULL) {
        rt_runtime_fail("allocation failed");
    }
    trux_env_table.keys = keys;
    rt_string* values = (rt_string*)realloc(trux_env_table.values, (size_t)cap * sizeof(rt_string));
    if (values == NULL) {
        rt_runtime_fail("allocation failed");
    }
    trux_env_table.values = values;
    trux_env_table.cap = cap;
}

static RT_UNUSED void rt_env_upsert(rt_arena* arena, const uint8_t* key, size_t key_len, const uint8_t* value, size_t value_len) {
    rt_string key_view = RT_LIT(rt_string, key_len == 0 ? NULL : key, key_len);
    for (int64_t i = 0; i < trux_env_table.len; i++) {
        if (rt_string_equal(trux_env_table.keys[i], key_view)) {
            trux_env_table.values[i] = rt_env_copy(arena, value, value_len);
            return;
        }
    }
    rt_env_grow();
    trux_env_table.keys[trux_env_table.len] = rt_env_copy(arena, key, key_len);
    trux_env_table.values[trux_env_table.len] = rt_env_copy(arena, value, value_len);
    trux_env_table.len++;
}

static RT_UNUSED void rt_env_add_entry(rt_arena* arena, const char* entry, size_t len) {
    size_t eq = 0;
    while (eq < len && (unsigned char)entry[eq] != '=') {
        eq++;
    }
    if (eq == len || eq == 0) {
        return;
    }
    rt_env_upsert(arena, (const uint8_t*)entry, eq, (const uint8_t*)entry + eq + 1, len - eq - 1);
}

static RT_UNUSED void rt_env_load_process(rt_arena* arena) {
    extern char** environ;
    if (environ == NULL) {
        return;
    }
    for (char** cursor = environ; *cursor != NULL; cursor++) {
        rt_env_add_entry(arena, *cursor, strlen(*cursor));
    }
}

static RT_UNUSED bool rt_env_space(unsigned char c) {
    return c == ' ' || c == '\t';
}

static RT_UNUSED bool rt_env_key_ok(const uint8_t* data, size_t len) {
    if (len == 0) {
        return false;
    }
    unsigned char first = data[0];
    if (!((first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z') || first == '_')) {
        return false;
    }
    for (size_t i = 1; i < len; i++) {
        unsigned char c = data[i];
        if (!((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_')) {
            return false;
        }
    }
    return true;
}

static RT_UNUSED void rt_env_parse_line(rt_arena* arena, const uint8_t* data, size_t len, int64_t line_no) {
    size_t start = 0;
    while (start < len && rt_env_space(data[start])) {
        start++;
    }
    if (start == len || data[start] == '#') {
        return;
    }
    size_t eq = start;
    while (eq < len && data[eq] != '=') {
        eq++;
    }
    if (eq == len) {
        rt_env_fail_line(line_no, "invalid line");
    }
    size_t key_end = eq;
    while (key_end > start && rt_env_space(data[key_end - 1])) {
        key_end--;
    }
    if (!rt_env_key_ok(data + start, key_end - start)) {
        rt_env_fail_line(line_no, "invalid key");
    }
    rt_env_upsert(arena, data + start, key_end - start, data + eq + 1, len - (eq + 1));
}

static RT_UNUSED void rt_env_parse(rt_arena* arena, const uint8_t* data, size_t len) {
    size_t i = 0;
    int64_t line_no = 1;
    while (i < len) {
        size_t start = i;
        while (i < len && data[i] != '\n') {
            i++;
        }
        size_t end = i;
        if (i < len && data[i] == '\n') {
            i++;
        }
        if (end > start && data[end - 1] == '\r') {
            end--;
        }
        rt_env_parse_line(arena, data + start, end - start, line_no);
        line_no++;
    }
}

static RT_UNUSED void rt_env_load_file(rt_arena* arena) {
    const char* path = ".env";
    struct stat st;
    if (stat(path, &st) != 0) {
        if (errno == ENOENT) {
            return;
        }
        rt_env_fail(".env: cannot open");
    }
    if (!S_ISREG(st.st_mode)) {
        rt_env_fail(".env: not a regular file");
    }
    FILE* file = fopen(path, "rb");
    if (file == NULL) {
        rt_env_fail(".env: cannot open");
    }
    if (fseek(file, 0, SEEK_END) != 0) {
        fclose(file);
        rt_env_fail(".env: cannot read");
    }
    long file_size = ftell(file);
    if (file_size < 0) {
        fclose(file);
        rt_env_fail(".env: cannot read");
    }
    if (fseek(file, 0, SEEK_SET) != 0) {
        fclose(file);
        rt_env_fail(".env: cannot read");
    }
    size_t size = (size_t)file_size;
    uint8_t* data = NULL;
    if (size > 0) {
        data = (uint8_t*)malloc(size);
        if (data == NULL) {
            fclose(file);
            rt_runtime_fail("allocation failed");
        }
        if (fread(data, 1, size, file) != size) {
            free(data);
            fclose(file);
            rt_env_fail(".env: cannot read");
        }
    }
    if (fclose(file) != 0) {
        free(data);
        rt_env_fail(".env: cannot read");
    }
    if (size > 0) {
        rt_env_parse(arena, data, size);
    }
    free(data);
}

static RT_UNUSED void rt_env_load(rt_arena* arena) {
    rt_env_load_process(arena);
    rt_env_load_file(arena);
}

static RT_UNUSED int64_t rt_env_len(void) {
    return trux_env_table.len;
}

static RT_UNUSED rt_string rt_env_key(int64_t index) {
    return trux_env_table.keys[index];
}

static RT_UNUSED rt_string rt_env_value(int64_t index) {
    return trux_env_table.values[index];
}
#endif

#endif
`

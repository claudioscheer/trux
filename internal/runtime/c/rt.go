package c

const Source = `#include <stdbool.h>
#include <stdint.h>
#include <stddef.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <inttypes.h>

#if defined(__GNUC__) || defined(__clang__)
#define RT_UNUSED __attribute__((unused))
#else
#define RT_UNUSED
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
    rt_arena* temp;
} rt_context;

typedef struct {
    size_t start;
    size_t end;
} rt_range;

static RT_UNUSED void rt_runtime_fail(const char* message) {
    fprintf(stderr, "trux runtime error: %s\n", message);
    exit(1);
}

static RT_UNUSED size_t rt_arena_align_up(size_t size) {
    size_t align = _Alignof(max_align_t);
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

    rt_arena_chunk* chunk = malloc(sizeof(rt_arena_chunk) + cap);
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
    rt_list_allocation* node = malloc(sizeof(rt_list_allocation));
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
    int64_t end = has_end ? end_value : (int64_t)len;
    if (start < 0 || end < 0 || start > end || (uint64_t)end > len) {
        fprintf(stderr, "trux runtime error: slice %" PRId64 ":%" PRId64 " out of bounds for length %zu\n", start, end, len);
        exit(1);
    }
    return (rt_range){(size_t)start, (size_t)end};
}

static RT_UNUSED void rt_print_int(int64_t value) {
    printf("%" PRId64, value);
}

static RT_UNUSED void rt_print_float(double value) {
    printf("%.15g", value);
}

static RT_UNUSED void rt_print_string(rt_string value) {
    fwrite(value.data, 1, value.len, stdout);
}

static RT_UNUSED void rt_print_bool(bool value) {
    printf("%s", value ? "true" : "false");
}

static RT_UNUSED void rt_print_newline(void) {
    putchar('\n');
}

static RT_UNUSED bool rt_string_equal(rt_string left, rt_string right) {
    if (left.len != right.len) {
        return false;
    }
    if (left.len == 0) {
        return true;
    }
    return memcmp(left.data, right.data, left.len) == 0;
}

static RT_UNUSED rt_string rt_string_concat(rt_arena* arena, rt_string left, rt_string right) {
    if (left.len == 0) {
        return right;
    }
    if (right.len == 0) {
        return left;
    }
    if (left.len > SIZE_MAX - right.len) {
        rt_runtime_fail("string length overflow");
    }

    size_t len = left.len + right.len;
    uint8_t* data = rt_arena_alloc(arena, len);
    memcpy(data, left.data, left.len);
    memcpy(data + left.len, right.data, right.len);
    return (rt_string){data, len};
}

static RT_UNUSED rt_string rt_string_index(rt_string value, int64_t index) {
    size_t checked = rt_check_index(value.len, index);
    return (rt_string){value.data + checked, 1};
}

static RT_UNUSED rt_string rt_string_slice(rt_string value, bool has_start, int64_t start, bool has_end, int64_t end) {
    rt_range range = rt_check_slice(value.len, has_start, start, has_end, end);
    const uint8_t* data = value.data == NULL ? NULL : value.data + range.start;
    return (rt_string){data, range.end - range.start};
}

static RT_UNUSED bool rt_string_contains(rt_string needle, rt_string haystack) {
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
static RT_UNUSED rt_array_##NAME rt_array_##NAME##_from_values(rt_arena* arena, const CTYPE* values, size_t len) { \
    CTYPE* data = rt_arena_alloc_count(arena, len, sizeof(CTYPE), false); \
    if (len > 0) { \
        memcpy(data, values, len * sizeof(CTYPE)); \
    } \
    return (rt_array_##NAME){data, len}; \
} \
static RT_UNUSED rt_slice_##NAME rt_make_slice_##NAME(rt_arena* arena, int64_t count) { \
    size_t len = rt_checked_count(count, "slice length"); \
    CTYPE* data = rt_arena_alloc_count(arena, len, sizeof(CTYPE), true); \
    return (rt_slice_##NAME){data, len}; \
} \
static RT_UNUSED CTYPE rt_array_##NAME##_get(rt_array_##NAME value, int64_t index) { \
    return value.data[rt_check_index(value.len, index)]; \
} \
static RT_UNUSED CTYPE rt_slice_##NAME##_get(rt_slice_##NAME value, int64_t index) { \
    return value.data[rt_check_index(value.len, index)]; \
} \
static RT_UNUSED CTYPE rt_list_##NAME##_get(rt_list_##NAME* value, int64_t index) { \
    return value->data[rt_check_index(value->len, index)]; \
} \
static RT_UNUSED void rt_array_##NAME##_set(rt_array_##NAME value, int64_t index, CTYPE elem) { \
    value.data[rt_check_index(value.len, index)] = elem; \
} \
static RT_UNUSED void rt_slice_##NAME##_set(rt_slice_##NAME value, int64_t index, CTYPE elem) { \
    value.data[rt_check_index(value.len, index)] = elem; \
} \
static RT_UNUSED void rt_list_##NAME##_set(rt_list_##NAME* value, int64_t index, CTYPE elem) { \
    value->data[rt_check_index(value->len, index)] = elem; \
} \
static RT_UNUSED rt_slice_##NAME rt_array_##NAME##_slice(rt_array_##NAME value, bool has_start, int64_t start, bool has_end, int64_t end) { \
    rt_range range = rt_check_slice(value.len, has_start, start, has_end, end); \
    CTYPE* data = value.data == NULL ? NULL : value.data + range.start; \
    return (rt_slice_##NAME){data, range.end - range.start}; \
} \
static RT_UNUSED rt_slice_##NAME rt_slice_##NAME##_slice(rt_slice_##NAME value, bool has_start, int64_t start, bool has_end, int64_t end) { \
    rt_range range = rt_check_slice(value.len, has_start, start, has_end, end); \
    CTYPE* data = value.data == NULL ? NULL : value.data + range.start; \
    return (rt_slice_##NAME){data, range.end - range.start}; \
} \
static RT_UNUSED rt_slice_##NAME rt_list_##NAME##_slice(rt_list_##NAME* value, bool has_start, int64_t start, bool has_end, int64_t end) { \
    rt_range range = rt_check_slice(value->len, has_start, start, has_end, end); \
    CTYPE* data = value->data == NULL ? NULL : value->data + range.start; \
    return (rt_slice_##NAME){data, range.end - range.start}; \
} \
static RT_UNUSED void rt_list_##NAME##_free(void* ptr) { \
    rt_list_##NAME* list = ptr; \
    free(list->data); \
    free(list); \
} \
static RT_UNUSED rt_list_##NAME* rt_list_##NAME##_new(rt_arena* arena, size_t cap) { \
    rt_list_##NAME* list = malloc(sizeof(rt_list_##NAME)); \
    if (list == NULL) { \
        rt_runtime_fail("allocation failed"); \
    } \
    list->data = NULL; \
    list->len = 0; \
    list->cap = cap; \
    if (cap > 0) { \
        list->data = malloc(rt_checked_bytes(cap, sizeof(CTYPE))); \
        if (list->data == NULL) { \
            free(list); \
            rt_runtime_fail("allocation failed"); \
        } \
    } \
    rt_arena_register_list(arena, list, rt_list_##NAME##_free); \
    return list; \
} \
static RT_UNUSED rt_list_##NAME* rt_list_##NAME##_from_values(rt_arena* arena, const CTYPE* values, size_t len) { \
    rt_list_##NAME* list = rt_list_##NAME##_new(arena, len); \
    if (len > 0) { \
        memcpy(list->data, values, len * sizeof(CTYPE)); \
    } \
    list->len = len; \
    return list; \
} \
static RT_UNUSED void rt_list_##NAME##_append(rt_list_##NAME* list, CTYPE elem) { \
    if (list->len == list->cap) { \
        size_t new_cap = list->cap == 0 ? 4 : list->cap * 2; \
        if (new_cap < list->cap || new_cap > SIZE_MAX / sizeof(CTYPE)) { \
            rt_runtime_fail("list capacity overflow"); \
        } \
        CTYPE* data = realloc(list->data, new_cap * sizeof(CTYPE)); \
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
`

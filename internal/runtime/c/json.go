package c

import _ "embed"

const JSONHeaderName = "trux_json_runtime.h"

//go:embed yyjson/yyjson.h
var YYJSONHeaderSource string

//go:embed yyjson/yyjson.c
var YYJSONCSource string

const JSONHeaderSource = `
#ifndef TRUX_JSON_RUNTIME_H
#define TRUX_JSON_RUNTIME_H

#include "yyjson.h"

typedef struct yyjson_doc trux_json_doc;
typedef struct yyjson_val trux_json_val;
typedef struct yyjson_mut_doc trux_json_mut_doc;
typedef struct yyjson_mut_val trux_json_mut_val;

#endif /* TRUX_JSON_RUNTIME_H */
`

const JSONImplSource = `
/* --- JSON (yyjson) Runtime Implementation --- */

static void* rt_yyjson_malloc(void* ctx, size_t size) {
    return rt_arena_alloc((rt_arena*)ctx, size);
}

static void* rt_yyjson_realloc(void* ctx, void* ptr, size_t old_size, size_t size) {
    void* new_ptr = rt_arena_alloc((rt_arena*)ctx, size);
    if (ptr && old_size > 0) {
        memcpy(new_ptr, ptr, old_size < size ? old_size : size);
    }
    return new_ptr;
}

static void rt_yyjson_free(void* ctx, void* ptr) {
    (void)ctx;
    (void)ptr;
}

static RT_UNUSED trux_tuple_ptr_struct_13_trux_json_doc_error rt_json_parse(rt_arena* arena, rt_string input) {
    if (input.len == 0 || input.data == NULL) {
        rt_error err = rt_error_new(arena, rt_string_from_c(arena, "unexpected end of JSON input"));
        return RT_LIT(trux_tuple_ptr_struct_13_trux_json_doc_error, NULL, err);
    }
    yyjson_read_err read_err;
    yyjson_alc alc = { rt_yyjson_malloc, rt_yyjson_realloc, rt_yyjson_free, arena };
    yyjson_doc* doc = yyjson_read_opts((char*)input.data, input.len, YYJSON_READ_NOFLAG, &alc, &read_err);
    if (!doc) {
        char buf[256];
        snprintf(buf, sizeof(buf), "json syntax error at position %zu: %s", (size_t)read_err.pos, read_err.msg ? read_err.msg : "unknown error");
        rt_error err = rt_error_new(arena, rt_string_from_c(arena, buf));
        return RT_LIT(trux_tuple_ptr_struct_13_trux_json_doc_error, NULL, err);
    }
    return RT_LIT(trux_tuple_ptr_struct_13_trux_json_doc_error, (trux_json_doc*)doc, NULL);
}

static RT_UNUSED trux_json_val* rt_json_root(rt_arena* arena, trux_json_doc* doc) {
    (void)arena;
    if (!doc) return NULL;
    return (trux_json_val*)yyjson_doc_get_root((yyjson_doc*)doc);
}

static RT_UNUSED trux_json_val* rt_json_get(rt_arena* arena, trux_json_val* val, rt_string key) {
    (void)arena;
    if (!val) return NULL;
    return (trux_json_val*)yyjson_obj_getn((yyjson_val*)val, (const char*)key.data, key.len);
}

static RT_UNUSED trux_json_val* rt_json_at(rt_arena* arena, trux_json_val* val, int64_t index) {
    (void)arena;
    if (!val || index < 0) return NULL;
    return (trux_json_val*)yyjson_arr_get((yyjson_val*)val, (size_t)index);
}

static RT_UNUSED int64_t rt_json_len(rt_arena* arena, trux_json_val* val) {
    (void)arena;
    if (!val) return 0;
    if (yyjson_is_arr((yyjson_val*)val)) return (int64_t)yyjson_arr_size((yyjson_val*)val);
    if (yyjson_is_obj((yyjson_val*)val)) return (int64_t)yyjson_obj_size((yyjson_val*)val);
    if (yyjson_is_str((yyjson_val*)val)) return (int64_t)yyjson_get_len((yyjson_val*)val);
    return 0;
}

static RT_UNUSED bool rt_json_is_object(rt_arena* arena, trux_json_val* val) {
    (void)arena;
    return val && yyjson_is_obj((yyjson_val*)val);
}

static RT_UNUSED bool rt_json_is_array(rt_arena* arena, trux_json_val* val) {
    (void)arena;
    return val && yyjson_is_arr((yyjson_val*)val);
}

static RT_UNUSED bool rt_json_is_string(rt_arena* arena, trux_json_val* val) {
    (void)arena;
    return val && yyjson_is_str((yyjson_val*)val);
}

static RT_UNUSED bool rt_json_is_int(rt_arena* arena, trux_json_val* val) {
    (void)arena;
    return val && (yyjson_is_int((yyjson_val*)val) || yyjson_is_uint((yyjson_val*)val));
}

static RT_UNUSED bool rt_json_is_float(rt_arena* arena, trux_json_val* val) {
    (void)arena;
    return val && yyjson_is_real((yyjson_val*)val);
}

static RT_UNUSED bool rt_json_is_bool(rt_arena* arena, trux_json_val* val) {
    (void)arena;
    return val && yyjson_is_bool((yyjson_val*)val);
}

static RT_UNUSED bool rt_json_is_null(rt_arena* arena, trux_json_val* val) {
    (void)arena;
    return !val || yyjson_is_null((yyjson_val*)val);
}

static RT_UNUSED rt_string rt_json_get_string(rt_arena* arena, trux_json_val* val) {
    if (!val) return RT_LIT(rt_string, NULL, 0);
    const char* s = yyjson_get_str((yyjson_val*)val);
    if (!s) return RT_LIT(rt_string, NULL, 0);
    size_t len = yyjson_get_len((yyjson_val*)val);
    uint8_t* copy = (uint8_t*)rt_arena_alloc(arena, len);
    memcpy(copy, s, len);
    return RT_LIT(rt_string, copy, len);
}

static RT_UNUSED int64_t rt_json_get_int(rt_arena* arena, trux_json_val* val) {
    (void)arena;
    if (!val) return 0;
    if (yyjson_is_int((yyjson_val*)val)) return yyjson_get_sint((yyjson_val*)val);
    if (yyjson_is_uint((yyjson_val*)val)) return (int64_t)yyjson_get_uint((yyjson_val*)val);
    if (yyjson_is_real((yyjson_val*)val)) return (int64_t)yyjson_get_real((yyjson_val*)val);
    if (yyjson_is_bool((yyjson_val*)val)) return yyjson_get_bool((yyjson_val*)val) ? 1 : 0;
    return 0;
}

static RT_UNUSED double rt_json_get_float(rt_arena* arena, trux_json_val* val) {
    (void)arena;
    if (!val) return 0.0;
    if (yyjson_is_num((yyjson_val*)val)) return yyjson_get_num((yyjson_val*)val);
    return 0.0;
}

static RT_UNUSED bool rt_json_get_bool(rt_arena* arena, trux_json_val* val) {
    (void)arena;
    if (!val) return false;
    return yyjson_get_bool((yyjson_val*)val);
}

static RT_UNUSED rt_string rt_json_field_string(rt_arena* arena, trux_json_val* val, rt_string key) {
    return rt_json_get_string(arena, rt_json_get(arena, val, key));
}

static RT_UNUSED int64_t rt_json_field_int(rt_arena* arena, trux_json_val* val, rt_string key) {
    return rt_json_get_int(arena, rt_json_get(arena, val, key));
}

static RT_UNUSED double rt_json_field_float(rt_arena* arena, trux_json_val* val, rt_string key) {
    return rt_json_get_float(arena, rt_json_get(arena, val, key));
}

static RT_UNUSED bool rt_json_field_bool(rt_arena* arena, trux_json_val* val, rt_string key) {
    return rt_json_get_bool(arena, rt_json_get(arena, val, key));
}

static RT_UNUSED trux_tuple_map_string_string_error rt_json_decode_map(rt_arena* arena, rt_string input) {
    if (input.len == 0 || input.data == NULL) {
        rt_error err = rt_error_new(arena, rt_string_from_c(arena, "unexpected end of JSON input"));
        return RT_LIT(trux_tuple_map_string_string_error, NULL, err);
    }
    yyjson_read_err read_err;
    yyjson_alc alc = { rt_yyjson_malloc, rt_yyjson_realloc, rt_yyjson_free, arena };
    yyjson_doc* doc = yyjson_read_opts((char*)input.data, input.len, YYJSON_READ_NOFLAG, &alc, &read_err);
    if (!doc) {
        char buf[256];
        snprintf(buf, sizeof(buf), "json syntax error at position %zu: %s", (size_t)read_err.pos, read_err.msg ? read_err.msg : "unknown error");
        rt_error err = rt_error_new(arena, rt_string_from_c(arena, buf));
        return RT_LIT(trux_tuple_map_string_string_error, NULL, err);
    }
    yyjson_val* root = yyjson_doc_get_root(doc);
    if (!yyjson_is_obj(root)) {
        rt_error err = rt_error_new(arena, rt_string_from_c(arena, "expected top-level JSON object"));
        return RT_LIT(trux_tuple_map_string_string_error, NULL, err);
    }
    size_t count = yyjson_obj_size(root);
    rt_map_string_string* m = rt_map_string_string_new(arena, (int64_t)count);
    size_t idx, max;
    yyjson_val *key, *val;
    yyjson_obj_foreach(root, idx, max, key, val) {
        const char* k_str = yyjson_get_str(key);
        size_t k_len = yyjson_get_len(key);
        uint8_t* k_buf = (uint8_t*)rt_arena_alloc(arena, k_len);
        memcpy(k_buf, k_str, k_len);
        rt_string k = RT_LIT(rt_string, k_buf, k_len);

        rt_string v;
        if (yyjson_is_str(val)) {
            const char* v_str = yyjson_get_str(val);
            size_t v_len = yyjson_get_len(val);
            uint8_t* v_buf = (uint8_t*)rt_arena_alloc(arena, v_len);
            memcpy(v_buf, v_str, v_len);
            v = RT_LIT(rt_string, v_buf, v_len);
        } else {
            size_t v_len = 0;
            char* written = yyjson_val_write_opts(val, YYJSON_WRITE_NOFLAG, &alc, &v_len, NULL);
            if (written) {
                v = RT_LIT(rt_string, (const uint8_t*)written, v_len);
            } else {
                v = RT_LIT(rt_string, NULL, 0);
            }
        }
        rt_map_string_string_set(m, k, v);
    }
    return RT_LIT(trux_tuple_map_string_string_error, m, NULL);
}

static RT_UNUSED rt_string rt_json_encode_map(rt_arena* arena, rt_map_string_string* m) {
    if (!m || m->len == 0) {
        return rt_string_from_c(arena, "{}");
    }
    yyjson_alc alc = { rt_yyjson_malloc, rt_yyjson_realloc, rt_yyjson_free, arena };
    yyjson_mut_doc* doc = yyjson_mut_doc_new(&alc);
    yyjson_mut_val* root = yyjson_mut_obj(doc);
    yyjson_mut_doc_set_root(doc, root);
    for (size_t i = 0; i < m->entries_cap; i++) {
        if (!m->entries[i].deleted && m->entries[i].key.data != NULL) {
            rt_string k = m->entries[i].key;
            rt_string v = m->entries[i].value;
            yyjson_mut_val* k_val = yyjson_mut_strn(doc, (const char*)k.data, k.len);
            yyjson_mut_val* v_val = yyjson_mut_strn(doc, (const char*)v.data, v.len);
            yyjson_mut_obj_add(root, k_val, v_val);
        }
    }
    size_t out_len = 0;
    char* json_str = yyjson_mut_write_opts(doc, YYJSON_WRITE_NOFLAG, &alc, &out_len, NULL);
    if (!json_str) {
        return rt_string_from_c(arena, "{}");
    }
    return RT_LIT(rt_string, (const uint8_t*)json_str, out_len);
}

static RT_UNUSED rt_string rt_json_stringify(rt_arena* arena, trux_json_doc* doc) {
    if (!doc) return RT_LIT(rt_string, NULL, 0);
    yyjson_alc alc = { rt_yyjson_malloc, rt_yyjson_realloc, rt_yyjson_free, arena };
    size_t out_len = 0;
    char* json_str = yyjson_write_opts((yyjson_doc*)doc, YYJSON_WRITE_NOFLAG, &alc, &out_len, NULL);
    if (!json_str) return RT_LIT(rt_string, NULL, 0);
    return RT_LIT(rt_string, (const uint8_t*)json_str, out_len);
}

static RT_UNUSED bool rt_json_valid(rt_arena* arena, rt_string input) {
    (void)arena;
    if (input.len == 0 || input.data == NULL) return false;
    yyjson_read_err err;
    yyjson_doc* doc = yyjson_read_opts((char*)input.data, input.len, YYJSON_READ_NOFLAG, NULL, &err);
    if (doc) {
        yyjson_doc_free(doc);
        return true;
    }
    return false;
}

static void rt_json_mut_doc_free(void* ptr) {
    yyjson_mut_doc_free((yyjson_mut_doc*)ptr);
}

static RT_UNUSED trux_json_mut_doc* rt_json_new_doc(rt_arena* arena) {
    /* The document owns its bytes on the heap. A string added from a called
       function, or inside a loop that rewinds this frame, stays intact until
       the arena that created the document is released. */
    yyjson_mut_doc* doc = yyjson_mut_doc_new(NULL);
    if (doc == NULL) {
        rt_runtime_fail("allocation failed");
    }
    rt_list_allocation* node = (rt_list_allocation*)malloc(sizeof(*node));
    if (node == NULL) {
        yyjson_mut_doc_free(doc);
        rt_runtime_fail("allocation failed");
    }
    rt_arena_track_list(arena, node, doc, rt_json_mut_doc_free);
    return (trux_json_mut_doc*)doc;
}

static yyjson_mut_val* rt_json_copy_string(yyjson_mut_doc* doc, rt_string value) {
    const char* data = (const char*)value.data;
    if (value.len == 0 || data == NULL) {
        data = "";
    }
    return yyjson_mut_strncpy(doc, data, value.len);
}

static RT_UNUSED trux_json_mut_val* rt_json_new_object(rt_arena* arena, trux_json_mut_doc* doc) {
    (void)arena;
    return (trux_json_mut_val*)yyjson_mut_obj((yyjson_mut_doc*)doc);
}

static RT_UNUSED trux_json_mut_val* rt_json_new_array(rt_arena* arena, trux_json_mut_doc* doc) {
    (void)arena;
    return (trux_json_mut_val*)yyjson_mut_arr((yyjson_mut_doc*)doc);
}

static RT_UNUSED void rt_json_set_root(rt_arena* arena, trux_json_mut_doc* doc, trux_json_mut_val* root) {
    (void)arena;
    if (!doc) return;
    yyjson_mut_doc_set_root((yyjson_mut_doc*)doc, (yyjson_mut_val*)root);
}

static RT_UNUSED void rt_json_add_string(rt_arena* arena, trux_json_mut_doc* doc, trux_json_mut_val* obj, rt_string key, rt_string val) {
    (void)arena;
    if (!doc || !obj) return;
    yyjson_mut_val* k = rt_json_copy_string((yyjson_mut_doc*)doc, key);
    yyjson_mut_val* v = rt_json_copy_string((yyjson_mut_doc*)doc, val);
    yyjson_mut_obj_add((yyjson_mut_val*)obj, k, v);
}

static RT_UNUSED void rt_json_add_int(rt_arena* arena, trux_json_mut_doc* doc, trux_json_mut_val* obj, rt_string key, int64_t val) {
    (void)arena;
    if (!doc || !obj) return;
    yyjson_mut_val* k = rt_json_copy_string((yyjson_mut_doc*)doc, key);
    yyjson_mut_val* v = yyjson_mut_sint((yyjson_mut_doc*)doc, val);
    yyjson_mut_obj_add((yyjson_mut_val*)obj, k, v);
}

static RT_UNUSED void rt_json_add_float(rt_arena* arena, trux_json_mut_doc* doc, trux_json_mut_val* obj, rt_string key, double val) {
    (void)arena;
    if (!doc || !obj) return;
    yyjson_mut_val* k = rt_json_copy_string((yyjson_mut_doc*)doc, key);
    yyjson_mut_val* v = yyjson_mut_real((yyjson_mut_doc*)doc, val);
    yyjson_mut_obj_add((yyjson_mut_val*)obj, k, v);
}

static RT_UNUSED void rt_json_add_bool(rt_arena* arena, trux_json_mut_doc* doc, trux_json_mut_val* obj, rt_string key, bool val) {
    (void)arena;
    if (!doc || !obj) return;
    yyjson_mut_val* k = rt_json_copy_string((yyjson_mut_doc*)doc, key);
    yyjson_mut_val* v = yyjson_mut_bool((yyjson_mut_doc*)doc, val);
    yyjson_mut_obj_add((yyjson_mut_val*)obj, k, v);
}

static RT_UNUSED void rt_json_add_null(rt_arena* arena, trux_json_mut_doc* doc, trux_json_mut_val* obj, rt_string key) {
    (void)arena;
    if (!doc || !obj) return;
    yyjson_mut_val* k = rt_json_copy_string((yyjson_mut_doc*)doc, key);
    yyjson_mut_val* v = yyjson_mut_null((yyjson_mut_doc*)doc);
    yyjson_mut_obj_add((yyjson_mut_val*)obj, k, v);
}

static RT_UNUSED void rt_json_add_val(rt_arena* arena, trux_json_mut_doc* doc, trux_json_mut_val* obj, rt_string key, trux_json_mut_val* val) {
    (void)arena;
    if (!doc || !obj) return;
    yyjson_mut_val* k = rt_json_copy_string((yyjson_mut_doc*)doc, key);
    yyjson_mut_obj_add((yyjson_mut_val*)obj, k, (yyjson_mut_val*)val);
}

static RT_UNUSED void rt_json_append_val(rt_arena* arena, trux_json_mut_val* arr, trux_json_mut_val* val) {
    (void)arena;
    if (!arr || !val) return;
    yyjson_mut_arr_append((yyjson_mut_val*)arr, (yyjson_mut_val*)val);
}

static RT_UNUSED rt_string rt_json_write(rt_arena* arena, trux_json_mut_doc* doc) {
    if (!doc) return RT_LIT(rt_string, NULL, 0);
    yyjson_alc alc = { rt_yyjson_malloc, rt_yyjson_realloc, rt_yyjson_free, arena };
    size_t out_len = 0;
    char* json_str = yyjson_mut_write_opts((yyjson_mut_doc*)doc, YYJSON_WRITE_NOFLAG, &alc, &out_len, NULL);
    if (!json_str) return RT_LIT(rt_string, NULL, 0);
    return RT_LIT(rt_string, (const uint8_t*)json_str, out_len);
}
`

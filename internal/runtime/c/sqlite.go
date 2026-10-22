package c

import _ "embed"

const SQLiteHeaderName = "trux_sqlite_runtime.h"

//go:embed sqlite/sqlite3.h
var SQLite3HeaderSource string

//go:embed sqlite/sqlite3.c
var SQLite3CSource string

const SQLiteHeaderSource = `
#ifndef TRUX_SQLITE_RUNTIME_H
#define TRUX_SQLITE_RUNTIME_H

#include "sqlite3.h"

typedef struct trux_sqlite_db trux_sqlite_db;
struct trux_sqlite_db {
    sqlite3* handle;
    trux_sqlite_db* next;
    int is_open;
};

typedef struct trux_sqlite_stmt trux_sqlite_stmt;
struct trux_sqlite_stmt {
    sqlite3_stmt* handle;
    trux_sqlite_db* db;
    trux_sqlite_stmt* next;
    int is_finalized;
};

void rt_sqlite_cleanup_all(void);

#endif /* TRUX_SQLITE_RUNTIME_H */
`

const SQLiteImplSource = `
/* --- SQLite Runtime Implementation --- */
static trux_sqlite_db* g_trux_sqlite_dbs = NULL;
static trux_sqlite_stmt* g_trux_sqlite_stmts = NULL;

RT_UNUSED void rt_sqlite_cleanup_all(void) {
    trux_sqlite_stmt* stmt = g_trux_sqlite_stmts;
    while (stmt != NULL) {
        if (!stmt->is_finalized && stmt->handle != NULL) {
            sqlite3_finalize(stmt->handle);
            stmt->handle = NULL;
            stmt->is_finalized = 1;
        }
        stmt = stmt->next;
    }
    g_trux_sqlite_stmts = NULL;

    trux_sqlite_db* db = g_trux_sqlite_dbs;
    while (db != NULL) {
        if (db->is_open && db->handle != NULL) {
            sqlite3_close_v2(db->handle);
            db->handle = NULL;
            db->is_open = 0;
        }
        db = db->next;
    }
    g_trux_sqlite_dbs = NULL;
}

static RT_UNUSED trux_tuple_ptr_struct_14_trux_sqlite_db_error rt_sqlite_open(rt_arena* arena, rt_string path) {
    char* c_path = rt_string_to_c_string(path, "sqlite open path");
    trux_sqlite_db* db = (trux_sqlite_db*)malloc(sizeof(trux_sqlite_db));
    if (db == NULL) {
        free(c_path);
        return RT_LIT(trux_tuple_ptr_struct_14_trux_sqlite_db_error, NULL, rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"failed to allocate sqlite db handle", 35)));
    }
    db->handle = NULL;
    db->is_open = 0;
    db->next = g_trux_sqlite_dbs;
    g_trux_sqlite_dbs = db;

    int rc = sqlite3_open(c_path, &db->handle);
    free(c_path);
    if (rc != SQLITE_OK) {
        const char* errmsg = db->handle ? sqlite3_errmsg(db->handle) : sqlite3_errstr(rc);
        rt_error err = rt_error_new(arena, rt_string_from_c(arena, errmsg));
        if (db->handle != NULL) {
            sqlite3_close(db->handle);
            db->handle = NULL;
        }
        return RT_LIT(trux_tuple_ptr_struct_14_trux_sqlite_db_error, NULL, err);
    }
    db->is_open = 1;
    return RT_LIT(trux_tuple_ptr_struct_14_trux_sqlite_db_error, db, NULL);
}

static RT_UNUSED rt_error rt_sqlite_close(rt_arena* arena, trux_sqlite_db* db) {
    if (db == NULL || !db->is_open || db->handle == NULL) {
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"database connection is not open", 31));
    }
    int rc = sqlite3_close_v2(db->handle);
    db->handle = NULL;
    db->is_open = 0;
    if (rc != SQLITE_OK) {
        return rt_error_new(arena, rt_string_from_c(arena, sqlite3_errstr(rc)));
    }
    return NULL;
}

static RT_UNUSED rt_error rt_sqlite_exec(rt_arena* arena, trux_sqlite_db* db, rt_string sql) {
    if (db == NULL || !db->is_open || db->handle == NULL) {
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"database connection is not open", 31));
    }
    char* c_sql = rt_string_to_c_string(sql, "sqlite exec sql");
    char* errmsg = NULL;
    int rc = sqlite3_exec(db->handle, c_sql, NULL, NULL, &errmsg);
    free(c_sql);
    if (rc != SQLITE_OK) {
        rt_error err = rt_error_new(arena, rt_string_from_c(arena, errmsg ? errmsg : sqlite3_errstr(rc)));
        if (errmsg != NULL) {
            sqlite3_free(errmsg);
        }
        return err;
    }
    return NULL;
}

static RT_UNUSED trux_tuple_list_map_string_string_error rt_sqlite_query(rt_arena* arena, trux_sqlite_db* db, rt_string sql) {
    if (db == NULL || !db->is_open || db->handle == NULL) {
        return RT_LIT(trux_tuple_list_map_string_string_error, NULL, rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"database connection is not open", 31)));
    }
    char* c_sql = rt_string_to_c_string(sql, "sqlite query sql");
    sqlite3_stmt* stmt = NULL;
    int rc = sqlite3_prepare_v2(db->handle, c_sql, -1, &stmt, NULL);
    free(c_sql);
    if (rc != SQLITE_OK) {
        return RT_LIT(trux_tuple_list_map_string_string_error, NULL, rt_error_new(arena, rt_string_from_c(arena, sqlite3_errmsg(db->handle))));
    }
    rt_list_map_string_string* rows = rt_list_map_string_string_new(arena, 0);
    int col_count = sqlite3_column_count(stmt);
    while ((rc = sqlite3_step(stmt)) == SQLITE_ROW) {
        rt_map_string_string* row = rt_map_string_string_new(arena, col_count);
        for (int i = 0; i < col_count; i++) {
            const char* col_name = sqlite3_column_name(stmt, i);
            const char* text_val = (const char*)sqlite3_column_text(stmt, i);
            rt_string k = rt_string_from_c(arena, col_name ? col_name : "");
            rt_string v = rt_string_from_c(arena, text_val ? text_val : "");
            rt_map_string_string_set(row, k, v);
        }
        rt_list_map_string_string_append(rows, row);
    }
    if (rc != SQLITE_DONE) {
        rt_error err = rt_error_new(arena, rt_string_from_c(arena, sqlite3_errmsg(db->handle)));
        sqlite3_finalize(stmt);
        return RT_LIT(trux_tuple_list_map_string_string_error, rows, err);
    }
    sqlite3_finalize(stmt);
    return RT_LIT(trux_tuple_list_map_string_string_error, rows, NULL);
}

static RT_UNUSED int64_t rt_sqlite_last_insert_rowid(rt_arena* arena, trux_sqlite_db* db) {
    (void)arena;
    if (db == NULL || !db->is_open || db->handle == NULL) {
        return 0;
    }
    return (int64_t)sqlite3_last_insert_rowid(db->handle);
}

static RT_UNUSED int64_t rt_sqlite_changes(rt_arena* arena, trux_sqlite_db* db) {
    (void)arena;
    if (db == NULL || !db->is_open || db->handle == NULL) {
        return 0;
    }
    return (int64_t)sqlite3_changes(db->handle);
}

static RT_UNUSED int64_t rt_sqlite_total_changes(rt_arena* arena, trux_sqlite_db* db) {
    (void)arena;
    if (db == NULL || !db->is_open || db->handle == NULL) {
        return 0;
    }
    return (int64_t)sqlite3_total_changes(db->handle);
}

static RT_UNUSED rt_error rt_sqlite_begin(rt_arena* arena, trux_sqlite_db* db) {
    if (db == NULL || !db->is_open || db->handle == NULL) {
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"database connection is not open", 31));
    }
    int rc = sqlite3_exec(db->handle, "BEGIN", NULL, NULL, NULL);
    if (rc != SQLITE_OK) {
        return rt_error_new(arena, rt_string_from_c(arena, sqlite3_errmsg(db->handle)));
    }
    return NULL;
}

static RT_UNUSED rt_error rt_sqlite_commit(rt_arena* arena, trux_sqlite_db* db) {
    if (db == NULL || !db->is_open || db->handle == NULL) {
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"database connection is not open", 31));
    }
    int rc = sqlite3_exec(db->handle, "COMMIT", NULL, NULL, NULL);
    if (rc != SQLITE_OK) {
        return rt_error_new(arena, rt_string_from_c(arena, sqlite3_errmsg(db->handle)));
    }
    return NULL;
}

static RT_UNUSED rt_error rt_sqlite_rollback(rt_arena* arena, trux_sqlite_db* db) {
    if (db == NULL || !db->is_open || db->handle == NULL) {
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"database connection is not open", 31));
    }
    int rc = sqlite3_exec(db->handle, "ROLLBACK", NULL, NULL, NULL);
    if (rc != SQLITE_OK) {
        return rt_error_new(arena, rt_string_from_c(arena, sqlite3_errmsg(db->handle)));
    }
    return NULL;
}

static RT_UNUSED trux_tuple_ptr_struct_16_trux_sqlite_stmt_error rt_sqlite_prepare(rt_arena* arena, trux_sqlite_db* db, rt_string sql) {
    if (db == NULL || !db->is_open || db->handle == NULL) {
        return RT_LIT(trux_tuple_ptr_struct_16_trux_sqlite_stmt_error, NULL, rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"database connection is not open", 31)));
    }
    char* c_sql = rt_string_to_c_string(sql, "sqlite prepare sql");
    sqlite3_stmt* handle = NULL;
    int rc = sqlite3_prepare_v2(db->handle, c_sql, -1, &handle, NULL);
    free(c_sql);
    if (rc != SQLITE_OK) {
        return RT_LIT(trux_tuple_ptr_struct_16_trux_sqlite_stmt_error, NULL, rt_error_new(arena, rt_string_from_c(arena, sqlite3_errmsg(db->handle))));
    }
    trux_sqlite_stmt* stmt = (trux_sqlite_stmt*)malloc(sizeof(trux_sqlite_stmt));
    if (stmt == NULL) {
        sqlite3_finalize(handle);
        return RT_LIT(trux_tuple_ptr_struct_16_trux_sqlite_stmt_error, NULL, rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"failed to allocate sqlite stmt handle", 37)));
    }
    stmt->handle = handle;
    stmt->db = db;
    stmt->is_finalized = 0;
    stmt->next = g_trux_sqlite_stmts;
    g_trux_sqlite_stmts = stmt;
    return RT_LIT(trux_tuple_ptr_struct_16_trux_sqlite_stmt_error, stmt, NULL);
}

static RT_UNUSED trux_tuple_bool_error rt_sqlite_step(rt_arena* arena, trux_sqlite_stmt* stmt) {
    if (stmt == NULL || stmt->is_finalized || stmt->handle == NULL) {
        return RT_LIT(trux_tuple_bool_error, false, rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"statement is not valid or finalized", 35)));
    }
    int rc = sqlite3_step(stmt->handle);
    if (rc == SQLITE_ROW) {
        return RT_LIT(trux_tuple_bool_error, true, NULL);
    }
    if (rc == SQLITE_DONE) {
        return RT_LIT(trux_tuple_bool_error, false, NULL);
    }
    const char* errmsg = (stmt->db && stmt->db->handle) ? sqlite3_errmsg(stmt->db->handle) : sqlite3_errstr(rc);
    return RT_LIT(trux_tuple_bool_error, false, rt_error_new(arena, rt_string_from_c(arena, errmsg)));
}

static RT_UNUSED rt_error rt_sqlite_reset(rt_arena* arena, trux_sqlite_stmt* stmt) {
    if (stmt == NULL || stmt->is_finalized || stmt->handle == NULL) {
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"statement is not valid or finalized", 35));
    }
    int rc = sqlite3_reset(stmt->handle);
    if (rc != SQLITE_OK) {
        const char* errmsg = (stmt->db && stmt->db->handle) ? sqlite3_errmsg(stmt->db->handle) : sqlite3_errstr(rc);
        return rt_error_new(arena, rt_string_from_c(arena, errmsg));
    }
    return NULL;
}

static RT_UNUSED rt_error rt_sqlite_clear_bindings(rt_arena* arena, trux_sqlite_stmt* stmt) {
    if (stmt == NULL || stmt->is_finalized || stmt->handle == NULL) {
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"statement is not valid or finalized", 35));
    }
    int rc = sqlite3_clear_bindings(stmt->handle);
    if (rc != SQLITE_OK) {
        const char* errmsg = (stmt->db && stmt->db->handle) ? sqlite3_errmsg(stmt->db->handle) : sqlite3_errstr(rc);
        return rt_error_new(arena, rt_string_from_c(arena, errmsg));
    }
    return NULL;
}

static RT_UNUSED rt_error rt_sqlite_finalize(rt_arena* arena, trux_sqlite_stmt* stmt) {
    if (stmt == NULL || stmt->is_finalized || stmt->handle == NULL) {
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"statement is already finalized", 30));
    }
    int rc = sqlite3_finalize(stmt->handle);
    stmt->handle = NULL;
    stmt->is_finalized = 1;
    if (rc != SQLITE_OK) {
        return rt_error_new(arena, rt_string_from_c(arena, sqlite3_errstr(rc)));
    }
    return NULL;
}

static RT_UNUSED rt_error rt_sqlite_bind_int(rt_arena* arena, trux_sqlite_stmt* stmt, int64_t index, int64_t val) {
    if (stmt == NULL || stmt->is_finalized || stmt->handle == NULL) {
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"statement is not valid or finalized", 35));
    }
    int rc = sqlite3_bind_int64(stmt->handle, (int)index, (sqlite3_int64)val);
    if (rc != SQLITE_OK) {
        const char* errmsg = (stmt->db && stmt->db->handle) ? sqlite3_errmsg(stmt->db->handle) : sqlite3_errstr(rc);
        return rt_error_new(arena, rt_string_from_c(arena, errmsg));
    }
    return NULL;
}

static RT_UNUSED rt_error rt_sqlite_bind_float(rt_arena* arena, trux_sqlite_stmt* stmt, int64_t index, double val) {
    if (stmt == NULL || stmt->is_finalized || stmt->handle == NULL) {
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"statement is not valid or finalized", 35));
    }
    int rc = sqlite3_bind_double(stmt->handle, (int)index, val);
    if (rc != SQLITE_OK) {
        const char* errmsg = (stmt->db && stmt->db->handle) ? sqlite3_errmsg(stmt->db->handle) : sqlite3_errstr(rc);
        return rt_error_new(arena, rt_string_from_c(arena, errmsg));
    }
    return NULL;
}

static RT_UNUSED rt_error rt_sqlite_bind_text(rt_arena* arena, trux_sqlite_stmt* stmt, int64_t index, rt_string val) {
    if (stmt == NULL || stmt->is_finalized || stmt->handle == NULL) {
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"statement is not valid or finalized", 35));
    }
    int rc = sqlite3_bind_text64(stmt->handle, (int)index, (const char*)val.data, val.len, SQLITE_TRANSIENT, SQLITE_UTF8);
    if (rc != SQLITE_OK) {
        const char* errmsg = (stmt->db && stmt->db->handle) ? sqlite3_errmsg(stmt->db->handle) : sqlite3_errstr(rc);
        return rt_error_new(arena, rt_string_from_c(arena, errmsg));
    }
    return NULL;
}

static RT_UNUSED rt_error rt_sqlite_bind_null(rt_arena* arena, trux_sqlite_stmt* stmt, int64_t index) {
    if (stmt == NULL || stmt->is_finalized || stmt->handle == NULL) {
        return rt_error_new(arena, RT_LIT(rt_string, (const uint8_t*)"statement is not valid or finalized", 35));
    }
    int rc = sqlite3_bind_null(stmt->handle, (int)index);
    if (rc != SQLITE_OK) {
        const char* errmsg = (stmt->db && stmt->db->handle) ? sqlite3_errmsg(stmt->db->handle) : sqlite3_errstr(rc);
        return rt_error_new(arena, rt_string_from_c(arena, errmsg));
    }
    return NULL;
}

static RT_UNUSED int64_t rt_sqlite_bind_index(rt_arena* arena, trux_sqlite_stmt* stmt, rt_string name) {
    (void)arena;
    if (stmt == NULL || stmt->is_finalized || stmt->handle == NULL) {
        return 0;
    }
    char* c_name = rt_string_to_c_string(name, "sqlite param name");
    int idx = sqlite3_bind_parameter_index(stmt->handle, c_name);
    free(c_name);
    return (int64_t)idx;
}

static RT_UNUSED int64_t rt_sqlite_column_count(rt_arena* arena, trux_sqlite_stmt* stmt) {
    (void)arena;
    if (stmt == NULL || stmt->is_finalized || stmt->handle == NULL) {
        return 0;
    }
    return (int64_t)sqlite3_column_count(stmt->handle);
}

static RT_UNUSED rt_string rt_sqlite_column_name(rt_arena* arena, trux_sqlite_stmt* stmt, int64_t col) {
    if (stmt == NULL || stmt->is_finalized || stmt->handle == NULL) {
        return RT_LIT(rt_string, NULL, 0);
    }
    const char* name = sqlite3_column_name(stmt->handle, (int)col);
    return rt_string_from_c(arena, name ? name : "");
}

static RT_UNUSED int64_t rt_sqlite_column_index(rt_arena* arena, trux_sqlite_stmt* stmt, rt_string name) {
    (void)arena;
    if (stmt == NULL || stmt->is_finalized || stmt->handle == NULL) {
        return -1;
    }
    char* c_name = rt_string_to_c_string(name, "sqlite column name");
    int count = sqlite3_column_count(stmt->handle);
    int64_t found = -1;
    for (int i = 0; i < count; i++) {
        const char* cn = sqlite3_column_name(stmt->handle, i);
        if (cn && strcmp(cn, c_name) == 0) {
            found = (int64_t)i;
            break;
        }
    }
    free(c_name);
    return found;
}

static RT_UNUSED rt_string rt_sqlite_column_type(rt_arena* arena, trux_sqlite_stmt* stmt, int64_t col) {
    (void)arena;
    if (stmt == NULL || stmt->is_finalized || stmt->handle == NULL) {
        return RT_LIT(rt_string, (const uint8_t*)"UNKNOWN", 7);
    }
    int t = sqlite3_column_type(stmt->handle, (int)col);
    switch (t) {
    case SQLITE_INTEGER:
        return RT_LIT(rt_string, (const uint8_t*)"INTEGER", 7);
    case SQLITE_FLOAT:
        return RT_LIT(rt_string, (const uint8_t*)"FLOAT", 5);
    case SQLITE_TEXT:
        return RT_LIT(rt_string, (const uint8_t*)"TEXT", 4);
    case SQLITE_BLOB:
        return RT_LIT(rt_string, (const uint8_t*)"BLOB", 4);
    case SQLITE_NULL:
        return RT_LIT(rt_string, (const uint8_t*)"NULL", 4);
    default:
        return RT_LIT(rt_string, (const uint8_t*)"UNKNOWN", 7);
    }
}

static RT_UNUSED int64_t rt_sqlite_column_int(rt_arena* arena, trux_sqlite_stmt* stmt, int64_t col) {
    (void)arena;
    if (stmt == NULL || stmt->is_finalized || stmt->handle == NULL) {
        return 0;
    }
    return (int64_t)sqlite3_column_int64(stmt->handle, (int)col);
}

static RT_UNUSED double rt_sqlite_column_float(rt_arena* arena, trux_sqlite_stmt* stmt, int64_t col) {
    (void)arena;
    if (stmt == NULL || stmt->is_finalized || stmt->handle == NULL) {
        return 0.0;
    }
    return (double)sqlite3_column_double(stmt->handle, (int)col);
}

static RT_UNUSED rt_string rt_sqlite_column_text(rt_arena* arena, trux_sqlite_stmt* stmt, int64_t col) {
    if (stmt == NULL || stmt->is_finalized || stmt->handle == NULL) {
        return RT_LIT(rt_string, NULL, 0);
    }
    const char* text = (const char*)sqlite3_column_text(stmt->handle, (int)col);
    return rt_string_from_c(arena, text ? text : "");
}

static RT_UNUSED bool rt_sqlite_column_is_null(rt_arena* arena, trux_sqlite_stmt* stmt, int64_t col) {
    (void)arena;
    if (stmt == NULL || stmt->is_finalized || stmt->handle == NULL) {
        return true;
    }
    return sqlite3_column_type(stmt->handle, (int)col) == SQLITE_NULL;
}
`

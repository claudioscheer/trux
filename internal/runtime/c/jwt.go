package c

const JWTHeaderName = "trux_jwt_runtime.h"

const JWTHeaderSource = `
#ifndef TRUX_JWT_RUNTIME_H
#define TRUX_JWT_RUNTIME_H

#include "yyjson.h"
#include <jwt.h>

typedef struct trux_jwt_claims {
    yyjson_doc *doc;
} trux_jwt_claims;

#endif /* TRUX_JWT_RUNTIME_H */
`

const JWTImplSource = `
/* --- JWT (libjwt HS256) Runtime Implementation --- */

static void *rt_jwt_yyjson_malloc(void *ctx, size_t size) {
    return rt_arena_alloc((rt_arena *)ctx, size);
}

static void *rt_jwt_yyjson_realloc(void *ctx, void *ptr, size_t old_size, size_t size) {
    void *new_ptr = rt_arena_alloc((rt_arena *)ctx, size);
    if (ptr && old_size > 0 && new_ptr) {
        memcpy(new_ptr, ptr, old_size < size ? old_size : size);
    }
    return new_ptr;
}

static void rt_jwt_yyjson_free(void *ctx, void *ptr) {
    (void)ctx;
    (void)ptr;
}

static RT_UNUSED void trux_jwt_keep(char *dst, size_t n, const char *msg) {
    if (msg == NULL || msg[0] == '\0') {
        msg = "jwt error";
    }
    snprintf(dst, n, "%s", msg);
}

static RT_UNUSED char *trux_jwt_cstr(rt_string value, char *errbuf, size_t errlen) {
    rt_check_string(value);
    for (size_t i = 0; i < value.len; i++) {
        if (value.data[i] == 0) {
            trux_jwt_keep(errbuf, errlen, "jwt string contains a NUL byte");
            return NULL;
        }
    }
    char *out = (char *)malloc(value.len + 1);
    if (out == NULL) {
        trux_jwt_keep(errbuf, errlen, "jwt allocation failed");
        return NULL;
    }
    if (value.len > 0) {
        memcpy(out, value.data, value.len);
    }
    out[value.len] = '\0';
    return out;
}

static RT_UNUSED char *trux_jwt_b64url(const unsigned char *in, size_t len, char *errbuf, size_t errlen) {
    static const char table[] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_";
    size_t out_len = (len / 3) * 4;
    if (len % 3 == 1) {
        out_len += 2;
    } else if (len % 3 == 2) {
        out_len += 3;
    }
    char *out = (char *)malloc(out_len + 1);
    if (out == NULL) {
        trux_jwt_keep(errbuf, errlen, "jwt allocation failed");
        return NULL;
    }
    size_t i = 0;
    size_t j = 0;
    while (i + 3 <= len) {
        unsigned int n = ((unsigned int)in[i] << 16) | ((unsigned int)in[i + 1] << 8) | (unsigned int)in[i + 2];
        out[j++] = table[(n >> 18) & 63];
        out[j++] = table[(n >> 12) & 63];
        out[j++] = table[(n >> 6) & 63];
        out[j++] = table[n & 63];
        i += 3;
    }
    if (i < len) {
        unsigned int n = ((unsigned int)in[i] << 16);
        if (i + 1 < len) {
            n |= ((unsigned int)in[i + 1] << 8);
        }
        out[j++] = table[(n >> 18) & 63];
        out[j++] = table[(n >> 12) & 63];
        if (i + 1 < len) {
            out[j++] = table[(n >> 6) & 63];
        }
    }
    out[j] = '\0';
    return out;
}

static RT_UNUSED int trux_jwt_claims_object(const char *json, char *errbuf, size_t errlen) {
    yyjson_read_err read_err;
    yyjson_doc *doc = yyjson_read_opts((char *)json, strlen(json), YYJSON_READ_NOFLAG, NULL, &read_err);
    if (doc == NULL || !yyjson_is_obj(yyjson_doc_get_root(doc))) {
        if (doc != NULL) {
            yyjson_doc_free(doc);
        }
        trux_jwt_keep(errbuf, errlen, "jwt claims must be a JSON object");
        return 1;
    }
    yyjson_doc_free(doc);
    return 0;
}

static RT_UNUSED trux_tuple_string_error rt_jwt_sign(rt_arena *arena, rt_string claims, rt_string secret) {
    char errbuf[512];
    errbuf[0] = '\0';
    char *claims_c = NULL;
    char *secret_c = NULL;
    char *k = NULL;
    char *jwk_json = NULL;
    char *token = NULL;
    jwk_set_t *set = NULL;
    jwt_builder_t *builder = NULL;
    rt_string signed_token = RT_LIT(rt_string, NULL, 0);
    int ok = 0;

    if (claims.len == 0) {
        trux_jwt_keep(errbuf, sizeof(errbuf), "jwt claims are empty");
        goto done;
    }
    if (secret.len == 0) {
        trux_jwt_keep(errbuf, sizeof(errbuf), "jwt secret is empty");
        goto done;
    }
    claims_c = trux_jwt_cstr(claims, errbuf, sizeof(errbuf));
    secret_c = trux_jwt_cstr(secret, errbuf, sizeof(errbuf));
    if (claims_c == NULL || secret_c == NULL) {
        goto done;
    }
    if (trux_jwt_claims_object(claims_c, errbuf, sizeof(errbuf)) != 0) {
        goto done;
    }
    k = trux_jwt_b64url((const unsigned char *)secret_c, strlen(secret_c), errbuf, sizeof(errbuf));
    if (k == NULL) {
        goto done;
    }
    {
        size_t jlen = strlen(k) + 40;
        jwk_json = (char *)malloc(jlen);
        if (jwk_json == NULL) {
            trux_jwt_keep(errbuf, sizeof(errbuf), "jwt allocation failed");
            goto done;
        }
        snprintf(jwk_json, jlen, "{\"kty\":\"oct\",\"k\":\"%s\",\"alg\":\"HS256\"}", k);
    }
    set = jwks_create(jwk_json);
    if (set == NULL || jwks_error(set) || jwks_item_count(set) == 0) {
        trux_jwt_keep(errbuf, sizeof(errbuf), set == NULL ? "jwt secret is not a valid HS256 key" : jwks_error_msg(set));
        goto done;
    }
    builder = jwt_builder_new();
    if (builder == NULL) {
        trux_jwt_keep(errbuf, sizeof(errbuf), "jwt allocation failed");
        goto done;
    }
    if (jwt_builder_setkey(builder, JWT_ALG_HS256, jwks_item_get(set, 0)) != 0) {
        trux_jwt_keep(errbuf, sizeof(errbuf), jwt_builder_error_msg(builder));
        goto done;
    }
    jwt_builder_enable_iat(builder, 0);
    {
        jwt_value_t value;
        memset(&value, 0, sizeof(value));
        value.type = JWT_VALUE_JSON;
        value.name = NULL;
        value.json_val = claims_c;
        value.replace = 1;
        if (jwt_builder_claim_set(builder, &value) != JWT_VALUE_ERR_NONE) {
            trux_jwt_keep(errbuf, sizeof(errbuf), "jwt claims must be a JSON object");
            goto done;
        }
    }
    token = jwt_builder_generate(builder);
    if (token == NULL) {
        trux_jwt_keep(errbuf, sizeof(errbuf), jwt_builder_error_msg(builder));
        goto done;
    }
    signed_token = rt_string_from_c(arena, token);
    ok = 1;

done:
    if (builder != NULL) {
        jwt_builder_free(builder);
    }
    if (set != NULL) {
        jwks_free(set);
    }
    free(token);
    free(jwk_json);
    free(k);
    free(secret_c);
    free(claims_c);
    if (!ok) {
        return RT_LIT(trux_tuple_string_error, RT_LIT(rt_string, NULL, 0), rt_error_new(arena, rt_string_from_c(arena, errbuf)));
    }
    return RT_LIT(trux_tuple_string_error, signed_token, NULL);
}

typedef struct trux_jwt_cap {
    char *json;
} trux_jwt_cap;

static int trux_jwt_capture_claims(jwt_t *jwt, jwt_config_t *cfg) {
    trux_jwt_cap *cap = (trux_jwt_cap *)cfg->ctx;
    jwt_value_t value;
    memset(&value, 0, sizeof(value));
    value.type = JWT_VALUE_JSON;
    value.name = NULL;
    if (jwt_claim_get(jwt, &value) != JWT_VALUE_ERR_NONE || value.json_val == NULL) {
        return 1;
    }
    cap->json = value.json_val;
    return 0;
}

static RT_UNUSED trux_tuple_ptr_struct_15_trux_jwt_claims_error rt_jwt_verify(rt_arena *arena, rt_string token, rt_string secret) {
    char errbuf[512];
    errbuf[0] = '\0';
    char *token_c = NULL;
    char *secret_c = NULL;
    char *k = NULL;
    char *jwk_json = NULL;
    jwk_set_t *set = NULL;
    jwt_checker_t *checker = NULL;
    trux_jwt_cap cap;
    trux_jwt_claims *claims = NULL;
    int ok = 0;
    memset(&cap, 0, sizeof(cap));

    if (token.len == 0) {
        trux_jwt_keep(errbuf, sizeof(errbuf), "jwt token is empty");
        goto done;
    }
    if (secret.len == 0) {
        trux_jwt_keep(errbuf, sizeof(errbuf), "jwt secret is empty");
        goto done;
    }
    token_c = trux_jwt_cstr(token, errbuf, sizeof(errbuf));
    secret_c = trux_jwt_cstr(secret, errbuf, sizeof(errbuf));
    if (token_c == NULL || secret_c == NULL) {
        goto done;
    }
    k = trux_jwt_b64url((const unsigned char *)secret_c, strlen(secret_c), errbuf, sizeof(errbuf));
    if (k == NULL) {
        goto done;
    }
    {
        size_t jlen = strlen(k) + 40;
        jwk_json = (char *)malloc(jlen);
        if (jwk_json == NULL) {
            trux_jwt_keep(errbuf, sizeof(errbuf), "jwt allocation failed");
            goto done;
        }
        snprintf(jwk_json, jlen, "{\"kty\":\"oct\",\"k\":\"%s\",\"alg\":\"HS256\"}", k);
    }
    set = jwks_create(jwk_json);
    if (set == NULL || jwks_error(set) || jwks_item_count(set) == 0) {
        trux_jwt_keep(errbuf, sizeof(errbuf), set == NULL ? "jwt secret is not a valid HS256 key" : jwks_error_msg(set));
        goto done;
    }
    checker = jwt_checker_new();
    if (checker == NULL) {
        trux_jwt_keep(errbuf, sizeof(errbuf), "jwt allocation failed");
        goto done;
    }
    if (jwt_checker_setkey(checker, JWT_ALG_HS256, jwks_item_get(set, 0)) != 0) {
        trux_jwt_keep(errbuf, sizeof(errbuf), jwt_checker_error_msg(checker));
        goto done;
    }
    {
        jwt_alg_t algs[1];
        algs[0] = JWT_ALG_HS256;
        if (jwt_checker_setalgs(checker, algs, 1) != 0) {
            trux_jwt_keep(errbuf, sizeof(errbuf), jwt_checker_error_msg(checker));
            goto done;
        }
    }
    if (jwt_checker_setcb(checker, trux_jwt_capture_claims, &cap) != 0) {
        trux_jwt_keep(errbuf, sizeof(errbuf), jwt_checker_error_msg(checker));
        goto done;
    }
    if (jwt_checker_verify(checker, token_c) != 0) {
        trux_jwt_keep(errbuf, sizeof(errbuf), jwt_checker_error_msg(checker));
        goto done;
    }
    if (cap.json == NULL || trux_jwt_claims_object(cap.json, errbuf, sizeof(errbuf)) != 0) {
        if (errbuf[0] == '\0') {
            trux_jwt_keep(errbuf, sizeof(errbuf), "jwt claims must be a JSON object");
        }
        goto done;
    }
    {
        yyjson_read_err read_err;
        yyjson_alc alc = {rt_jwt_yyjson_malloc, rt_jwt_yyjson_realloc, rt_jwt_yyjson_free, arena};
        yyjson_doc *doc = yyjson_read_opts(cap.json, strlen(cap.json), YYJSON_READ_NOFLAG, &alc, &read_err);
        if (doc == NULL || !yyjson_is_obj(yyjson_doc_get_root(doc))) {
            trux_jwt_keep(errbuf, sizeof(errbuf), "jwt claims must be a JSON object");
            goto done;
        }
        claims = (trux_jwt_claims *)rt_arena_alloc(arena, sizeof(*claims));
        claims->doc = doc;
    }
    ok = 1;

done:
    free(cap.json);
    if (checker != NULL) {
        jwt_checker_free(checker);
    }
    if (set != NULL) {
        jwks_free(set);
    }
    free(jwk_json);
    free(k);
    free(secret_c);
    free(token_c);
    if (!ok) {
        return RT_LIT(trux_tuple_ptr_struct_15_trux_jwt_claims_error, NULL, rt_error_new(arena, rt_string_from_c(arena, errbuf)));
    }
    return RT_LIT(trux_tuple_ptr_struct_15_trux_jwt_claims_error, claims, NULL);
}

static RT_UNUSED yyjson_val *trux_jwt_claim_val(trux_jwt_claims *claims, rt_string key) {
    if (claims == NULL || claims->doc == NULL) {
        return NULL;
    }
    yyjson_val *root = yyjson_doc_get_root(claims->doc);
    if (!yyjson_is_obj(root)) {
        return NULL;
    }
    if (key.len > 0 && key.data == NULL) {
        return NULL;
    }
    return yyjson_obj_getn(root, (const char *)key.data, key.len);
}

static RT_UNUSED rt_string rt_jwt_get_string(rt_arena *arena, trux_jwt_claims *claims, rt_string key) {
    yyjson_val *val = trux_jwt_claim_val(claims, key);
    if (val == NULL) {
        return RT_LIT(rt_string, NULL, 0);
    }
    const char *s = yyjson_get_str(val);
    if (s == NULL) {
        return RT_LIT(rt_string, NULL, 0);
    }
    size_t len = yyjson_get_len(val);
    uint8_t *copy = (uint8_t *)rt_arena_alloc(arena, len);
    memcpy(copy, s, len);
    return RT_LIT(rt_string, copy, len);
}

static RT_UNUSED int64_t rt_jwt_get_int(rt_arena *arena, trux_jwt_claims *claims, rt_string key) {
    (void)arena;
    yyjson_val *val = trux_jwt_claim_val(claims, key);
    if (val == NULL) {
        return 0;
    }
    if (yyjson_is_int(val)) {
        return yyjson_get_sint(val);
    }
    if (yyjson_is_uint(val)) {
        return (int64_t)yyjson_get_uint(val);
    }
    if (yyjson_is_real(val)) {
        return (int64_t)yyjson_get_real(val);
    }
    if (yyjson_is_bool(val)) {
        return yyjson_get_bool(val) ? 1 : 0;
    }
    return 0;
}

static RT_UNUSED bool rt_jwt_get_bool(rt_arena *arena, trux_jwt_claims *claims, rt_string key) {
    (void)arena;
    yyjson_val *val = trux_jwt_claim_val(claims, key);
    if (val == NULL) {
        return false;
    }
    return yyjson_get_bool(val);
}

static RT_UNUSED double rt_jwt_get_float(rt_arena *arena, trux_jwt_claims *claims, rt_string key) {
    (void)arena;
    yyjson_val *val = trux_jwt_claim_val(claims, key);
    if (val == NULL) {
        return 0.0;
    }
    if (yyjson_is_num(val)) {
        return yyjson_get_num(val);
    }
    return 0.0;
}
`

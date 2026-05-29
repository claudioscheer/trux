package c

const Source = `#include <stdio.h>
#include <string.h>

typedef struct {
    const uint8_t* data;
    size_t len;
} rt_string;

static void rt_print_int(int64_t value) {
    printf("%lld", (long long)value);
}

static void rt_print_float(double value) {
    printf("%.15g", value);
}

static void rt_print_string(rt_string value) {
    fwrite(value.data, 1, value.len, stdout);
}

static void rt_print_bool(bool value) {
    printf("%s", value ? "true" : "false");
}

static void rt_print_newline(void) {
    putchar('\n');
}

static bool rt_string_equal(rt_string left, rt_string right) {
    if (left.len != right.len) {
        return false;
    }
    if (left.len == 0) {
        return true;
    }
    return memcmp(left.data, right.data, left.len) == 0;
}

static bool rt_string_contains(rt_string needle, rt_string haystack) {
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
`

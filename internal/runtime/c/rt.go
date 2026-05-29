package c

const Source = `#include <stdio.h>

typedef struct {
    const uint8_t* data;
    size_t len;
} rt_string;

static void rt_print_int(int64_t value) {
    printf("%lld", (long long)value);
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
`

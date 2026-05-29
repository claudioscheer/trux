package c

const Source = `#include <stdio.h>

static void rt_print_int(int64_t value) {
    printf("%lld\n", (long long)value);
}
`

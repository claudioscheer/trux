#include "trux_runtime.h"

int64_t trux_f_3_add(rt_context* trux_ctx, rt_arena* trux_result_arena, int64_t trux_v_1_a, int64_t trux_v_1_b);
int64_t trux_f_4_main(rt_context* trux_ctx, rt_arena* trux_result_arena);

#line 5 "hello.tx"
int64_t trux_f_3_add(rt_context* trux_ctx, rt_arena* trux_result_arena, int64_t trux_v_1_a, int64_t trux_v_1_b) {
    (void)trux_ctx;
    (void)trux_result_arena;
    int64_t trux_return_value;
    #line 6 "hello.tx"
    trux_return_value = rt_add_i64(trux_v_1_a, trux_v_1_b);
    goto trux_return;
    rt_runtime_fail("missing return");
trux_return:
    return trux_return_value;
}

#line 9 "hello.tx"
int64_t trux_f_4_main(rt_context* trux_ctx, rt_arena* trux_result_arena) {
    (void)trux_ctx;
    (void)trux_result_arena;
    int64_t trux_return_value;
    #line 11 "hello.tx"
    int64_t trux_v_1_x = trux_f_3_add(trux_ctx, trux_result_arena, 1, 2);
    #line 14 "hello.tx"
    rt_print_int(trux_v_1_x);
    rt_print_newline();
    #line 16 "hello.tx"
    trux_return_value = 0;
    goto trux_return;
    rt_runtime_fail("missing return");
trux_return:
    return trux_return_value;
}

int main(void) {
    rt_stdio_binary();
    rt_arena trux_arena;
    rt_arena_init(&trux_arena);
    rt_context trux_ctx = {&trux_arena, NULL};
    int64_t trux_exit_code = trux_f_4_main(&trux_ctx, &trux_arena);
    rt_context_free_chunks(&trux_ctx);
    rt_arena_deinit(&trux_arena);
    return (int)trux_exit_code;
}

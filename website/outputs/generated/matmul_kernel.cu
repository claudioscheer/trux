#line 15 "gpu_matrix_multiply.tx"
__global__ void trux_k_12_matmulKernel(int64_t trux_dim_x, int64_t trux_dim_y, int64_t* __restrict__ trux_v_1_a, int64_t* __restrict__ trux_v_1_b, int64_t* __restrict__ trux_v_3_out, int64_t trux_v_1_n) {
    int64_t trux_v_3_row = ((int64_t)(blockIdx.y * blockDim.y + threadIdx.y));
    int64_t trux_v_3_col = ((int64_t)(blockIdx.x * blockDim.x + threadIdx.x));
    if (trux_v_3_row < trux_dim_y && trux_v_3_col < trux_dim_x) {
        #line 16 "gpu_matrix_multiply.tx"
        int64_t trux_v_3_sum = 0;
        #line 18 "gpu_matrix_multiply.tx"
        {
            #line 18 "gpu_matrix_multiply.tx"
            int64_t trux_v_1_k = 0;
            for (; trux_v_1_k < trux_v_1_n;) {
                #line 19 "gpu_matrix_multiply.tx"
                trux_v_3_sum = rt_add_i64(trux_v_3_sum, rt_mul_i64(trux_v_1_a[rt_add_i64(rt_mul_i64(trux_v_3_row, trux_v_1_n), trux_v_1_k)], trux_v_1_b[rt_add_i64(rt_mul_i64(trux_v_1_k, trux_v_1_n), trux_v_3_col)]));
                #line 18 "gpu_matrix_multiply.tx"
                trux_v_1_k = rt_add_i64(trux_v_1_k, 1);
            }
        }
        #line 22 "gpu_matrix_multiply.tx"
        trux_v_3_out[rt_add_i64(rt_mul_i64(trux_v_3_row, trux_v_1_n), trux_v_3_col)] = trux_v_3_sum;
    }
}

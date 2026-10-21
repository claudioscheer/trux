trux_k_12_matmulKernel<<<dim3((unsigned int)(((trux_v_1_n) + 15) / 16), (unsigned int)(((trux_v_1_n) + 15) / 16), 1u), dim3(16u, 16u, 1u)>>>(trux_v_1_n, trux_v_1_n, trux_v_2_dA.data, trux_v_2_dB.data, trux_v_4_dOut.data, trux_v_1_n);
rt_cuda_check(cudaGetLastError(), "launch");
#line 112 "gpu_matrix_multiply.tx"
rt_gpu_sync();
#line 113 "gpu_matrix_multiply.tx"
rt_gpu_copy_to_host_int(trux_v_4_dOut, trux_v_6_gpuOut);

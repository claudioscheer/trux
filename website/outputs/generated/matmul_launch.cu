trux_k_12_matmulKernel<<<dim3((unsigned int)(((trux_v_1_n) + 15) / 16), (unsigned int)(((trux_v_1_n) + 15) / 16), 1u), dim3(16u, 16u, 1u), 0, rt_gpu_launch_stream()>>>(trux_v_1_n, trux_v_1_n, trux_v_2_dA.data, trux_v_2_dB.data, trux_v_4_dOut.data, trux_v_1_n);
rt_cuda_check(cudaGetLastError(), "launch");

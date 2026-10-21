static RT_UNUSED void rt_gpu_copy_to_host_int(rt_gpu_buffer_int device, rt_slice_int host) {
    if (host.len > device.len) rt_runtime_fail("gpu copyToHost host slice is larger than device buffer");
    rt_cuda_check(cudaMemcpy(host.data, device.data, host.len * sizeof(int64_t), cudaMemcpyDeviceToHost), "copyToHost");
}
static RT_UNUSED void rt_gpu_sync(void) {
    rt_cuda_check(cudaDeviceSynchronize(), "sync");
}

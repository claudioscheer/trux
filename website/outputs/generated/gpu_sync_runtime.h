static RT_UNUSED void rt_gpu_copy_to_host_int(rt_gpu_buffer_int device, rt_slice_int host) {
    if (host.len > device.len) rt_runtime_fail("gpu copyToHost host slice is larger than device buffer");
    if (g_rt_gpu_stream_ready) {
        size_t bytes = host.len * sizeof(int64_t);
        rt_gpu_ws_slot* slot = rt_gpu_ws_by_device(device.data);
        if (slot != NULL && slot->pinned != NULL) {
            rt_cuda_check(cudaMemcpyAsync(slot->pinned, device.data, bytes, cudaMemcpyDeviceToHost, g_rt_gpu_stream), "copyToHost");
            rt_cuda_check(cudaStreamSynchronize(g_rt_gpu_stream), "copyToHost");
            memcpy(host.data, slot->pinned, bytes);
            return;
        }
        rt_cuda_check(cudaMemcpyAsync(host.data, device.data, bytes, cudaMemcpyDeviceToHost, g_rt_gpu_stream), "copyToHost");
        rt_cuda_check(cudaStreamSynchronize(g_rt_gpu_stream), "copyToHost");
        return;
    }
    rt_cuda_check(cudaMemcpy(host.data, device.data, host.len * sizeof(int64_t), cudaMemcpyDeviceToHost), "copyToHost");
}
static RT_UNUSED void rt_gpu_sync(void) {
    if (g_rt_gpu_stream_ready) {
        rt_cuda_check(cudaStreamSynchronize(g_rt_gpu_stream), "sync");
        return;
    }
    rt_cuda_check(cudaDeviceSynchronize(), "sync");
}

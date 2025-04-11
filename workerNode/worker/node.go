package worker

type GPU struct {
	GPUModel      string `json:"gpu_model"`       // 显卡型号
	TotalMemoryMB uint64 `json:"total_memory_mb"` // 最大显存
	FreeMemoryMB  uint64 `json:"free_memory_mb"`  // 可用显存
}

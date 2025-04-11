package cluster

import (
	"time"
)

// Node
type Node struct {
	NodeID     string
	IP         string
	Port       string
	LastActive time.Time
	Status     string         // 节点状态 "online", "offline", "unhealthy"
	GPUs       map[string]GPU // 显卡状态（可能有多张）
}

type GPU struct {
	GPUModel      string `json:"gpu_model"`       // 显卡型号
	TotalMemoryMB uint64 `json:"total_memory_mb"` // 最大显存
	FreeMemoryMB  uint64 `json:"free_memory_mb"`  // 可用显存
}

package main

import (
	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

type GPU struct {
	GPUModel      string `json:"gpu_model"`       // 显卡型号
	TotalMemoryMB uint64 `json:"total_memory_mb"` // 最大显存
	FreeMemoryMB  uint64 `json:"free_memory_mb"`  // 可用显存
}

func GetGPUInfo() ([]GPU, error) {
	if err := nvml.Init(); err != nil {
		return nil, err
	}
	defer nvml.Shutdown()

	count, err := nvml.DeviceGetCount()
	if err != nil {
		return nil, err
	}

	var gpus []GPU
	for i := 0; i < count; i++ {
		device, err := nvml.DeviceGetHandleByIndex(i)
		if err != nil {
			continue
		}

		name, _ := device.GetName()
		memInfo, _ := device.GetMemoryInfo()

		gpus = append(gpus, GPU{
			GPUModel:      name,
			TotalMemoryMB: memInfo.Total / 1024 / 1024, // 转换为MB
			FreeMemoryMB:  memInfo.Free / 1024 / 1024,
		})
	}
	return gpus, nil
}

func main() {

}

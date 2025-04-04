package container

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount" // 挂载相关
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat" // 端口映射相关
)

type ModelConfig struct {
	ImageName    string
	ModelPath    string
	PortMapping  string
	EnvVars      map[string]string
	VolumeMounts map[string]string
	Command      []string
}

var modelConfigs = map[string]ModelConfig{
	"llama3-8b": {
		ImageName:    "model:v1",
		ModelPath:    "/models/Meta-Llama-3-8B",
		PortMapping:  "8000:8000",
		EnvVars:      map[string]string{"MODEL_NAME": "/models/Meta-Llama-3-8B"},
		VolumeMounts: map[string]string{"/root/Models": "/models"},
		Command:      []string{"python", "server.py"},
	},
	"gpt-neo-2.7b": {
		// 其他模型配置
	},
}

func StartModelContainer(modelName string) (string, error) {
	config, exists := modelConfigs[modelName]
	if !exists {
		return "", fmt.Errorf("model %s not supported", modelName)
	}

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return "", err
	}

	// 准备环境变量
	var envVars []string
	for k, v := range config.EnvVars {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	// 准备挂载卷
	var mounts []mount.Mount
	for src, dst := range config.VolumeMounts {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: src,
			Target: dst,
		})
	}

	// 创建容器
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: config.ImageName,
		Cmd:   config.Command,
		Env:   envVars,
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			"8000/tcp": []nat.PortBinding{{HostPort: "8000"}},
		},
		Mounts: mounts,
	}, nil, nil, "")
	if err != nil {
		return "", err
	}

	// 启动容器
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", err
	}

	return resp.ID, nil
}

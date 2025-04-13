package container

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount" // 挂载相关
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat" // 端口映射相关
)

// 创建推理实例，要加载的模型名称，通过环境变量传入
func StartModelContainer(modelName string) (string, error) {
	// 设置环境变量，以免docker 客户端与服务端api版本不一致报错
	os.Setenv("DOCKER_API_VERSION", "1.43")
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

	// TODO：获取系统中一个可用的端口号
	host_port := "31122"

	// 定义端口映射
	portBindings := nat.PortMap{
		"8000/tcp": []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: host_port,
			},
		},
	}

	// TODO: 把容器名字设置成和任务ID相关
	containerName := "tsif"
	// 创建容器
	resp, err := cli.ContainerCreate(ctx,
		&container.Config{
			Image: config.ImageName,
			Cmd:   config.Command,
			Env:   envVars,
			// 容器暴露的端口
			ExposedPorts: nat.PortSet{
				"8000/tcp": struct{}{},
			},
		},
		&container.HostConfig{
			PortBindings: portBindings,
			Mounts:       mounts,
			Privileged:   true,
			// Runtime:    "nvidia",
			Resources: container.Resources{
				DeviceRequests: []container.DeviceRequest{
					{
						Driver:       "nvidia",
						Count:        -1,
						Capabilities: [][]string{{"gpu", "nvidia", "compute", "utility"}},
					},
				},
			},
		},
		nil, nil, containerName)
	if err != nil {
		return "", err
	}

	// 启动容器
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", err
	}

	return host_port, nil
}

func DeleteContainer(containerName string) error {
	//  client.WithAPIVersionNegotiation()防止api版本不对齐而报错
	// 创建Docker客户端
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("无法创建docker客户端: %v", err)
		return err
	}
	defer cli.Close()

	// 获取所有容器
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{
		All: true, // 包括停止的容器
	})
	if err != nil {
		log.Fatalf("Error listing containers: %v", err)
		return err
	}

	// 查找指定名称的容器
	var containerID string
	for _, container := range containers {
		for _, name := range container.Names {
			if name == "/"+containerName {
				containerID = container.ID
				break
			}
		}
		if containerID != "" {
			break
		}
	}

	if containerID == "" {
		log.Fatalf("Container with name '%s' not found", containerName)
	}

	// 删除容器 - 使用新的container.RemoveOptions
	err = cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{
		Force: true, // 强制删除运行中的容器
	})
	if err != nil {
		log.Fatalf("Error removing container: %v", err)
	}

	fmt.Printf("成功删除容器: %s\n", containerName)
	return nil
}

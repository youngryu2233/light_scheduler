package main

import (
	"lightScheduler/cluster"
	"lightScheduler/task"
	"log"
	"time"
)

func main() {
	// 创建集群管理器，设置心跳间隔为5秒，超时时间为15秒
	cm := cluster.NewClusterManager(5*time.Second, 15*time.Second)

	// 启动健康检查
	go cm.StartHealthCheck()

	// 启动注册&心跳监测HTTP服务器
	go func() {
		if err := cm.StartHeartbeatHTTPServer("8080"); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// 创建任务等待队列
	wq := task.NewTaskWaitQueue(128)
	// 启动队伍处理，不断检查队伍中是否有新的任务
	go wq.HandleQueue(cm)
	// 启动接受推理请求的服务器
	wq.StartTaskHTTPServer("9090")

	// 保持主goroutine存活，防止退出
	// select {}

}

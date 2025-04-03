package main

import (
	"lightScheduler/cluster"
	"log"
	"time"
)

func main() {
	// 创建集群管理器，设置心跳间隔为5秒，超时时间为15秒
	cm := cluster.NewClusterManager(5*time.Second, 15*time.Second)

	// 启动健康检查
	go cm.StartHealthCheck()

	// 启动心跳监测HTTP服务器
	if err := cm.StartHTTPServer("8080"); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}

	// 保持主goroutine存活，防止退出
	// select {}

	// 客户端启动（最终要分离出来）

}

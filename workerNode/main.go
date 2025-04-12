package main

import (
	// "log"
	// "os"
	// "os/signal"
	// "syscall"
	// "time"

	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"workerNode/worker"
)

func main() {

	// 创建节点配置
	config := &worker.Config{
		NodeID:    "node-1",
		IP:        "127.0.0.1",
		Port:      "7070",
		ServerURL: "http://localhost:8080",
		Interval:  200000 * time.Second,
		Timeout:   10 * time.Second,
	}

	// 创建工作节点
	node := worker.NewWorker(config)

	// 连接到集群中，注册节点，并且开启心跳协程
	go func() {
		if err := node.StartLink(); err != nil {
			log.Fatalf("启动失败: %v", err)
		}
	}()

	// 启动调度服务器
	node.StartScheduler("10000")

	// 设置信号处理
	// 创建一个缓冲大小为1的os.Signal通道，用于接受系统信号
	sigChan := make(chan os.Signal, 1)
	// 把SIGINT和.SIGTERM两种信号转发到sigChan通道中
	// SIGINT是用户按下ctrl+C，sigChan是系统要求进程终止
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 在此阻塞，直到sigChan通道接收到信号
	<-sigChan
	log.Println("接收到终止信号，正在关闭...")

	// 停止客户端
	node.Stop()
	log.Println("节点已正常退出")

}

// func main() {

// 	os.Setenv("DOCKER_API_VERSION", "1.43")
// 	containerID, err := container.StartModelContainer("llama3-8b")
// 	if err != nil {
// 		fmt.Printf(err.Error())
// 	} else {
// 		fmt.Printf("容器ID是：%s", containerID)
// 	}

// }

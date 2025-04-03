package cluster

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

type ClusterManager struct {
	mu        sync.RWMutex
	nodes     map[string]*Node
	heartbeat time.Duration
	timeout   time.Duration
	// 设置一个通道，用于在不同goroutine之间通信
	// 当这个通道监听到信号，则停止健康检查
	stopChan   chan struct{}
	httpServer *http.Server
}

// 创建一个新的集群管理器
func NewClusterManager(heartbeat, timeout time.Duration) *ClusterManager {
	return &ClusterManager{
		nodes:     make(map[string]*Node),
		heartbeat: heartbeat,
		timeout:   timeout,
		stopChan:  make(chan struct{}),
	}
}

func (cm *ClusterManager) RegisterNode(id, ip, port string) error {
	// 加锁，函数返回时解锁
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 如果节点已经注册，这期间又收到相同节点的注册请求，直接返回
	_, exits := cm.nodes[id]
	if exits {
		return nil
	}

	// 检查是否已注册，已注册就返回错误
	// if _, exits := cm.nodes[id]; exits {
	// 	return fmt.Errorf("node with ID %s already exists", id)
	// }

	// 添加节点
	cm.nodes[id] = &Node{
		NodeID:     id,
		IP:         ip,
		Port:       port,
		LastActive: time.Now(),
		Status:     "online",
	}

	log.Printf("Node %s registered at %s:%s", id, ip, port)
	return nil
}

// 更新节点的心跳时间为现在，状态设置为健康
func (cm *ClusterManager) UpdateHeartbeat(nodeID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	// 取出节点，把时间更新为现在
	node, exits := cm.nodes[nodeID]
	if !exits {
		return fmt.Errorf("node %s not found", nodeID)
	}

	// 如果节点状态原本是不健康的，打印一条日志，说明已经健康
	if cm.nodes[nodeID].Status == "unhealthy" {
		log.Printf("Node %s has been restored to a healthy state", nodeID)
	}

	node.LastActive = time.Now()
	node.Status = "online"
	return nil
}

// 启动健康检查
func (cm *ClusterManager) StartHealthCheck() {
	// 每隔heartbeat发送一次检查信号
	ticker := time.NewTicker(cm.heartbeat)
	defer ticker.Stop()

	// 进入一个死循环，持续监听两个通道
	// ticker.C中有信号，则执行节点健康检查
	// stopChan有信号，则说明要停止健康检查了，直接退出循环
	for {
		select {
		case <-cm.stopChan:
			return
		case <-ticker.C:
			cm.checkNodeHealth()
		}
	}
}

// 检查所有节点健康状况，检查所有不健康的情况
func (cm *ClusterManager) checkNodeHealth() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	// 遍历所有节点，计算上次活跃到现在的时间差
	for id, node := range cm.nodes {
		// 如果时间差大于预设的超时时间，则把节点标记为离线，并且把它从节点中去除
		if now.Sub(node.LastActive) > cm.timeout {
			node.Status = "offline"
			log.Printf("Node %s is offline (last active: %v)", id, node.LastActive)
			delete(cm.nodes, id)
			// 如果只是大于超时的一半，则标记为不健康
		} else if now.Sub(node.LastActive) > cm.timeout/2 {
			node.Status = "unhealthy"
			log.Printf("Node %s is unhealthy (last active: %v)", id, node.LastActive)
		}
	}
}

// 启动http服务器，用于处理节点注册和心跳
func (cm *ClusterManager) StartHTTPServer(port string) error {
	// 创建一个http请求多路复用器mux，可以把不同请求路径路由给对应处理函数
	mux := http.NewServeMux()
	mux.HandleFunc("/register", cm.handleRegister)
	mux.HandleFunc("/heartbeat", cm.handleHeartbeat)

	// 创建一个http服务器实例，指明访问端口和处理器
	cm.httpServer = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// 创建一个tcp监听器，监听http服务器指明的端口
	listener, err := net.Listen("tcp", cm.httpServer.Addr)
	if err != nil {
		return err
	}

	log.Printf("HTTP server listening on %s", cm.httpServer.Addr)
	// 启动一个goroutine运行服务器

	if err := cm.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Printf("HTTP server error: %v", err)
	}

	return nil

}

// 停止集群管理器
func (cm *ClusterManager) Stop() {
	// 关闭通道，所有监听这个通道的goroutine都会停止执行
	close(cm.stopChan)
	if cm.httpServer != nil {
		// 创建一个带有5秒超时的上下文
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		// 在函数返回时，无论是否超时，都调用cancel取消上下文，释放资源
		defer cancel()
		// httpServer.Shutdown，优雅地关闭http服务器
		// 即等待正在进行的请求处理完毕，但一旦上下文超时，会立刻关闭服务器
		if err := cm.httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}
}

// 处理注册节点
func (cm *ClusterManager) handleRegister(w http.ResponseWriter, r *http.Request) {
	// 如果不是post方法，使用http.Error函数向客户端返回一个错误响应
	if r.Method != "POST" {
		// StatusMethodNotAllowed是405错误码
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从请求中获取数据
	id := r.FormValue("node_id")
	ip := r.FormValue("ip")
	port := r.FormValue("port")

	// 如果有参数没传，返回错误响应
	if id == "" || ip == "" || port == "" {
		http.Error(w, "Missing parameters", http.StatusBadRequest)
		return
	}

	// 调用注册节点函数
	if err := cm.RegisterNode(id, ip, port); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 把响应码设置为201，成功创建
	w.WriteHeader(http.StatusCreated)
	// Fprintf向响应w中写入内容，返回给客户端
	fmt.Fprintf(w, "Node %s registered successfully", id)
}

// 处理心跳
func (cm *ClusterManager) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	// 必须是post请求
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取id
	nodeID := r.FormValue("node_id")
	if nodeID == "" {
		http.Error(w, "Missing node ID", http.StatusBadRequest)
		return
	}

	// 更新节点的心跳时间为现在，并且状态设置为健康
	if err := cm.UpdateHeartbeat(nodeID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// 返回一个成功的响应
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Heartbeat for node %s updated", nodeID)
}

// 获取所有节点的状态
func (cm *ClusterManager) GetNodes() map[string]*Node {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	nodesCopy := make(map[string]*Node)
	for id, node := range cm.nodes {
		nodesCopy[id] = &Node{
			NodeID:     node.NodeID,
			IP:         node.IP,
			Port:       node.Port,
			LastActive: node.LastActive,
			Status:     node.Status,
		}
	}
	return nodesCopy
}

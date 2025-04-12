package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"lightScheduler/cluster"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	pb "lightScheduler/schedule" // 替换为你的包路径

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TaskWaitQueue 基于Channel的任务队列
type TaskWaitQueue struct {
	queue     chan *Task
	closeOnce sync.Once
	closed    chan struct{}
}

// NewTaskWaitQueue 创建新队列
func NewTaskWaitQueue(size int) *TaskWaitQueue {
	return &TaskWaitQueue{
		queue:  make(chan *Task, size),
		closed: make(chan struct{}),
	}
}

// Enqueue 添加任务
func (q *TaskWaitQueue) Enqueue(req *Task) error {
	select {
	case q.queue <- req:
		return nil
	case <-q.closed:
		return errors.New("queue closed")
	default:
		return errors.New("queue full")
	}
}

// Dequeue 获取任务
func (q *TaskWaitQueue) Dequeue() (*Task, error) {
	select {
	case req := <-q.queue:
		return req, nil
	case <-q.closed:
		return nil, errors.New("queue closed")
	}
}

// Close 关闭队列
func (q *TaskWaitQueue) Close() {
	q.closeOnce.Do(func() {
		close(q.closed)
		close(q.queue)
	})
}

// 启动任务接受服务器
func (q *TaskWaitQueue) StartTaskHTTPServer(port string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/inference", q.addToWaitQueue)
	mux.HandleFunc("/health", q.handleHealth)

	http_server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	listener, err := net.Listen("tcp", http_server.Addr)
	if err != nil {
		return err
	}

	log.Printf("Inference HTTP server listening on %s", http_server.Addr)

	err = http_server.Serve(listener)
	if err != nil {
		log.Printf("Inference HTTP server error: %s", err)
	}

	return nil
}

// 把任务添加到等待队列
func (q *TaskWaitQueue) addToWaitQueue(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		// StatusMethodNotAllowed是405错误码
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// 1. 定义请求结构体
	type RequestBody struct {
		ModelName    string `json:"model_name"`
		OriginPrompt string `json:"origin_prompt"`
	}

	// 2. 解析请求体
	var reqBody RequestBody
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 3. 验证必填字段
	if reqBody.ModelName == "" {
		http.Error(w, "model_name and prompt are required", http.StatusBadRequest)
		return
	}

	// 从请求体中取出值，构造任务
	modelName := reqBody.ModelName
	origin_prompt := reqBody.OriginPrompt

	new_task := &Task{
		ModelName:    modelName,
		OriginPrompt: origin_prompt,
	}

	if err := q.Enqueue(new_task); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	log.Printf("等待队列中的任务数：%d", len(q.queue))

}

// 健康测试，实际上会返回当前等待队列中的任务数量
func (q *TaskWaitQueue) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "等待队列中的任务数：%d", len(q.queue))
}

// 持续不断取出等待队列中的元素
func (q *TaskWaitQueue) HandleQueue(cm *cluster.ClusterManager) {
	for {
		select {
		case task := <-q.queue:
			// 把任务调度到合适的节点上
			log.Printf("任务已加入：%s", task.ModelName)
			sechedule(task, cm)
		case <-q.closed:
			fmt.Println("Processor stopped by close signal")
			return
		}
	}
}

func sechedule(task *Task, cm *cluster.ClusterManager) {
	// 先获取任务中模型的显存需求
	model_info := ModelsInfo[task.ModelName]
	require_mem_MB := model_info.size_GB * 1024

	// TODO,遍历cm的节点列表，选择一个合适的节点调度
	var target_node *cluster.Node = nil
	for _, node := range cm.GetNodes() {
		var node_availabe_mem_MB uint64
		node_availabe_mem_MB = 0
		for _, gpu := range node.GPUs {
			node_availabe_mem_MB += gpu.FreeMemoryMB
		}
		if node_availabe_mem_MB >= require_mem_MB {
			target_node = node
			break
		}
	}

	if target_node == nil {
		log.Fatalf("没有找到合适的节点调度任务")
		return
	}

	url := target_node.IP + ":10000"

	// grpc通信服务器的地址
	conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := pb.NewScheduleServiceClient(conn)

	// 增加超时时间到 30 秒，启动一个容器是比较耗时的
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 发送请求
	r, err := c.ProcessMessage(ctx, &pb.ScheduleRequest{
		ModelName:    task.ModelName,
		OriginPrompt: task.OriginPrompt,
	})

	if err != nil {
		log.Fatalf("rpc请求创建容器失败: %v", err)
	}

	if r.Success {
		fmt.Printf("访问端口是: %s \n", r.Port)
		fmt.Printf("响应内容: %s", r.Message)

	} else {
		log.Printf("处理失败: %s", r.Message)
	}

}

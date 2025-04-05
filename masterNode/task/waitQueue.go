package task

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
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
func (q *TaskWaitQueue) HandleQueue() {
	for {
		select {
		case task := <-q.queue:
			log.Printf("任务%s已经调度到节点1上", task.ModelName)
		case <-q.closed:
			fmt.Println("Processor stopped by close signal")
			return
		}
	}
}

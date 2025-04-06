package worker

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Worker struct {
	config     *Config
	httpClient *http.Client
	stopChan   chan struct{}
	wg         sync.WaitGroup
	registered bool
	mu         sync.Mutex
}

// 创建新的工作节点
func NewWorker(config *Config) *Worker {
	return &Worker{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		stopChan: make(chan struct{}),
	}
}

// Start 启动客户端
func (w *Worker) Start() error {
	// 先注册节点
	if err := w.register(); err != nil {
		return fmt.Errorf("注册失败: %v", err)
	}

	// 启动心跳协程
	// w.wg.Add(1)
	go w.heartbeat()
	log.Printf("节点客户端已启动，ID: %s", w.config.NodeID)

	return nil
}

// 停止工作节点
func (w *Worker) Stop() {
	close(w.stopChan)
	w.wg.Wait()
	log.Println("节点客户端已停止")
}

// 注册节点
func (w *Worker) register() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	params := url.Values{}
	params.Add("node_id", w.config.NodeID)
	params.Add("ip", w.config.IP)
	params.Add("port", w.config.Port)
	// 将参数编码到URL路径中
	url := fmt.Sprintf("%s/register?%s",
		w.config.ServerURL,
		params.Encode())

	// 发送空body的POST请求，指定请求体的内容类型为json格式，请求体为nil
	// 一直阻塞直到响应回来
	resp, err := w.httpClient.Post(url, "application/json", nil)
	if err != nil {
		return err
	}
	// 函数返回时关闭响应体，避免资源泄露
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("注册失败，状态码: %d", resp.StatusCode)
	}

	// 修改节点为已注册
	w.registered = true
	log.Println("节点注册成功")
	return nil
}

// 持续发送心跳
func (w *Worker) heartbeat() {
	// 返回前通知wg，该goroutine已完成
	// defer w.wg.Done()
	// 设置一个定时器，每隔一段时间向ticker.C通道发送信号
	ticker := time.NewTicker(w.config.Interval)
	defer ticker.Stop()

	// 死循环持续监听事件
	for {
		select {
		// ticker发来信号，则发送心跳
		case <-ticker.C:
			if err := w.sendHeartbeat(); err != nil {
				log.Printf("心跳失败: %v", err)

				// 如果是因为未注册，尝试重新注册
				if errors.Is(err, ErrNotRegistered) {
					if regErr := w.register(); regErr != nil {
						log.Printf("重新注册失败: %v", regErr)
					}
				}
			}
			// 接收到停止信号则停止监听
		case <-w.stopChan:
			return
		}
	}
}

// 节点未注册的错误定义
var (
	ErrNotRegistered = errors.New("节点未注册")
)

// 发送一次心跳
func (w *Worker) sendHeartbeat() error {
	// 上锁
	w.mu.Lock()
	defer w.mu.Unlock()

	// 如果未注册，返回错误
	if !w.registered {
		return ErrNotRegistered
	}

	// 只发送节点ID
	// 构造查询参数
	params := url.Values{}
	params.Add("node_id", w.config.NodeID)
	url := fmt.Sprintf("%s/heartbeat?%s", w.config.ServerURL, params.Encode())
	// 发送请求
	resp, err := w.httpClient.Post(url, "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 如果节点不存在，标记为未注册状态
		if resp.StatusCode == http.StatusNotFound {
			w.registered = false
			return ErrNotRegistered
		}
		return fmt.Errorf("心跳请求失败，状态码: %d", resp.StatusCode)
	}

	log.Println("心跳成功")
	return nil
}

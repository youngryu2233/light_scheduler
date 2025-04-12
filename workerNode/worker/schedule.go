package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"
	pb "workerNode/schedule"

	"workerNode/container"

	"google.golang.org/grpc"
)

// 开启调度服务器
func (worker *Worker) StartScheduler(port string) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("工作节点调度服务器failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterScheduleServiceServer(s, &server{})

	log.Println("调度Server started on port " + port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

type server struct {
	pb.UnimplementedScheduleServiceServer
}

func (s *server) ProcessMessage(ctx context.Context, req *pb.ScheduleRequest) (*pb.ScheduleResponse, error) {

	// 获取请求中的模型名和提示词
	model_name := req.GetModelName()
	origin_prompt := req.GetOriginPrompt()
	fmt.Printf("模型名是: %s，原生提示词是: %s \n", model_name, origin_prompt)

	// 启动容器
	host_port, err := StartContainerInstance(model_name)
	if err != nil {
		// 构造失败响应
		success := false
		message := err.Error()
		port := host_port

		return &pb.ScheduleResponse{
			Success: success,
			Port:    port,
			Message: message,
		}, nil
	}

	// TODO 等待容器加载完毕，服务就绪
	// 定义请求的 URL
	url_ready := "http://localhost:" + host_port + "/health"
	// 定义重试间隔时间（这里设置为 2 秒）
	retryInterval := 2 * time.Second

	for {
		// 创建一个新的 HTTP GET 请求
		resp_ready, err := http.Get(url_ready)
		if err != nil {
			fmt.Printf("发送请求出错: %v，将在 %v 后重试\n", err, retryInterval)
			time.Sleep(retryInterval)
			continue
		}
		// 确保在函数返回时关闭响应体
		defer resp_ready.Body.Close()

		// 读取响应体的内容
		body, err := io.ReadAll(resp_ready.Body)
		if err != nil {
			fmt.Printf("读取响应体出错: %v，将在 %v 后重试\n", err, retryInterval)
			time.Sleep(retryInterval)
			continue
		}

		// 定义一个 HealthResponse 结构体实例用于解析响应体
		type HealthResponse struct {
			Status string `json:"status"`
			Device string `json:"device"`
		}
		var healthResp HealthResponse
		// 将响应体解析到结构体中
		err = json.Unmarshal(body, &healthResp)
		if err != nil {
			fmt.Printf("解析响应体出错: %v，将在 %v 后重试\n", err, retryInterval)
			time.Sleep(retryInterval)
			continue
		}
		// 打印解析后的结果
		fmt.Printf("容器已经就绪，可以开始访问")
		break
	}

	// 把初始提示词询问容器，返回响应
	url := "http://localhost:" + host_port
	data := map[string]string{
		"prompt": origin_prompt,
	}
	// 将数据编码为 JSON 格式
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("JSON 编码出错:", err)
	}
	// 创建一个新的 HTTP POST 请求
	req_infer, err := http.NewRequest("POST", url+"/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("创建请求出错:", err)
	}
	// 设置请求头，指定内容类型为 application/json
	req_infer.Header.Set("Content-Type", "application/json")
	// 创建一个 HTTP 客户端
	client := &http.Client{}
	// 发送请求
	resp, err := client.Do(req_infer)
	if err != nil {
		fmt.Println("发送请求出错:", err)
	}
	// 确保在函数返回时关闭响应体
	defer resp.Body.Close()
	// 读取响应体的内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应体出错:", err)
	}
	// 定义一个结构体用于解析响应体
	type Response struct {
		Result string `json:"result"`
	}
	var response Response
	// 将响应体解析到结构体中
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("解析响应体出错:", err)
	}
	generate_result := response.Result

	// 构造成功响应，把响应返回
	success := true
	message := generate_result
	port := host_port

	return &pb.ScheduleResponse{
		Success: success,
		Port:    port,
		Message: message,
	}, nil

}

// 在本节点上启动一个容器推理实例
func StartContainerInstance(model_name string) (string, error) {
	os.Setenv("DOCKER_API_VERSION", "1.43")
	host_port, err := container.StartModelContainer(model_name)
	if err != nil {
		return "", err
	} else {
		fmt.Printf("请访问端口和模型对话：%s", host_port)
	}
	return host_port, nil
}

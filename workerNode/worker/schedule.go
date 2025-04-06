package worker

import (
	"context"
	"fmt"
	"log"
	"net"
	pb "workerNode/schedule"

	"google.golang.org/grpc"
)

// 开启调度服务器
func (worker *Worker) StartScheduler(port string) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
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
	fmt.Printf("模型名是%s，原生提示词是%s", model_name, origin_prompt)

	// TODO 调度启动容器

	success := true
	message := "已经调度成功"
	port := "30000"

	return &pb.ScheduleResponse{
		Success: success,
		Port:    port,
		Message: message,
	}, nil

}

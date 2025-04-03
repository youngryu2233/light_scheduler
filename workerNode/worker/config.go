package worker

import "time"

// Config 客户端配置
type Config struct {
	NodeID    string        `json:"node_id"`    // 节点ID
	IP        string        `json:"ip"`         // 节点IP
	Port      string        `json:"port"`       // 节点端口
	ServerURL string        `json:"server_url"` // 服务端地址
	Interval  time.Duration `json:"interval"`   // 心跳间隔
	Timeout   time.Duration `json:"timeout"`    // 请求超时时间
}

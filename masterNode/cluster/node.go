package cluster

import (
	"time"
)

// Node 表示集群中的一个节点
type Node struct {
	NodeID     string
	IP         string
	Port       string
	LastActive time.Time
	Status     string // "online", "offline", "unhealthy"
}

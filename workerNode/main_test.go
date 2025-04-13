package main

import (
	"testing"
	"workerNode/container"
)

// 定义测试函数，函数名必须以Test开头
func Test(t *testing.T) {
	// container.StartModelContainer("llama3-8b")
	container.DeleteContainer("tsif")
}

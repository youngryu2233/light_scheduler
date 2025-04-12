package task

// 字典，用于查询模型中的信息
var ModelsInfo = map[string]ModelInfo{
	"lamma3-8b": {
		size_GB: 16,
	},

	"gpt": {
		size_GB: 0,
	},
}

type ModelInfo struct {
	size_GB uint64
}

// TODO，增加一些增删改查的接口，用于操作 ModelInfo

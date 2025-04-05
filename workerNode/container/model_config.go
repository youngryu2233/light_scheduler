package container

type ModelConfig struct {
	ImageName    string
	ModelPath    string
	PortMapping  string
	EnvVars      map[string]string
	VolumeMounts map[string]string
	Command      []string
}

var modelConfigs = map[string]ModelConfig{
	"llama3-8b": {
		ImageName:    "model:v1",
		ModelPath:    "/models/Meta-Llama-3-8B",
		PortMapping:  "8000:8000",
		EnvVars:      map[string]string{"MODEL_NAME": "/models/Meta-Llama-3-8B"},
		VolumeMounts: map[string]string{"/root/Models": "/models"},
		Command:      []string{"python", "/app/server.py"},
	},
	"gpt-neo-2.7b": {
		// 其他模型配置
	},
}

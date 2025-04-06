package task

type Task struct {
	TaskID       string `json:"task_id"`
	ModelName    string `json:"model_name"`
	OriginPrompt string `json:"origin_prompt"`
	NodeIP       string `json:"node_ip"`
	Port         string `json:"port"`
	Status       string `json:"status"`
}

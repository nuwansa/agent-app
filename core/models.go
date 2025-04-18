package core

type AgentMeta struct {
	Name          string            `json:"name,required" polycode:"id"`
	Description   string            `json:"description,required"`
	SystemContext string            `json:"context"`
	Tools         []ToolDescriptor  `json:"tools"`
	Agents        []AgentDescriptor `json:"agents"`
}

type ToolResult struct {
	ToolName string
	Output   string
}

type AgentResult struct {
	AgentName string
	Output    string
}

type LatestTask struct {
	Id         string `json:"id" polycode:"id"`
	TaskId     int64  `json:"taskId"`
	LastTaskId int64  `json:"lastTaskId"`
}

type AgentInstallRequest struct {
	Name          string            `json:"name,required"`
	Description   string            `json:"description,required"`
	SystemContext string            `json:"context"`
	Tools         []ToolDescriptor  `json:"tools"`
	Agents        []AgentDescriptor `json:"agents"`
}

type AgentInput struct {
	Name  string   `json:"name,required"`
	Input LLMInput `json:"input"`
}

type AgentOutput struct {
	Output LLMOutput `json:"output"`
}

type EmptyResponse struct {
}

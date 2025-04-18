package core

import tools2 "polycode/agent-app/tools"

var registry *ToolRegistry = nil

type ToolRegistry struct {
	tools map[string]ToolExecutor
}

func GetToolRegistry() *ToolRegistry {
	if registry == nil {
		tools := make(map[string]ToolExecutor)
		registerInbuiltTools(tools)
		registry = &ToolRegistry{
			tools: tools,
		}
	}
	return registry
}

func registerInbuiltTools(tools map[string]ToolExecutor) {
	executor, err := NewInbuiltTooExecutor("get_current_time", "get current time", tools2.GetCurrentTime)
	if err != nil {
		panic(err)
	}
	tools[executor.GetName()] = executor

	executor, err = NewInbuiltTooExecutor("get_weather", "get current weather", tools2.GetWeather)
	if err != nil {
		panic(err)
	}
	tools[executor.GetName()] = executor

	executor, err = NewInbuiltTooExecutor("get_latest_news", "get latest news", tools2.GetLatestNews)
	if err != nil {
		panic(err)
	}
	tools[executor.GetName()] = executor
}

func (tr *ToolRegistry) RegisterTool(name string, executor ToolExecutor) {
	tr.tools[name] = executor
}

func (tr *ToolRegistry) GetTool(name string) ToolExecutor {
	return tr.tools[name]
}

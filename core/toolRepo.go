package core

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
)

type ToolExecutor interface {
	GetName() string
	GetDescription() string
	Execute(ctx context.Context, input string) (string, error)
	GetToolDescriptor() ToolDescriptor
}

type AgentExecutor interface {
	GetName() string
	GetDescription() string
	GetAgentDescriptor() AgentDescriptor
	Execute(ctx context.Context, name string, taskHistory *TaskHistory, input LLMInput) (LLMOutput, error)
}

func NewToolRepo(registry *ToolRegistry, agentHandler func(ctx context.Context, name string, taskHistory *TaskHistory, Input LLMInput) (LLMOutput, error)) *ToolRepo {
	return &ToolRepo{
		registry:     registry,
		tools:        make(map[string]ToolExecutor),
		agents:       make(map[string]AgentExecutor),
		agentHandler: agentHandler,
	}
}

type ToolRepo struct {
	registry     *ToolRegistry
	tools        map[string]ToolExecutor
	agents       map[string]AgentExecutor
	agentHandler func(ctx context.Context, name string, taskHistory *TaskHistory, Input LLMInput) (LLMOutput, error)
}

func (repo *ToolRepo) RegisterAgent(desc AgentDescriptor) error {
	repo.agents[desc.Name] = &AgentExecutorImpl{
		Desc:    desc,
		Handler: repo.agentHandler,
	}
	return nil
}

func (repo *ToolRepo) RegisterTool(desc ToolDescriptor) error {
	if desc.Inbuilt {
		tool := repo.registry.GetTool(desc.Name)
		if tool == nil {
			return fmt.Errorf("tool %s not found", desc.Name)
		}
		repo.tools[tool.GetName()] = tool
	} else {
		repo.tools[desc.Name] = NewRemoteToolExecutor(desc)
	}
	return nil
}

func (repo *ToolRepo) ListToolDescriptors() []ToolDescriptor {
	var list []ToolDescriptor
	for _, item := range repo.tools {
		list = append(list, item.GetToolDescriptor())
	}
	return list
}

func (repo *ToolRepo) ListAgentDescriptors() []AgentDescriptor {
	var list []AgentDescriptor
	for _, item := range repo.agents {
		list = append(list, item.GetAgentDescriptor())
	}
	return list
}

func (repo *ToolRepo) GetTool(name string) ToolExecutor {
	return repo.tools[name]
}

func (repo *ToolRepo) GetAgent(name string) AgentExecutor {
	return repo.agents[name]
}

func NewRemoteToolExecutor(desc ToolDescriptor) ToolExecutor {
	return &RemoteToolExecutor{Descriptor: desc}
}

type RemoteToolExecutor struct {
	Descriptor ToolDescriptor
}

func (r *RemoteToolExecutor) GetName() string {
	return r.Descriptor.Name
}

func (r *RemoteToolExecutor) GetDescription() string {
	return r.Descriptor.Description
}

func (r *RemoteToolExecutor) Execute(ctx context.Context, input string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (r *RemoteToolExecutor) GetToolDescriptor() ToolDescriptor {
	return r.Descriptor
}

func NewInbuiltTooExecutor(name string, description string, handler any) (ToolExecutor, error) {
	handlerValue := reflect.ValueOf(handler)
	handlerType := handlerValue.Type()

	if handlerType.Kind() != reflect.Func {
		return nil, fmt.Errorf("handler is not a function")
	}
	numIn := handlerType.NumIn()
	if numIn != 2 {
		return nil, fmt.Errorf("handler function must have  two parameters")
	}

	numOut := handlerType.NumOut()
	if numOut != 2 {
		return nil, fmt.Errorf("handler function must have two return values")
	}
	// Get the input parameter type
	inputType := handlerType.In(1)

	// Create a pointer to the input type
	inputPtr := reflect.New(inputType)

	schema, _, err := GetSchema(inputPtr.Interface())
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	executor := &InbuiltToolExecutor{
		toolDescriptor: ToolDescriptor{
			Name:        name,
			Description: description,
			Parameters:  json.RawMessage(b),
			Inbuilt:     true,
		},
		inputType: inputType,
		handler:   handlerValue,
	}

	return executor, nil
}

type InbuiltToolExecutor struct {
	toolDescriptor ToolDescriptor
	inputType      reflect.Type
	handler        reflect.Value
}

func (i *InbuiltToolExecutor) GetName() string {
	return i.toolDescriptor.Name
}

func (i *InbuiltToolExecutor) GetDescription() string {
	return i.toolDescriptor.Description
}

func (i *InbuiltToolExecutor) GetToolDescriptor() ToolDescriptor {
	return i.toolDescriptor
}

func (i *InbuiltToolExecutor) Execute(ctx context.Context, input string) (string, error) {
	// Create a pointer to the input type
	inputPtr := reflect.New(i.inputType)

	// Unmarshal the JSON input into the inputPtr
	err := json.Unmarshal([]byte(input), inputPtr.Interface())
	if err != nil {
		//return nil, fmt.Errorf("failed to unmarshal JSON input: %w", err)
		return "", fmt.Errorf("failed to unmarshal JSON input: %w", err)
	}

	// Prepare the arguments
	ctxValue := reflect.ValueOf(ctx)
	inputValue := inputPtr.Elem()

	args := []reflect.Value{ctxValue, inputValue}

	// Call the handler function
	results := i.handler.Call(args)

	// Handle return values
	if len(results) != 2 {
		return "", fmt.Errorf("handler function must return two values")
	}

	result := results[0].Interface()
	errInterface := results[1].Interface()

	if errInterface != nil {
		err, ok := errInterface.(error)
		if !ok {
			return "", fmt.Errorf("handler function's second return value is not an error")
		}
		return "error :" + err.Error(), nil
		//return nil, err
	}
	if result == nil {
		return "", nil
	} else {
		b, err := json.Marshal(result)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}

type AgentExecutorImpl struct {
	Desc    AgentDescriptor
	Handler func(ctx context.Context, name string, taskHistory *TaskHistory, Input LLMInput) (LLMOutput, error)
}

func (a *AgentExecutorImpl) GetAgentDescriptor() AgentDescriptor {
	return a.Desc
}

func (a *AgentExecutorImpl) GetName() string {
	return a.Desc.Name
}

func (a *AgentExecutorImpl) GetDescription() string {
	return a.Desc.Description
}

func (a *AgentExecutorImpl) Execute(ctx context.Context, name string, taskHistory *TaskHistory, input LLMInput) (LLMOutput, error) {
	return a.Handler(ctx, name, taskHistory, input)
}

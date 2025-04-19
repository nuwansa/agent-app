package agent_service

import (
	"context"
	"fmt"
	"github.com/cloudimpl/next-coder-sdk/polycode"
	"polycode/agent-app/core"
	"polycode/agent-app/gemini"
)

var agents = make(map[string]*core.Agent)
var llm core.LLM

func init() {
	llm2, err := gemini.NewGemini("AIzaSyAvS5tLK4qoljjBn2l-oxD9CVGxwn4ew0I", "gemini-2.0-flash-exp")
	if err != nil {
		panic(err)
	}
	llm = llm2
}

func InstallAgent(ctx polycode.ServiceContext, req core.AgentInstallRequest) (core.EmptyResponse, error) {
	meta := core.AgentMeta{
		Name:          req.Name,
		Description:   req.Description,
		SystemContext: req.SystemContext,
		Tools:         req.Tools,
		Agents:        req.Agents,
	}
	collection := ctx.Db().Collection("agent")
	err := collection.InsertOne(meta)
	return core.EmptyResponse{}, err
}

func UpdateAgent(ctx polycode.ServiceContext, req core.AgentInstallRequest) (core.EmptyResponse, error) {
	meta := core.AgentMeta{
		Name:          req.Name,
		Description:   req.Description,
		SystemContext: req.SystemContext,
		Tools:         req.Tools,
		Agents:        req.Agents,
	}
	collection := ctx.Db().Collection("agent")
	err := collection.UpdateOne(meta)
	return core.EmptyResponse{}, err
}

func CallAgent(ctx polycode.WorkflowContext, req core.AgentInput) (core.AgentOutput, error) {
	if req.Input.SessionKey == "" {
		return core.AgentOutput{}, fmt.Errorf("session key required")
	}
	ds := ctx.UnsafeDb().WithPartitionKey(req.Input.SessionKey).Get()
	agent, err := getAgent(ctx, ds, req.Name)
	if err != nil {
		return core.AgentOutput{}, err
	}

	collection := ds.Collection(agent.Name)
	history, err := loadLastTask(ds)
	out, err := agent.Run(ctx, history, req.Input)
	if err != nil {
		return core.AgentOutput{}, err
	}
	err = collection.UpsertOne(history)
	if err != nil {
		return core.AgentOutput{}, err
	}
	latestTask := core.LatestTask{Id: "xxx"}
	if history.Status == "completed" {
		latestTask.LastTaskId = history.TaskId
		latestTask.TaskId = 0
	} else {
		latestTask.TaskId = history.TaskId
		if history.GetPreviousTask() != nil {
			latestTask.LastTaskId = history.GetPreviousTask().TaskId
		}
	}
	latestColl := ds.Collection("agent:latest")
	err = latestColl.UpsertOne(latestTask)
	if err != nil {
		return core.AgentOutput{}, err
	}
	return core.AgentOutput{
		Output: out,
	}, nil
}

func callAgent(ds polycode.UnsafeDataStore, ctx context.Context, history *core.TaskHistory, req core.AgentInput) (core.AgentOutput, error) {
	agent, err := getAgent(ctx, ds, req.Name)
	if err != nil {
		return core.AgentOutput{}, err
	}

	out, err := agent.Run(ctx, history, req.Input)
	if err != nil {
		return core.AgentOutput{}, err
	}
	return core.AgentOutput{
		Output: out,
	}, nil
}

func getAgent(ctx context.Context, ds polycode.UnsafeDataStore, name string) (*core.Agent, error) {
	agent := agents[name]
	if agent == nil {
		collection := ds.Collection("agent:session")
		agentDesc := core.AgentMeta{}
		exist, err := collection.GetOne(name, &agentDesc)
		if err != nil {
			return nil, err
		}
		if !exist {
			return nil, fmt.Errorf("agent %s not found", name)
		}

		agent, err = core.NewAgent(name, agentDesc.Description, agentDesc.SystemContext, llm, agentDesc.Tools, agentDesc.Agents, func(ctx context.Context, name string, taskHistory *core.TaskHistory, Input core.LLMInput) (core.LLMOutput, error) {
			out, err := callAgent(ds, ctx, taskHistory, core.AgentInput{
				Name:  name,
				Input: Input,
			})
			if err != nil {
				return core.LLMOutput{}, err
			}
			return out.Output, nil
		})

		if err != nil {
			return nil, err
		}
		agents[name] = agent
	}
	return agent, nil

}

func loadLastTask(ds polycode.UnsafeDataStore) (*core.TaskHistory, error) {

	collection := ds.Collection("agent:latest")
	latestTask := core.LatestTask{}
	exist, err := collection.GetOne("xxx", &latestTask)
	if err != nil {
		return nil, err
	}
	taskHistory := core.NewTaskHistory()
	if exist {
		if latestTask.LastTaskId != 0 {
			taskCollection := ds.Collection("agent:session")
			previousTask := core.NewTaskHistory()
			exist, err := taskCollection.GetOne(fmt.Sprintf("%019d", latestTask.LastTaskId), previousTask)
			if err != nil {
				return nil, err
			}
			if !exist {
				taskHistory.SetPreviousTask(nil)
			} else {
				taskHistory.SetPreviousTask(previousTask)
			}
		}
		if latestTask.TaskId != 0 {
			taskCollection := ds.Collection("agent:session")
			exist, err := taskCollection.GetOne(fmt.Sprintf("%019d", latestTask.TaskId), taskHistory)
			if err != nil {
				return nil, err
			}
			if exist {
				return taskHistory, nil
			}
		}
	}

	taskHistory.TaskId = latestTask.TaskId + 1
	taskHistory.Id = fmt.Sprintf("%019d", taskHistory.TaskId)
	taskHistory.Status = "in_progress"
	return taskHistory, nil
}

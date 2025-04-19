package agent_service

import (
	"context"
	"fmt"
	"github.com/cloudimpl/next-coder-sdk/polycode"
	"polycode/agent-app/core"
	"polycode/agent-app/gemini"
)

var agent *core.Agent = nil
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
		Id:            "xxx",
		Name:          req.Name,
		Description:   req.Description,
		SystemContext: req.SystemContext,
		Tools:         req.Tools,
		Agents:        req.Agents,
	}
	collection := ctx.Db().Collection("agent")
	err := collection.UpsertOne(meta)
	return core.EmptyResponse{}, err
}

func UpdateAgent(ctx polycode.ServiceContext, req core.AgentInstallRequest) (core.EmptyResponse, error) {
	meta := core.AgentMeta{
		Id:            "xxx",
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
	history, err := loadLastTask(ds, req.Input.TaskId)
	out, err := agent.Run(ctx, history, req.Input)
	if err != nil {
		return core.AgentOutput{}, err
	}
	err = collection.UpsertOne(history)
	if err != nil {
		return core.AgentOutput{}, err
	}
	if req.Input.TaskId == 0 {
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
	}

	return core.AgentOutput{
		Output: out,
	}, nil
}

func callAgent(ctx polycode.WorkflowContext, req core.AgentInput) (core.AgentOutput, error) {

	resp := ctx.Service(req.Name).WithPartitionKey(req.Input.SessionKey).Get().RequestReply(polycode.TaskOptions{}, "CallAgent", req)
	output := core.AgentOutput{}
	err := resp.Get(&output)
	if err != nil {
		return core.AgentOutput{}, err
	}
	return output, nil
}

func getAgent(wrkCtx polycode.WorkflowContext, ds polycode.UnsafeDataStore, name string) (*core.Agent, error) {
	if agent == nil {
		collection := ds.Collection("agent:session")
		agentDesc := core.AgentMeta{}
		exist, err := collection.GetOne("xxx", &agentDesc)
		if err != nil {
			return nil, err
		}
		if !exist {
			return nil, fmt.Errorf("agent %s not found", name)
		}

		agent, err = core.NewAgent(agentDesc.Name, agentDesc.Description, agentDesc.SystemContext, llm, agentDesc.Tools, agentDesc.Agents, func(ctx context.Context, name string, Input core.LLMInput) (core.LLMOutput, error) {
			out, err := callAgent(wrkCtx, core.AgentInput{
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
	}
	return agent, nil

}

func loadTaskById(ds polycode.UnsafeDataStore, taskId int64) (*core.TaskHistory, error) {

	taskCollection := ds.Collection("agent:session")
	taskHistory := core.NewTaskHistory()
	exist, err := taskCollection.GetOne(fmt.Sprintf("%019d", taskId), taskHistory)
	if err != nil {
		return nil, err
	}
	if !exist {
		taskHistory.TaskId = taskId
		taskHistory.Id = fmt.Sprintf("%019d", taskHistory.TaskId)
		taskHistory.Status = "in_progress"
	}
	return taskHistory, nil
}

func loadLastTask(ds polycode.UnsafeDataStore, taskId int64) (*core.TaskHistory, error) {
	if taskId != 0 {
		return loadTaskById(ds, taskId)
	}
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

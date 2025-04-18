package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

var systemAgentContext = `
You are an AI assistant designed to process user input and maintain task-specific chat history. Your goal is to provide appropriate responses while keeping track of the current task status.

You will receive one input variables:
<user_input>
{{USER_INPUT}}
</user_input>

agent specific system context as follows
<agent_system_context>
{{agent_system_context}}
</agent_system_context>

Follow these steps to process the input and instruction and generate the appropriate output:

1. Read and understand the user input provided in the <user_input> tags.

2. Process the user input and think step by step to formulate right response.

3. generate an appropriate response based on the thinking and current task .

4. put your thinking between the <thinking></thinking> tag

5. After generating your response, you must include a task status update. The task status should be one of the following:
   - "in_progress": If the current task is ongoing and requires further interaction.
   - "completed": If the current task has been finished and no further interaction is needed.

6. Format your output as follows:
   <response>
   [Your response to the user input goes here]
   </response>

   <task_status>[task status: either "in_progress" or "completed"]</task_status>

Here's an example of how your output should look:

<response>
Certainly! I'd be happy to help you with that. Based on the information you provided, here's what I suggest...
</response>

<task_status>in_progress</task_status>

Remember to always include both the <response> and <task_status> tags in your output. The task status will help determine when to reset the chat history for the next task.
`

func NewAgent(name string, description string, systemContext string, llm LLM, tools []ToolDescriptor, agents []AgentDescriptor, agentHandler func(ctx context.Context, name string, taskHistory *TaskHistory, Input LLMInput) (LLMOutput, error)) (*Agent, error) {
	toolRepo := NewToolRepo(GetToolRegistry(), agentHandler)

	agent := &Agent{
		Name:             name,
		Description:      description,
		SystemContext:    systemContext,
		IsConversational: false,
		LLM:              llm,
		toolRepo:         toolRepo,
	}
	for _, tool := range tools {
		err := agent.RegisterTool(tool.Name)
		if err != nil {
			return nil, err
		}
	}
	for _, a := range agents {
		err := agent.RegisterAgent(a)
		if err != nil {
			return nil, err
		}
	}
	return agent, nil
}

func NewTaskHistory() *TaskHistory {
	return &TaskHistory{AgentsHistory: make(map[string]*TaskHistory)}
}

type TaskHistory struct {
	Id            string                  `json:"id" polycode:"id"`
	TaskId        int64                   `json:"taskId"`
	Contents      []ChatContent           `json:"contents"`
	Status        string                  `json:"status"`
	Stats         Stats                   `json:"stats"`
	AgentsHistory map[string]*TaskHistory `json:"agentsHistory"`
	previousTask  *TaskHistory
}

func (th *TaskHistory) SetPreviousTask(previousTask *TaskHistory) {
	th.previousTask = previousTask
}

func (th *TaskHistory) GetPreviousTask() *TaskHistory {
	return th.previousTask
}

type Agent struct {
	Name             string
	Description      string
	SystemContext    string
	IsConversational bool
	LLM              LLM
	toolRepo         *ToolRepo
}

func (agent *Agent) GetName() string {
	return agent.Name
}

func (agent *Agent) GetDescription() string {
	return agent.Description
}

func (agent *Agent) RegisterTool(name string) error {
	return agent.toolRepo.RegisterTool(ToolDescriptor{
		Name:        name,
		Description: "",
		Parameters:  nil,
		Inbuilt:     true,
	})
}

func (agent *Agent) RegisterAgent(a AgentDescriptor) error {
	return agent.toolRepo.RegisterAgent(AgentDescriptor{
		Name:        a.Name,
		Description: a.Description,
	})
}

func (agent *Agent) Run(ctx context.Context, taskHistory *TaskHistory, input LLMInput) (LLMOutput, error) {

	systemContext := ReplaceLabels(agent.SystemContext, input.Labels)
	toolsContext := ""
	tools := agent.toolRepo.ListToolDescriptors()
	agents := agent.toolRepo.ListAgentDescriptors()
	if len(tools) > 0 {
		toolsContext = GetToolPrompt(tools, agents)
	}

	agentContext := ReplaceLabels(systemAgentContext, map[string]string{"agent_system_context": agent.SystemContext})
	systemContext = agentContext + "\n" + toolsContext
	input.Text = fmt.Sprintf("<user_input>%s</user_input>", input.Text)
	out, err := agent.run(ctx, systemContext, taskHistory, input)
	if err != nil {
		return LLMOutput{}, nil
	}

	taskHistory.Stats = out.Stats
	return out, nil
}

func (agent *Agent) run(ctx context.Context, systemContext string, taskHistory *TaskHistory, input LLMInput) (LLMOutput, error) {

	var chatContents []ChatContent
	if taskHistory.previousTask != nil {
		chatContents = append(chatContents, taskHistory.previousTask.Contents...)
	}
	chatContents = append(chatContents, taskHistory.Contents...)

	output, err := agent.LLM.Generate(ctx, systemContext, chatContents, input)
	if err != nil {
		return LLMOutput{}, err
	}
	if input.Text != "" {
		taskHistory.Contents = append(taskHistory.Contents, NewContent("user", input.Text))
	}

	toolCalls, err := ExtractToolCalls(output.Text)
	if err != nil {
		return LLMOutput{}, err
	}
	var results []ToolResult
	for _, toolCall := range toolCalls {
		println("toolCall", toolCall.ToolName, toolCall.Parameters)
		ret, err := agent.executeTool(ctx, toolCall.ToolName, toolCall.Parameters)
		var out = ""
		if err != nil {
			out = err.Error()
		} else {
			out = ret
		}
		results = append(results, ToolResult{
			ToolName: toolCall.ToolName,
			Output:   out,
		})
	}
	agentCalls, err := ExtractAgentCalls(output.Text)
	if err != nil {
		return LLMOutput{}, err
	}

	var agentResult []AgentResult
	for _, agentCall := range agentCalls {
		agentHistory := taskHistory.AgentsHistory[agentCall.AgentName]
		if agentHistory == nil {
			agentHistory = NewTaskHistory()
			taskHistory.AgentsHistory[agentCall.AgentName] = agentHistory
		}
		println("agentCall", agentCall.AgentName, agentCall.Input)
		ret, err := agent.executeAgent(ctx, agentCall.AgentName, agentHistory, LLMInput{
			SessionKey:      input.SessionKey,
			ChildSessionKey: taskHistory.Id,
			Text:            agentCall.Input,
			Image:           nil,
			Labels:          nil,
		})
		var out = ""
		if err != nil {
			out = err.Error()
		} else {
			out = ret.Text
		}
		agentResult = append(agentResult, AgentResult{
			AgentName: agentCall.AgentName,
			Output:    out,
		})
	}

	taskHistory.Contents = append(taskHistory.Contents, NewContent("assistant", output.Text))
	if len(results) > 0 {
		resultsStr, err := json.Marshal(results)
		if err != nil {
			return LLMOutput{}, err
		}
		taskHistory.Contents = append(taskHistory.Contents, NewContent("user", "<tool_result>"+string(resultsStr)+"</tool_result>"))
		return agent.run(ctx, systemContext, taskHistory, LLMInput{})
	}

	if len(agentResult) > 0 {
		resultsStr, err := json.Marshal(agentResult)
		if err != nil {
			return LLMOutput{}, err
		}
		taskHistory.Contents = append(taskHistory.Contents, NewContent("user", "<agent_result>"+string(resultsStr)+"</agent_result>"))
		return agent.run(ctx, systemContext, taskHistory, LLMInput{})
	}
	response, err := agent.extractTagContent(strings.TrimSpace(output.Text), "response")
	if err != nil {
		println("error output2 ", output.Text)
		return agent.run(ctx, systemContext, taskHistory, LLMInput{
			Text: "it look like response tag not properly completed.correct the error silently.",
		})
		//return LLMOutput{}, fmt.Errorf("error processing your request,please try again later")
	}
	status, err := agent.extractTagContent(strings.TrimSpace(output.Text), "task_status")
	if err != nil {
		println("error output3 ", output.Text)
		return LLMOutput{}, fmt.Errorf("error processing your request,please try again later")
	}
	println(fmt.Sprintf("task status %s", status))
	taskHistory.Status = status

	output.Text = response
	return output, err
}

func (agent *Agent) executeTool(ctx context.Context, name string, input map[string]any) (string, error) {
	executor := agent.toolRepo.GetTool(name)
	if executor == nil {
		return "", fmt.Errorf("tool %s not found", name)
	}
	b, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	return executor.Execute(ctx, string(b))
}

func (agent *Agent) executeAgent(ctx context.Context, name string, taskHistory *TaskHistory, input LLMInput) (LLMOutput, error) {
	executor := agent.toolRepo.GetAgent(name)
	if executor == nil {
		return LLMOutput{}, fmt.Errorf("agent %s not found", name)
	}

	return executor.Execute(ctx, name, taskHistory, input)
}

// extractTagContent extracts the inner text of the given XML tag from xmlStr.
func (agent *Agent) extractTagContent(xmlStr, tag string) (string, error) {
	//decoder := xml.NewDecoder(strings.NewReader(xmlStr))
	//for {
	//	token, err := decoder.Token()
	//	if err != nil {
	//		if err == io.EOF {
	//			break // reached the end of XML without finding the tag
	//		}
	//		return "", err
	//	}
	//	if startElem, ok := token.(xml.StartElement); ok && startElem.Name.Local == tag {
	//		var content string
	//		// Decode the element content into the variable 'content'
	//		err = decoder.DecodeElement(&content, &startElem)
	//		if err != nil {
	//			return "", err
	//		}
	//		return content, nil
	//	}
	//}
	//return "", fmt.Errorf("tag %q not found", tag)
	var results []string
	openTag := fmt.Sprintf("<%s>", tag)
	closeTag := fmt.Sprintf("</%s>", tag)

	//println(spew.Sprintf("<<<%s>>>", input))
	for {
		// Find the opening tag
		start := strings.Index(xmlStr, openTag)
		if start == -1 {
			//	println("err := opening tag not found")
			break
		}

		// Find the closing tag after the opening tag
		end := strings.Index(xmlStr[start:], closeTag)
		if end == -1 {
			//	println("err := closing tag not found")
			break
		}

		// Extract content between the tags
		content := xmlStr[start+len(openTag) : start+end]
		results = append(results, content)

		// Move the input forward to continue searching
		xmlStr = xmlStr[start+end+len(closeTag):]
	}

	return strings.Join(results, "\n"), nil
}

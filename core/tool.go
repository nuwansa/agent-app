package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/invopop/jsonschema"
	"reflect"
	"regexp"
	"strings"
)

var systemToolPrompt = `
You have access to the following tools/agents that you can use to assist users. Each tool/agent has specific capabilities and parameters that you must understand to use them correctly.
<tools>
{{tools}}
</tools>
Tools Usage Instructions
When using tools, follow these guidelines:

1.Tool Selection: Choose the most appropriate tool based on the user's request.
2.Parameter Formatting: When calling a tool, ensure all required parameters are provided in the correct format.
3.if required parameter is missing ask from the user to provide it.
4.Tool Invocation Format: Use the following format to invoke a tool:

<tools>
<tool_call>
  <tool_name>name_of_the_tool</tool_name>
  <parameters>
    {"param1": "value1", "param2": "value2"}
  </parameters>
</tool_call>
</tools>
4.Response Handling: After using a tool, incorporate the results naturally into your response.
5.Error Handling: If a tool call fails, explain the issue to the user and suggest alternative approaches.
6.Multiple Tool Calls: You can make multiple tool calls in sequence when necessary to satisfy complex requests.

<agents>
{{agents}}
</agents>
Agents Usage Instructions
When using agents , follow these guidelines
1. Determine if the task requires delegation to a specialized agent or if you can handle it directly.
2. If delegation is required, identify the most appropriate agent from the registered agents list.
3. Agent Invocation Format: Use the following format to invoke a agent:

<agents>
<agent_call>
  <agent_name>name_of_the_agent</agent_name>
  <input>
    natural  text input
  </input>
</agent_call>
</agents>
4. if you have a knowledge of answering agent question , reply back with the answer to the agent for the next step.
`

type AgentDescriptor struct {
	Name        string
	Description string
}

type AgentCall struct {
	AgentName string
	Input     string
}

type ToolDescriptor struct {
	Name        string          `json:"name"`
	ServiceName string          `json:"serviceName"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
	Inbuilt     bool            `json:"inbuilt"`
}

// ToolCall represents a parsed tool call from the content
type ToolCall struct {
	ToolName   string
	Parameters map[string]interface{}
}

//type ToolResult struct {
//	ToolName string
//	Output   string
//}

func GetToolPrompt(tools []ToolDescriptor, agents []AgentDescriptor) string {

	var toolsStr = []byte("[]")
	var err error
	if len(tools) > 0 {
		toolsStr, err = json.Marshal(tools)
		if err != nil {
			panic(err)
		}
	}
	toolPrompt := ReplaceLabels(systemToolPrompt, map[string]string{"tools": string(toolsStr)})

	var agentsStr = []byte("[]")
	if len(agents) > 0 {
		agentsStr, err = json.Marshal(agents)
		if err != nil {
			panic(err)
		}
	}
	toolPrompt = ReplaceLabels(toolPrompt, map[string]string{"agents": string(agentsStr)})
	return toolPrompt
}

func ReplaceLabels(template string, replacements map[string]string) string {
	for key, value := range replacements {
		placeholder := "{{" + key + "}}"
		template = strings.ReplaceAll(template, placeholder, value)
	}
	return template
}

var toolPattern = `<tool_call>\s*<tool_name>(.*?)</tool_name>\s*<parameters>\s*(.*?)\s*</parameters>\s*</tool_call>`
var toolRegEx = regexp.MustCompile(toolPattern)

var agentPattern = `<agent_call>\s*<agent_name>(.*?)</agent_name>\s*<input>\s*(.*?)\s*</input>\s*</agent_call>`
var agentRegEx = regexp.MustCompile(agentPattern)

// ExtractToolCalls extracts tool calls from the given content
func ExtractToolCalls(content string) ([]ToolCall, error) {
	var toolCalls []ToolCall

	// Define the regular expression toolPattern to match tool calls

	// Find all matches
	matches := toolRegEx.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) != 3 {
			continue
		}

		toolName := strings.TrimSpace(match[1])
		paramsJSON := strings.TrimSpace(match[2])

		// Parse parameters
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
			return nil, fmt.Errorf("failed to parse parameters for tool %s: %w", toolName, err)
		}

		toolCalls = append(toolCalls, ToolCall{
			ToolName:   toolName,
			Parameters: params,
		})
	}

	return toolCalls, nil
}

// ExtractAgentCalls extracts agent calls from the given content
func ExtractAgentCalls(content string) ([]AgentCall, error) {
	var agentCalls []AgentCall

	// Define the regular expression toolPattern to match tool calls

	// Find all matches
	matches := agentRegEx.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) != 3 {
			continue
		}

		agentName := strings.TrimSpace(match[1])
		input := strings.TrimSpace(match[2])

		agentCalls = append(agentCalls, AgentCall{
			AgentName: agentName,
			Input:     input,
		})
	}

	return agentCalls, nil
}

func StructToJSONSchema(v interface{}) ([]byte, error) {
	schema := jsonschema.Reflect(v)
	//
	//var buf bytes.Buffer
	//encoder := json.NewEncoder(&buf)
	////encoder.SetIndent("", "  ")
	//err := encoder.Encode(schema)
	//if err != nil {
	//	return "", err
	//}
	//
	//return buf.String(), nil
	return schema.MarshalJSON()
}

func GetSchema(obj interface{}) (interface{}, any, error) {
	var schema interface{}
	for _, v := range jsonschema.Reflect(obj).Definitions {
		schema = v
	}

	if reflect.ValueOf(obj).Kind() != reflect.Ptr {
		return nil, nil, errors.New("object must be a pointer")
	}

	pointsToValue := reflect.Indirect(reflect.ValueOf(obj))

	if pointsToValue.Kind() == reflect.Struct {
		return schema, obj, nil
	}

	if pointsToValue.Kind() == reflect.Slice {
		return nil, nil, errors.New("slice not supported as an input")
	}

	return schema, obj, nil
}

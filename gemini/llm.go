package gemini

import (
	"context"
	"google.golang.org/genai"
	"polycode/agent-app/core"
)

type Gemini struct {
	APIKey    string
	ModelName string
	client    *genai.Client
}

func (g *Gemini) StartChat(sessionId string) core.LLMChat {
	return nil
}

func NewGemini(apiKey string, modelName string) (*Gemini, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}

	return &Gemini{
		APIKey:    apiKey,
		ModelName: modelName,
		client:    client,
	}, nil
}

func (g *Gemini) Generate(ctx context.Context, systemContext string, history []core.ChatContent, input core.LLMInput) (core.LLMOutput, error) {

	inputTokenCount := int32(0)
	outputTokenCount := int32(0)
	totalTokenCount := int32(0)

	var contents []*genai.Content = nil
	for _, content := range history {
		if content.Role == "user" {
			contents = append(contents, &genai.Content{Role: "user", Parts: []*genai.Part{{Text: content.Content}}})
		} else if content.Role == "assistant" {
			contents = append(contents, &genai.Content{Role: "model", Parts: []*genai.Part{{Text: content.Content}}})
		}
	}
	if input.Text != "" {
		contents = append(contents, &genai.Content{Role: "user", Parts: []*genai.Part{{Text: input.Text}}})
	}

	var config *genai.GenerateContentConfig = nil
	if systemContext != "" {
		config =
			&genai.GenerateContentConfig{
				SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: systemContext}}},
			}
	}
	// Call the GenerateContent method.
	result, err := g.client.Models.GenerateContent(ctx,
		g.ModelName,
		contents,
		config,
	)
	if err != nil {
		return core.LLMOutput{}, err
	}

	inputTokenCount = result.UsageMetadata.PromptTokenCount

	outputTokenCount = result.UsageMetadata.CandidatesTokenCount
	
	totalTokenCount = result.UsageMetadata.TotalTokenCount

	return core.LLMOutput{Text: result.Text(), Stats: core.Stats{
		InputTokenCount:  inputTokenCount,
		OutputTokenCount: outputTokenCount,
		TotalTokenCount:  totalTokenCount,
	}}, nil
}

type GeminiChat struct {
	llm     *Gemini
	session *genai.Chat
}

func NewGeminiChat(gemini *Gemini) core.LLMChat {
	//chatSession :=
	//return &GeminiChat{llm: gemini, session: chatSession}
	return nil
}

func (g *GeminiChat) RequestReply(ctx context.Context, input core.LLMInput) (core.LLMOutput, error) {
	return core.LLMOutput{}, nil
}

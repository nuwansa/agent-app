package core

import "context"

type Image struct {
	Data string `json:"data"`
	Path string `json:"path"`
}

type LLMInput struct {
	SessionKey      string
	ChildSessionKey string
	Text            string
	Image           []Image
	Labels          map[string]string
}

type LLMOutput struct {
	Text  string
	Stats Stats
}

type Stats struct {
	InputTokenCount  int32 `json:"input_token_count,omitempty"`
	OutputTokenCount int32 `json:"output_token_count,omitempty"`
	TotalTokenCount  int32 `json:"total_token_count,omitempty"`
}

type ChatContent struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func NewContent(role string, content string) ChatContent {
	return ChatContent{
		Role:    role,
		Content: content,
	}
}

type LLM interface {
	Generate(ctx context.Context, systemContext string, history []ChatContent, input LLMInput) (LLMOutput, error)
	StartChat(sessionId string) LLMChat
}

type LLMChat interface {
	RequestReply(ctx context.Context, input LLMInput) (LLMOutput, error)
}

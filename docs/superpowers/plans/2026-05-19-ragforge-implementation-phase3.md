# RAGForge Implementation Plan - Phase 3: Engine Layer

> Continuation of Phase 2. Requires Phases 1-2 to be complete.

---

### Task 3.1: Create engine interfaces and LLM provider

**Files:**
- Create: `apps/ragforge/internal/engine/types.go`
- Create: `apps/ragforge/internal/engine/engine.go`
- Create: `apps/ragforge/internal/engine/llm/llm.go`
- Create: `apps/ragforge/internal/engine/llm/openai.go`
- Create: `apps/ragforge/internal/engine/embedding/embedding.go`
- Create: `apps/ragforge/internal/engine/embedding/openai.go`

- [ ] **Step 1: Create internal/engine/types.go**

```go
package engine

import "context"

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float32       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
	Stream      bool          `json:"stream"`
}

type ChatCompletionResponse struct {
	Content string `json:"content"`
}

type EmbeddingRequest struct {
	Model  string   `json:"model"`
	Inputs []string `json:"inputs"`
}

type EmbeddingResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

type LLMProvider interface {
	ChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error)
	ChatCompletionStream(ctx context.Context, req *ChatCompletionRequest) (<-chan string, error)
}

type EmbeddingProvider interface {
	CreateEmbedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)
}
```

- [ ] **Step 2: Create internal/engine/engine.go**

```go
package engine

type EngineFactory struct {
	llmProvider       LLMProvider
	embeddingProvider EmbeddingProvider
}

func NewEngineFactory() *EngineFactory {
	return &EngineFactory{}
}

func (f *EngineFactory) SetLLMProvider(p LLMProvider) {
	f.llmProvider = p
}

func (f *EngineFactory) SetEmbeddingProvider(p EmbeddingProvider) {
	f.embeddingProvider = p
}

func (f *EngineFactory) GetLLM() LLMProvider {
	return f.llmProvider
}

func (f *EngineFactory) GetEmbedding() EmbeddingProvider {
	return f.embeddingProvider
}
```

- [ ] **Step 3: Create internal/engine/llm/llm.go**

```go
package llm

import (
	"github.com/morehao/ark-go-starter/apps/ragforge/internal/engine"
)

type LLMType string

const (
	LLMTypeOpenAI LLMType = "openai"
)

func NewLLMProvider(llmType LLMType, apiKey, apiBase string) (engine.LLMProvider, error) {
	switch llmType {
	case LLMTypeOpenAI:
		return NewOpenAIProvider(apiKey, apiBase), nil
	}
	return nil, nil
}
```

- [ ] **Step 4: Create internal/engine/llm/openai.go**

```go
package llm

import (
	"context"

	"github.com/morehao/ark-go-starter/apps/ragforge/internal/engine"
	openai "github.com/sashabaranov/go-openai"
)

type OpenAIProvider struct {
	client *openai.Client
}

func NewOpenAIProvider(apiKey, apiBase string) *OpenAIProvider {
	cfg := openai.DefaultConfig(apiKey)
	if apiBase != "" {
		cfg.BaseURL = apiBase
	}
	return &OpenAIProvider{
		client: openai.NewClientWithConfig(cfg),
	}
}

func (p *OpenAIProvider) ChatCompletion(ctx context.Context, req *engine.ChatCompletionRequest) (*engine.ChatCompletionResponse, error) {
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return &engine.ChatCompletionResponse{}, nil
	}
	return &engine.ChatCompletionResponse{Content: resp.Choices[0].Message.Content}, nil
}

func (p *OpenAIProvider) ChatCompletionStream(ctx context.Context, req *engine.ChatCompletionRequest) (<-chan string, error) {
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	stream, err := p.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      true,
	})
	if err != nil {
		return nil, err
	}
	ch := make(chan string)
	go func() {
		defer stream.Close()
		defer close(ch)
		for {
			resp, err := stream.Recv()
			if err != nil {
				return
			}
			if len(resp.Choices) > 0 {
				ch <- resp.Choices[0].Delta.Content
			}
		}
	}()
	return ch, nil
}
```

- [ ] **Step 5: Create internal/engine/embedding/embedding.go**

```go
package embedding

import (
	"github.com/morehao/ark-go-starter/apps/ragforge/internal/engine"
)

type EmbeddingType string

const (
	EmbeddingTypeOpenAI EmbeddingType = "openai"
)

func NewEmbeddingProvider(embType EmbeddingType, apiKey, apiBase, modelName string) (engine.EmbeddingProvider, error) {
	switch embType {
	case EmbeddingTypeOpenAI:
		return NewOpenAIEmbeddingProvider(apiKey, apiBase, modelName), nil
	}
	return nil, nil
}
```

- [ ] **Step 6: Create internal/engine/embedding/openai.go**

```go
package embedding

import (
	"context"

	"github.com/morehao/ark-go-starter/apps/ragforge/internal/engine"
	openai "github.com/sashabaranov/go-openai"
)

type OpenAIEmbeddingProvider struct {
	client    *openai.Client
	modelName string
}

func NewOpenAIEmbeddingProvider(apiKey, apiBase, modelName string) *OpenAIEmbeddingProvider {
	cfg := openai.DefaultConfig(apiKey)
	if apiBase != "" {
		cfg.BaseURL = apiBase
	}
	if modelName == "" {
		modelName = openai.AdaEmbeddingV2
	}
	return &OpenAIEmbeddingProvider{
		client:    openai.NewClientWithConfig(cfg),
		modelName: modelName,
	}
}

func (p *OpenAIEmbeddingProvider) CreateEmbedding(ctx context.Context, req *engine.EmbeddingRequest) (*engine.EmbeddingResponse, error) {
	resp, err := p.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Model: openai.EmbeddingModel(p.modelName),
		Input: req.Inputs,
	})
	if err != nil {
		return nil, err
	}
	embeddings := make([][]float32, len(resp.Data))
	for i, item := range resp.Data {
		embeddings[i] = item.Embedding
	}
	return &engine.EmbeddingResponse{Embeddings: embeddings}, nil
}
```

- [ ] **Step 7: Verify compilation**

```bash
cd apps/ragforge && go build ./...
```
Expected: Build succeeds.

- [ ] **Step 8: Commit**

```bash
git add apps/ragforge/internal/engine/
git commit -m "feat(ragforge): add engine layer with LLM and embedding providers"
```

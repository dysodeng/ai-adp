package valueobject

// ModelCapability AI 模型能力类型
type ModelCapability string

const (
	ModelCapabilityLLM       ModelCapability = "llm"       // 大语言模型（对话、推理）
	ModelCapabilityEmbedding ModelCapability = "embedding" // 向量化模型
)

func (c ModelCapability) IsValid() bool {
	return c == ModelCapabilityLLM || c == ModelCapabilityEmbedding
}

func (c ModelCapability) String() string { return string(c) }

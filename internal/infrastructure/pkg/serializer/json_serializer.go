package serializer

import (
	"github.com/bytedance/sonic"
)

// JSONSerializer JSON序列化器
type JSONSerializer struct{}

func (s *JSONSerializer) Marshal(v any) (string, error) {
	data, err := sonic.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *JSONSerializer) Unmarshal(data string, v any) error {
	return sonic.Unmarshal([]byte(data), v)
}

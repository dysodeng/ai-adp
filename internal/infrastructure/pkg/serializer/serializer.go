package serializer

import "fmt"

// Serializer 序列化接口
type Serializer interface {
	Marshal(v any) (string, error)
	Unmarshal(data string, v any) error
}

// NewSerializer 根据名称创建序列化器
func NewSerializer(name string) Serializer {
	switch name {
	case "json", "":
		return &JSONSerializer{}
	case "msgpack":
		return &MsgpackSerializer{}
	default:
		panic(fmt.Sprintf("unsupported serializer: %s", name))
	}
}

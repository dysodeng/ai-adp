package serializer

import "github.com/vmihailenco/msgpack/v5"

// MsgpackSerializer msgpack序列化器
type MsgpackSerializer struct{}

func (s *MsgpackSerializer) Marshal(v any) (string, error) {
	data, err := msgpack.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *MsgpackSerializer) Unmarshal(data string, v any) error {
	return msgpack.Unmarshal([]byte(data), v)
}

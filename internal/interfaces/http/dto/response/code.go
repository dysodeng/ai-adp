package response

// Code 响应错误码
type Code int

const (
	CodeOk                  Code = 0
	CodeFail                Code = 400
	CodeInternalServerError Code = 500
)

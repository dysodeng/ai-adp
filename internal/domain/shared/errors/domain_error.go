package errors

// DomainError 领域错误基类
type DomainError struct {
	code    string
	message string
}

func New(code, message string) *DomainError {
	return &DomainError{code: code, message: message}
}

func (e *DomainError) Error() string { return e.message }
func (e *DomainError) Code() string  { return e.code }

// Is 按错误码判断
func Is(err error, code string) bool {
	de, ok := err.(*DomainError)
	if !ok {
		return false
	}
	return de.code == code
}

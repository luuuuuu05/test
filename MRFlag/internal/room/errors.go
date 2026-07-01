package room

type CodedError struct {
	Code    string
	Message string
	Detail  any
}

func (e *CodedError) Error() string {
	return e.Message
}

func coded(code, message string, detail any) *CodedError {
	return &CodedError{Code: code, Message: message, Detail: detail}
}

package handler

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func ErrorResponse(code, message string) errorPayload {
	return errorPayload{
		Code:    code,
		Message: message,
	}
}

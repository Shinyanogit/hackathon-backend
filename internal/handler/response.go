package handler

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error errorPayload `json:"error"`
}

func NewErrorResponse(code, message string) ErrorResponse {
	return ErrorResponse{
		Error: errorPayload{
			Code:    code,
			Message: message,
		},
	}
}

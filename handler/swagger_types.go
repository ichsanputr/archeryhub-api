package handler

// ErrorResponse is a standard error response for Swagger annotations.
type ErrorResponse struct {
	Error string `json:"error"`
}

// MessageResponse is a standard message response for Swagger annotations.
type MessageResponse struct {
	Message string `json:"message"`
}

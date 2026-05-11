package utils

type APIResponse struct {
	Meta Meta        `json:"meta"`
	Data interface{} `json:"data"`
}

type Meta struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Status  string `json:"status"`
}

func FormatResponse(message string, code int, status string, data interface{}) APIResponse {
	return APIResponse{
		Meta: Meta{
			Message: message,
			Code:    code,
			Status:  status,
		},
		Data: data,
	}
}

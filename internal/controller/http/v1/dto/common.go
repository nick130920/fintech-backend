package dto

// ErrorResponse representa una respuesta de error estándar
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Response representa una respuesta estándar de la API
type Response struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// SuccessResponse representa una respuesta exitosa estándar (alias de Response)
type SuccessResponse = Response

// PaginatedResponse representa una respuesta paginada
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// IDResponse representa una respuesta con solo un ID
type IDResponse struct {
	ID uint `json:"id"`
}

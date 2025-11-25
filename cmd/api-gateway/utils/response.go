package utils

import (
	"encoding/json"
	"net/http"
)

// JSONResponse returns a JSON response
func JSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// ErrorResponse returns an error response
func ErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	JSONResponse(w, statusCode, map[string]interface{}{
		"error": message,
	})
}

// SuccessResponse returns a success response
func SuccessResponse(w http.ResponseWriter, data interface{}) {
	JSONResponse(w, http.StatusOK, data)
}


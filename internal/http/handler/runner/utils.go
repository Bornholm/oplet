package runner

import (
	"encoding/json"
	"net/http"

	"github.com/bornholm/oplet/internal/slogx"
	"github.com/pkg/errors"
)

// HTTP response utilities
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// If we can't encode the response, write a simple error
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	writeJSONResponse(w, statusCode, ErrorResponse{
		Error: message,
	})
}

func writeErrorResponseWithCode(w http.ResponseWriter, statusCode int, message, code string) {
	writeJSONResponse(w, statusCode, ErrorResponse{
		Error: message,
		Code:  code,
	})
}

// Error handling utilities
func handleInternalError(h *Handler, w http.ResponseWriter, r *http.Request, err error, message string) {
	ctx := r.Context()
	h.logger.ErrorContext(ctx, message, slogx.Error(errors.WithStack(err)))
	writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
}

func handleValidationError(w http.ResponseWriter, err error) {
	writeErrorResponseWithCode(w, http.StatusBadRequest, err.Error(), "validation_error")
}

func handleNotFoundError(w http.ResponseWriter, resource string) {
	writeErrorResponseWithCode(w, http.StatusNotFound, resource+" not found", "not_found")
}

// Request parsing utilities
func parseJSONRequest(r *http.Request, dest interface{}) error {
	if r.Header.Get("Content-Type") != "application/json" {
		return ErrInvalidRequest("Content-Type must be application/json")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dest); err != nil {
		return ErrInvalidRequest("invalid JSON: %v", err)
	}

	return nil
}

// Path parameter utilities
func getTaskIDFromPath(r *http.Request) (string, error) {
	taskID := r.PathValue("taskID")
	if taskID == "" {
		return "", ErrInvalidRequest("taskID is required")
	}
	return taskID, nil
}

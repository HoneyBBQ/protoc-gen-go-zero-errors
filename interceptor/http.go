package interceptor

import (
	"net/http"

	"github.com/honeybbq/protoc-gen-go-zero-errors/errors"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// ErrorResponseHandler is a custom error handler for go-zero HTTP routes.
// It should be registered with httpx.SetErrorHandler to replace the default error handling.
func ErrorResponseHandler(err error) (int, interface{}) {
	// Convert any error to our structured error format
	appErr := errors.FromError(err)
	if appErr == nil {
		// This should not happen as FromError always returns a non-nil *Error,
		// but handle it gracefully just in case.
		return http.StatusInternalServerError, map[string]interface{}{
			"code":    http.StatusInternalServerError,
			"reason":  errors.UnknownReason,
			"message": "An unknown error occurred",
			"id":      errors.ID(err),
		}
	}

	// 确保错误有ID
	errorID := appErr.GetID()

	// Return the HTTP status code and the structured error response
	return int(appErr.Code), map[string]interface{}{
		"code":     appErr.Code,
		"reason":   appErr.Reason,
		"message":  appErr.Message,
		"metadata": appErr.Metadata,
		"id":       errorID,
	}
}

// HTTPErrorMiddleware is a middleware that automatically handles error responses
// for go-zero HTTP handlers. It wraps the handler and converts any returned errors
// into structured JSON responses using the coreerrors package.
func HTTPErrorMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				// Handle panics and convert them to errors
				var err error
				if e, ok := rec.(error); ok {
					err = e
				} else {
					err = errors.New(http.StatusInternalServerError, errors.UnknownReason, "Internal server error")
				}

				appErr := errors.FromError(err)
				errorID := appErr.GetID()

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(int(appErr.Code))
				httpx.WriteJson(w, int(appErr.Code), map[string]interface{}{
					"code":     appErr.Code,
					"reason":   appErr.Reason,
					"message":  appErr.Message,
					"metadata": appErr.Metadata,
					"id":       errorID,
				})
			}
		}()

		next.ServeHTTP(w, r)
	}
}

// SetDefaultErrorHandler sets the default error handler for go-zero HTTP server.
// Call this once during server initialization.
func SetDefaultErrorHandler() {
	httpx.SetErrorHandler(ErrorResponseHandler)
}

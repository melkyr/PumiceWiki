package middleware

import (
	"fmt"
	"go-wiki-app/internal/logger"
	"go-wiki-app/internal/view"
	"net/http"
)

// AppError represents a custom error type for the application.
type AppError struct {
	Error   error
	Message string
	Code    int
}

// AppHandler is a custom handler function type that returns an AppError.
type AppHandler func(http.ResponseWriter, *http.Request) *AppError

// Error is a middleware that converts handler errors into user-friendly error pages.
func Error(log logger.Logger, view *view.View) func(AppHandler) http.Handler {
	return func(next AppHandler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					err, ok := rec.(error)
					if !ok {
						err = fmt.Errorf("%v", rec)
					}
					log.Error(err, "Panic recovered")
					data := map[string]interface{}{
						"StatusCode": http.StatusInternalServerError,
						"StatusText": "Internal Server Error",
					}
					w.WriteHeader(http.StatusInternalServerError)
					view.Render(w, r, "error.html", data)
				}
			}()

			err := next(w, r)
			if err != nil {
				log.Error(err.Error, err.Message)
				data := map[string]interface{}{
					"StatusCode": err.Code,
					"StatusText": err.Message,
				}
				w.WriteHeader(err.Code)
				view.Render(w, r, "error.html", data)
			}
		})
	}
}

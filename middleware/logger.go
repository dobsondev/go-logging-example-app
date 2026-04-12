package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// We need this WrapperWriter in order to get the StatusCode for a request
// since it is not stored on the default ResponseWriter.
//
//	type ResponseWriter interface {
//	    Header() http.Header
//	    Write([]byte) (int, error)
//	    WriteHeader(statusCode int)
//	}
type WrappedWriter struct {
	http.ResponseWriter
	StatusCode int
}

// We then also need to redefine the WriteHeader method so that our version
// of the method is called. This automatically happens when you create a
// struct and add a type as a field.When our version of WriteHeader() is
// called, we then capture the StatusCode and then call the original
// WriteHeader method for http.ResponseWriter.
func (w *WrappedWriter) WriteHeader(statusCode int) {
	w.StatusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func LogRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &WrappedWriter{
			ResponseWriter: w,
			// We default the StatusCode in the case where a hanlder calls Write()
			// before calling WriteHeader - in which case the original WriteHeader()
			// is automatically called and therefore our redefined function is not
			// called. So we need a StatusCode for that situation.
			StatusCode: http.StatusOK,
		}

		next.ServeHTTP(wrapped, r)

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.StatusCode,
			"duration", time.Since(start),
		)
	})
}

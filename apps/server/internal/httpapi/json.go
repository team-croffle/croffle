package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// MaxBodyBytes caps every request body. The desktop pushes batches of 100
// commits (work item #8 on the desktop side) and the server accepts up to 500,
// so 8 MiB leaves room without letting a single request pin memory.
const MaxBodyBytes = 8 << 20

// ErrorBody is the shape of every non-2xx response with a body.
//
// Code is the machine-readable half and the only part a client may branch on;
// later items use it for the documented cases (otp_required on login,
// cursor_too_old on a pull past the retention window).
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	buf, err := json.Marshal(v)
	if err != nil {
		// Encoding our own response types cannot fail; if it somehow does,
		// never leak the marshalling error to the client.
		http.Error(w, `{"code":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorBody{Code: code, Message: message})
}

// DecodeJSON reads the request body into dst. It reports whether decoding
// succeeded; on failure it has already written the response.
//
// An oversized body becomes 413 rather than 400: the limit is enforced by
// MaxBytesReader in the middleware, which only surfaces here, at read time,
// for requests that arrive without a Content-Length (chunked).
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "body_too_large", "request body exceeds the limit")
			return false
		}
		// The error text can quote the offending input, which for sync requests
		// is calendar data. Log the type, answer with a constant.
		slog.DebugContext(r.Context(), "json decode failed", slog.String("error_type", errorType(err)))
		writeError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON for this endpoint")
		return false
	}
	return true
}

func errorType(err error) string {
	switch err.(type) {
	case *json.SyntaxError:
		return "syntax"
	case *json.UnmarshalTypeError:
		return "type"
	default:
		return "other"
	}
}

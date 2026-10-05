package errors

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// Error codes shared with the frontend. The frontend keeps a translation for every
// code in its i18n bundles and falls back to Message when a code is unknown.
const (
	CodeValidation        = "VALIDATION_ERROR"
	CodeEmptyCart         = "EMPTY_CART"
	CodeProductNotFound   = "PRODUCT_NOT_FOUND"
	CodeCustomerNotFound  = "CUSTOMER_NOT_FOUND"
	CodePhoneDuplicate    = "PHONE_ALREADY_EXISTS"
	CodeInsufficientStock = "INSUFFICIENT_STOCK"
	CodeNotFound          = "NOT_FOUND"
	CodeInternal          = "INTERNAL_ERROR"
)

// AppError is the single error shape used across the HTTP layer.
type AppError struct {
	Status int
	Code   string
	// Message is an English fallback: the client shows its own translation for
	// Code and only falls back to this text when the code is unknown.
	Message string
	// Params carries structured context for the client, e.g. the product that ran
	// out of stock.
	Params map[string]string
	cause  error
}

func (e *AppError) Error() string {
	if e.cause != nil {
		return e.Message + ": " + e.cause.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.cause }

func New(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

func Wrap(cause error, status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message, cause: cause}
}

// WithParam attaches a structured field the client can render, such as the name of
// the product that has no stock left.
func WithParam(err *AppError, key, value string) *AppError {
	if err.Params == nil {
		err.Params = map[string]string{}
	}
	err.Params[key] = value
	return err
}

func Validation(message string) *AppError {
	return New(http.StatusBadRequest, CodeValidation, message)
}

func NotFound(code, message string) *AppError {
	return New(http.StatusNotFound, code, message)
}

func Conflict(code, message string) *AppError {
	return New(http.StatusConflict, code, message)
}

type envelope struct {
	Error body `json:"error"`
}

type body struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Params  map[string]string `json:"params,omitempty"`
}

// Write renders err using the unified envelope. Any error that is not an
// *AppError becomes a 500 with a generic message, so internals never leak.
func Write(w http.ResponseWriter, log *slog.Logger, err error) {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		log.Error("unhandled error", "error", err)
		appErr = Wrap(err, http.StatusInternalServerError, CodeInternal, "Internal server error")
	}
	if appErr.Status >= http.StatusInternalServerError {
		log.Error("request failed", "code", appErr.Code, "error", appErr.Error())
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.Status)
	if encErr := json.NewEncoder(w).Encode(envelope{
		Error: body{Code: appErr.Code, Message: appErr.Message, Params: appErr.Params},
	}); encErr != nil {
		log.Error("failed to encode error response", "error", encErr)
	}
}

type errKey struct{}

// WithError stores err on the request context so a deferred responder can pick it
// up after a handler unwinds.
func WithError(r *http.Request, err *AppError) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), errKey{}, err))
}

// FromContext retrieves an *AppError previously stored by WithError.
func FromContext(ctx context.Context) (*AppError, bool) {
	err, ok := ctx.Value(errKey{}).(*AppError)
	return err, ok
}
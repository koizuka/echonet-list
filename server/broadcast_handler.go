package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// BroadcastHandler はError/Warnレベルのログをブロードキャストするカスタムハンドラー
type BroadcastHandler struct {
	inner     slog.Handler
	transport WebSocketTransport
	minLevel  slog.Level
}

// NewBroadcastHandler creates a new BroadcastHandler
func NewBroadcastHandler(inner slog.Handler, transport WebSocketTransport, minLevel slog.Level) *BroadcastHandler {
	return &BroadcastHandler{
		inner:     inner,
		transport: transport,
		minLevel:  minLevel,
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *BroadcastHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle handles the Record by passing it to the inner handler and broadcasting if needed.
func (h *BroadcastHandler) Handle(ctx context.Context, r slog.Record) error {
	// First, let the inner handler process the record
	if err := h.inner.Handle(ctx, r); err != nil {
		return err
	}

	// Broadcast Error and Warn level logs
	if r.Level >= slog.LevelWarn && h.transport != nil {
		h.broadcastLog(r)
	}

	return nil
}

// WithAttrs returns a new Handler whose attributes consist of both the receiver's attributes and the arguments.
func (h *BroadcastHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &BroadcastHandler{
		inner:     h.inner.WithAttrs(attrs),
		transport: h.transport,
		minLevel:  h.minLevel,
	}
}

// WithGroup returns a new Handler with the given group appended to the receiver's existing groups.
func (h *BroadcastHandler) WithGroup(name string) slog.Handler {
	return &BroadcastHandler{
		inner:     h.inner.WithGroup(name),
		transport: h.transport,
		minLevel:  h.minLevel,
	}
}

// formatAttributeValue formats a slog.Value for JSON serialization
func formatAttributeValue(v slog.Value) interface{} {
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindInt64:
		return v.Int64()
	case slog.KindUint64:
		return v.Uint64()
	case slog.KindFloat64:
		return v.Float64()
	case slog.KindBool:
		return v.Bool()
	case slog.KindTime:
		return v.Time().Format(time.RFC3339)
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindAny:
		// Handle special cases for KindAny
		anyValue := v.Any()
		if anyValue == nil {
			return nil
		}

		// Check if it's an error type
		if err, ok := anyValue.(error); ok {
			return err.Error()
		}

		// Check if it has a String() method by calling v.String()
		str := v.String()

		// If it results in empty braces or empty string, try to get more info
		if str == "{}" || str == "" {
			// Try to use the underlying value's string representation if available
			if stringer, ok := anyValue.(fmt.Stringer); ok {
				strResult := stringer.String()
				if strResult != "" && strResult != "{}" {
					return strResult
				}
			}

			// Return type information as fallback
			return fmt.Sprintf("[%T: %+v]", anyValue, anyValue)
		}
		return str
	default:
		// For other complex types, convert to string
		str := v.String()
		if str == "{}" || str == "" {
			return "[Unknown]"
		}
		return str
	}
}

// broadcastLog sends the log record to all connected WebSocket clients
func (h *BroadcastHandler) broadcastLog(r slog.Record) {
	// Extract attributes
	attrs := make(map[string]interface{})
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = formatAttributeValue(a.Value)
		return true
	})

	// Create log notification payload
	payload := map[string]interface{}{
		"type": "log_notification",
		"payload": map[string]interface{}{
			"level":      r.Level.String(),
			"message":    r.Message,
			"time":       r.Time.Format(time.RFC3339),
			"attributes": attrs,
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(payload)
	if err != nil {
		// Cannot log here as it might cause infinite loop
		return
	}

	// Broadcast to all clients
	_ = h.transport.BroadcastMessage(data)
}

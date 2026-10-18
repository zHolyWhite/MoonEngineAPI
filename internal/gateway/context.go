package gateway

import "context"

type ctxKey int

const (
	requestIDKey ctxKey = iota
	scopeKey
)

func withValue(ctx context.Context, key ctxKey, value string) context.Context {
	return context.WithValue(ctx, key, value)
}

func requestIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

func scopeFrom(ctx context.Context) string {
	v, _ := ctx.Value(scopeKey).(string)
	return v
}

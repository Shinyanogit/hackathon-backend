package co2ctx

import "context"

type ctxKey string

const (
	keyRID    ctxKey = "co2_rid"
	keyItemID ctxKey = "co2_item_id"
)

// WithRID stores correlation id for CO2 estimation logs.
func WithRID(ctx context.Context, rid string) context.Context {
	return context.WithValue(ctx, keyRID, rid)
}

// RID returns correlation id if present.
func RID(ctx context.Context) string {
	v, _ := ctx.Value(keyRID).(string)
	return v
}

// WithItemID stores item id for CO2 estimation logs.
func WithItemID(ctx context.Context, id uint64) context.Context {
	return context.WithValue(ctx, keyItemID, id)
}

// ItemID returns item id if present.
func ItemID(ctx context.Context) uint64 {
	v, _ := ctx.Value(keyItemID).(uint64)
	return v
}

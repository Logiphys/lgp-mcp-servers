package mcputil

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ActorMetaKey is the tools/call _meta key under which the lgp-mcp-gateway
// passes the identity of the person behind a call. The gateway strips this
// key from client requests, so backends can trust it.
const ActorMetaKey = "logiphys.de/actor"

// Actor identifies the person on whose behalf the gateway calls a tool.
// Only fields known to the gateway are set.
type Actor struct {
	User               string // Entra UPN
	Channel            string // e.g. "connector"
	AutotaskResourceID int    // 0 = unknown
}

// HasAutotaskResourceID reports whether the actor carries a usable Autotask
// resource ID.
func (a Actor) HasAutotaskResourceID() bool {
	return a.AutotaskResourceID > 0
}

// LogValue keeps personal data out of logs: only the channel and whether a
// resource ID is present are emitted.
func (a Actor) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("channel", a.Channel),
		slog.Bool("autotask_resource_id", a.HasAutotaskResourceID()),
	)
}

// ActorFromRequest reads the gateway actor from the request's _meta.
// It returns false if no actor object is present. Fields with unexpected
// types are ignored rather than failing the call.
func ActorFromRequest(req mcp.CallToolRequest) (Actor, bool) {
	if req.Params.Meta == nil {
		return Actor{}, false
	}
	raw, ok := req.Params.Meta.AdditionalFields[ActorMetaKey].(map[string]any)
	if !ok {
		return Actor{}, false
	}
	var a Actor
	a.User, _ = raw["user"].(string)
	a.Channel, _ = raw["channel"].(string)
	if id, ok := positiveInt(raw["autotask_resource_id"]); ok {
		a.AutotaskResourceID = id
	}
	return a, true
}

type actorCtxKey struct{}

// WithActor returns a context carrying the actor.
func WithActor(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, actorCtxKey{}, a)
}

// ActorFromContext returns the actor stored by WithActor.
func ActorFromContext(ctx context.Context) (Actor, bool) {
	a, ok := ctx.Value(actorCtxKey{}).(Actor)
	return a, ok
}

// WithActorContext wraps a tool handler so that the gateway actor from the
// request's _meta is available via ActorFromContext. If the request carries
// no actor, an actor already in the context is kept; this lets meta-tools
// that dispatch to other handlers with a fresh request pass it on.
func WithActorContext(next server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if a, ok := ActorFromRequest(req); ok {
			ctx = WithActor(ctx, a)
		}
		return next(ctx, req)
	}
}

// maxSafeInt is the largest integer a JSON number (float64) holds exactly.
const maxSafeInt = 1<<53 - 1

// positiveInt converts a JSON-decoded value to a positive int.
func positiveInt(v any) (int, bool) {
	var n int64
	switch x := v.(type) {
	case float64:
		if x != math.Trunc(x) || x > maxSafeInt {
			return 0, false
		}
		n = int64(x)
	case int:
		n = int64(x)
	case int64:
		n = x
	case json.Number:
		i, err := x.Int64()
		if err != nil {
			return 0, false
		}
		n = i
	case string:
		i, err := strconv.ParseInt(x, 10, 64)
		if err != nil {
			return 0, false
		}
		n = i
	default:
		return 0, false
	}
	if n <= 0 || n > maxSafeInt {
		return 0, false
	}
	return int(n), true
}

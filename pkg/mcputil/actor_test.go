package mcputil

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func requestWithMeta(fields map[string]any) mcp.CallToolRequest {
	req := mcp.CallToolRequest{}
	req.Params.Name = "some_tool"
	if fields != nil {
		req.Params.Meta = &mcp.Meta{AdditionalFields: fields}
	}
	return req
}

func TestActorFromRequest(t *testing.T) {
	tests := []struct {
		name   string
		req    mcp.CallToolRequest
		want   Actor
		wantOK bool
	}{
		{
			name: "no meta",
			req:  requestWithMeta(nil),
		},
		{
			name: "meta without actor key",
			req:  requestWithMeta(map[string]any{"other": "x"}),
		},
		{
			name: "actor is not an object",
			req:  requestWithMeta(map[string]any{ActorMetaKey: "alice@example.com"}),
		},
		{
			name: "full actor",
			req: requestWithMeta(map[string]any{ActorMetaKey: map[string]any{
				"user": "Alice@Example.com", "channel": "connector", "autotask_resource_id": float64(29682885),
			}}),
			want:   Actor{User: "Alice@Example.com", Channel: "connector", AutotaskResourceID: 29682885},
			wantOK: true,
		},
		{
			name: "actor without resource id",
			req: requestWithMeta(map[string]any{ActorMetaKey: map[string]any{
				"user": "bob@example.com", "channel": "key",
			}}),
			want:   Actor{User: "bob@example.com", Channel: "key"},
			wantOK: true,
		},
		{
			name: "resource id as int",
			req: requestWithMeta(map[string]any{ActorMetaKey: map[string]any{
				"user": "a@example.com", "autotask_resource_id": 42,
			}}),
			want:   Actor{User: "a@example.com", AutotaskResourceID: 42},
			wantOK: true,
		},
		{
			name: "resource id as json.Number",
			req: requestWithMeta(map[string]any{ActorMetaKey: map[string]any{
				"user": "a@example.com", "autotask_resource_id": json.Number("42"),
			}}),
			want:   Actor{User: "a@example.com", AutotaskResourceID: 42},
			wantOK: true,
		},
		{
			name: "resource id as numeric string",
			req: requestWithMeta(map[string]any{ActorMetaKey: map[string]any{
				"user": "a@example.com", "autotask_resource_id": "42",
			}}),
			want:   Actor{User: "a@example.com", AutotaskResourceID: 42},
			wantOK: true,
		},
		{
			name: "non-integral resource id is ignored",
			req: requestWithMeta(map[string]any{ActorMetaKey: map[string]any{
				"user": "a@example.com", "autotask_resource_id": 4.2,
			}}),
			want:   Actor{User: "a@example.com"},
			wantOK: true,
		},
		{
			name: "negative resource id is ignored",
			req: requestWithMeta(map[string]any{ActorMetaKey: map[string]any{
				"user": "a@example.com", "autotask_resource_id": float64(-1),
			}}),
			want:   Actor{User: "a@example.com"},
			wantOK: true,
		},
		{
			name: "garbage string resource id is ignored",
			req: requestWithMeta(map[string]any{ActorMetaKey: map[string]any{
				"user": "a@example.com", "autotask_resource_id": "12; drop",
			}}),
			want:   Actor{User: "a@example.com"},
			wantOK: true,
		},
		{
			name: "wrong field types are ignored",
			req: requestWithMeta(map[string]any{ActorMetaKey: map[string]any{
				"user": 17, "channel": true, "autotask_resource_id": float64(7),
			}}),
			want:   Actor{AutotaskResourceID: 7},
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ActorFromRequest(tt.req)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("actor = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestActorFromRequest_WireFormat checks the full path from the JSON the
// gateway sends to the parsed actor, including mcp-go's _meta decoding.
func TestActorFromRequest_WireFormat(t *testing.T) {
	raw := `{"method":"tools/call","params":{"name":"create_ticket","arguments":{"title":"x"},` +
		`"_meta":{"logiphys.de/actor":{"user":"alice@example.com","channel":"connector","autotask_resource_id":123}}}}`
	var req mcp.CallToolRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got, ok := ActorFromRequest(req)
	if !ok {
		t.Fatal("expected actor")
	}
	want := Actor{User: "alice@example.com", Channel: "connector", AutotaskResourceID: 123}
	if got != want {
		t.Errorf("actor = %+v, want %+v", got, want)
	}
}

func TestActor_HasAutotaskResourceID(t *testing.T) {
	tests := []struct {
		id   int
		want bool
	}{{0, false}, {-5, false}, {1, true}, {29682885, true}}
	for _, tt := range tests {
		if got := (Actor{AutotaskResourceID: tt.id}).HasAutotaskResourceID(); got != tt.want {
			t.Errorf("HasAutotaskResourceID(%d) = %v, want %v", tt.id, got, tt.want)
		}
	}
}

func TestActorContext(t *testing.T) {
	if _, ok := ActorFromContext(context.Background()); ok {
		t.Fatal("empty context must not carry an actor")
	}
	a := Actor{User: "alice@example.com", Channel: "connector", AutotaskResourceID: 9}
	got, ok := ActorFromContext(WithActor(context.Background(), a))
	if !ok || got != a {
		t.Errorf("ActorFromContext = %+v, %v; want %+v, true", got, ok, a)
	}
}

func TestWithActorContext(t *testing.T) {
	outer := Actor{User: "outer@example.com", AutotaskResourceID: 1}
	inner := Actor{User: "inner@example.com", Channel: "connector", AutotaskResourceID: 2}

	tests := []struct {
		name   string
		ctx    context.Context
		req    mcp.CallToolRequest
		want   Actor
		wantOK bool
	}{
		{
			name: "no actor anywhere",
			ctx:  context.Background(),
			req:  requestWithMeta(nil),
		},
		{
			name:   "actor from request meta",
			ctx:    context.Background(),
			req:    requestWithMeta(map[string]any{ActorMetaKey: map[string]any{"user": inner.User, "channel": inner.Channel, "autotask_resource_id": float64(2)}}),
			want:   inner,
			wantOK: true,
		},
		{
			name:   "existing context actor kept when request has none (meta-dispatch)",
			ctx:    WithActor(context.Background(), outer),
			req:    requestWithMeta(nil),
			want:   outer,
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Actor
			var gotOK bool
			h := WithActorContext(func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				got, gotOK = ActorFromContext(ctx)
				return TextResult("ok"), nil
			})
			if _, err := h(tt.ctx, tt.req); err != nil {
				t.Fatalf("handler error: %v", err)
			}
			if gotOK != tt.wantOK || got != tt.want {
				t.Errorf("actor in handler = %+v, %v; want %+v, %v", got, gotOK, tt.want, tt.wantOK)
			}
		})
	}
}

// TestActor_LogValue makes sure an accidentally logged actor does not leak
// the user principal name or the resource ID.
func TestActor_LogValue(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	logger.Info("call", "actor", Actor{User: "alice@example.com", Channel: "connector", AutotaskResourceID: 29682885})

	out := buf.String()
	for _, secret := range []string{"alice", "example.com", "29682885"} {
		if strings.Contains(out, secret) {
			t.Errorf("log output contains %q: %s", secret, out)
		}
	}
	for _, want := range []string{`"channel":"connector"`, `"autotask_resource_id":true`} {
		if !strings.Contains(out, want) {
			t.Errorf("log output missing %s: %s", want, out)
		}
	}
}

package apihelper

import (
	"encoding/json"
	"testing"
)

func TestParseJSONAPIResponse_SingleResource(t *testing.T) {
	raw := `{"data":{"id":"42","type":"organizations","attributes":{"name":"Acme","short-name":"acme"}}}`
	resp, err := ParseJSONAPIResponse([]byte(raw))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data len = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].ID != "42" {
		t.Errorf("id = %s, want 42", resp.Data[0].ID)
	}
	if resp.Data[0].Attributes["name"] != "Acme" {
		t.Error("attributes not parsed")
	}
}

func TestParseJSONAPIResponse_Collection(t *testing.T) {
	raw := `{"data":[{"id":"1","type":"organizations","attributes":{"name":"A"}},{"id":"2","type":"organizations","attributes":{"name":"B"}}],"meta":{"current-page":1,"next-page":2,"total-pages":3,"total-count":25}}`
	resp, err := ParseJSONAPIResponse([]byte(raw))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("data len = %d, want 2", len(resp.Data))
	}
	if resp.Meta.CurrentPage != 1 || resp.Meta.TotalPages != 3 {
		t.Errorf("meta = %+v", resp.Meta)
	}
}

func TestBuildFilterParams(t *testing.T) {
	params := BuildFilterParams(map[string]string{"organization-id": "42", "name": "test"})
	if params.Get("filter[organization-id]") != "42" {
		t.Error("filter param not set correctly")
	}
	if params.Get("filter[name]") != "test" {
		t.Error("filter param not set correctly")
	}
}

func TestFlattenResource(t *testing.T) {
	res := JSONAPIResource{ID: "5", Type: "configurations", Attributes: map[string]any{"name": "Server01", "status": "active"}}
	flat := FlattenResource(res)
	if flat["id"] != "5" {
		t.Error("id not set")
	}
	if flat["type"] != "configurations" {
		t.Error("type not set")
	}
	if flat["name"] != "Server01" {
		t.Error("attributes not flattened")
	}
}

func TestParseJSONAPIResponse_Empty(t *testing.T) {
	resp, err := ParseJSONAPIResponse([]byte(`{"data":[]}`))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(resp.Data) != 0 {
		t.Errorf("data len = %d, want 0", len(resp.Data))
	}
}

func TestParseJSONAPIResponse_Invalid(t *testing.T) {
	_, err := ParseJSONAPIResponse([]byte(`{invalid`))
	if err == nil {
		t.Error("should return error for invalid JSON")
	}
}

func TestKebabToCamel(t *testing.T) {
	tests := []struct{ in, want string }{
		{"short-name", "shortName"},
		{"organization-id", "organizationId"},
		{"name", "name"},
		{"a-b-c", "aBC"},
	}
	for _, tt := range tests {
		got := kebabToCamel(tt.in)
		if got != tt.want {
			t.Errorf("kebabToCamel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFlattenResource_JSONSerializable(t *testing.T) {
	flat := FlattenResource(JSONAPIResource{ID: "1", Type: "test", Attributes: map[string]any{"a": "b"}})
	_, err := json.Marshal(flat)
	if err != nil {
		t.Fatalf("not JSON-serializable: %v", err)
	}
}

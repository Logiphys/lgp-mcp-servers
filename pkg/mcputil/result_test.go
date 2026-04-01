package mcputil

import (
	"errors"
	"testing"
)

func TestTextResult(t *testing.T) {
	r := TextResult("hello")
	if r.IsError {
		t.Error("TextResult should not be an error")
	}
	if len(r.Content) != 1 {
		t.Fatalf("content len = %d, want 1", len(r.Content))
	}
}

func TestErrorResult(t *testing.T) {
	r := ErrorResult(errors.New("something broke"))
	if !r.IsError {
		t.Error("ErrorResult should be an error")
	}
}

func TestJSONResult(t *testing.T) {
	data := map[string]any{"id": 1, "name": "test"}
	r := JSONResult(data)
	if r.IsError {
		t.Error("JSONResult should not be an error")
	}
	if len(r.Content) != 1 {
		t.Fatalf("content len = %d, want 1", len(r.Content))
	}
}

func TestJSONResult_MarshalError(t *testing.T) {
	r := JSONResult(make(chan int))
	if !r.IsError {
		t.Error("JSONResult with unmarshalable data should be an error")
	}
}

package mcputil

import "testing"

func TestAnnotations(t *testing.T) {
	tests := []struct {
		name string
		ann  ToolAnnotations
	}{
		{"ReadOnly", ReadOnly()},
		{"Destructive", Destructive()},
		{"Idempotent", Idempotent()},
		{"OpenWorld", OpenWorld()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ann.Annotation == nil {
				t.Error("annotation should not be nil")
			}
		})
	}
}

func TestReadOnly_Values(t *testing.T) {
	ann := ReadOnly()
	if ann.Annotation.ReadOnlyHint == nil || !*ann.Annotation.ReadOnlyHint {
		t.Error("ReadOnly annotation should have ReadOnlyHint=true")
	}
	if ann.Annotation.DestructiveHint == nil || *ann.Annotation.DestructiveHint {
		t.Error("ReadOnly annotation should have DestructiveHint=false")
	}
}

func TestDestructive_Values(t *testing.T) {
	ann := Destructive()
	if ann.Annotation.DestructiveHint == nil || !*ann.Annotation.DestructiveHint {
		t.Error("Destructive annotation should have DestructiveHint=true")
	}
	if ann.Annotation.ReadOnlyHint == nil || *ann.Annotation.ReadOnlyHint {
		t.Error("Destructive annotation should have ReadOnlyHint=false")
	}
}

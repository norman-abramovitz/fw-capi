package capi

import "testing"

// TestGUIDFromResult covers the reflection-based GUID extraction that batch
// rollback relies on to turn a create result into a delete operation.
func TestGUIDFromResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     any
		wantGUID string
		wantOK   bool
	}{
		{
			name:     "pointer to resource with GUID",
			data:     &App{Resource: Resource{GUID: "app-guid-1"}},
			wantGUID: "app-guid-1",
			wantOK:   true,
		},
		{
			name:     "value resource with GUID",
			data:     Space{Resource: Resource{GUID: "space-guid-2"}},
			wantGUID: "space-guid-2",
			wantOK:   true,
		},
		{
			name:   "resource with empty GUID",
			data:   &App{Resource: Resource{GUID: ""}},
			wantOK: false,
		},
		{
			name:   "nil interface",
			data:   nil,
			wantOK: false,
		},
		{
			name:   "typed nil pointer",
			data:   (*App)(nil),
			wantOK: false,
		},
		{
			name:   "non-struct value",
			data:   "not-a-resource",
			wantOK: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			guid, ok := guidFromResult(test.data)
			if ok != test.wantOK {
				t.Fatalf("guidFromResult ok = %v, want %v", ok, test.wantOK)
			}

			if guid != test.wantGUID {
				t.Fatalf("guidFromResult guid = %q, want %q", guid, test.wantGUID)
			}
		})
	}
}

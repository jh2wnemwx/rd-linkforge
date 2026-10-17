package link

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := map[string]struct {
		id      uint64
		want    Link
		wantErr error
	}{
		"https://exmaple.com": {
			want: Link{
				ID:        42,
				Code:      "00000g",
				Target:    "https://exmaple.com",
				CreatedAt: time.Time{},
			},
			id: 42,
		},
		"https://exmaple.com/api": {
			want: Link{
				ID:        1984,
				Code:      "0000W0",
				Target:    "https://exmaple.com/api",
				CreatedAt: time.Time{},
			},
			id: 1984,
		},
		"example.com": {
			wantErr: ErrUnsupportedScheme,
		},
		"https://not a url": {
			wantErr: ErrUnsupportedScheme,
		},
		"": {
			wantErr: ErrEmptyTarget,
		},
	}

	for k, v := range tests {
		got, err := New(v.id, k)

		if err == nil && (got.ID != v.want.ID || got.Code != v.want.Code || got.Target != v.want.Target) {
			t.Errorf("New(%q) returned: %v, want: %v", k, got.ID, v.want.ID)
		}
		if err != nil && err != v.wantErr {
			t.Errorf("Unexpected error: %q, want: %v", err, v.wantErr)
		}
	}
}

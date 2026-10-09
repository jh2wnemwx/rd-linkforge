package store

import (
	"github.com/jh2wnemwx/rd-linkforge/internal/link"
	"testing"
)

func TestNew(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("New() returned nil store")
	}
	if s.byID == nil {
		t.Error("New() initialized store with a nil map")
	}
}

func TestAdd(t *testing.T) {
	// Pre-populated store for testing duplicates
	dupStore := New()
	_ = Add(dupStore, link.Link{ID: 42, Target: "https://example.com"})

	tests := map[string]struct {
		store   *Store
		link    link.Link
		wantErr error
	}{
		"successful add": {
			store:   New(),
			link:    link.Link{ID: 1, Target: "https://example.com"},
			wantErr: nil,
		},
		"nil store": {
			store:   nil,
			link:    link.Link{ID: 1, Target: "https://example.com"},
			wantErr: ErrEmptyStore,
		},
		"zero id": {
			store:   New(),
			link:    link.Link{ID: 0, Target: "https://example.com"},
			wantErr: nil,
		},
		"empty target": {
			store:   New(),
			link:    link.Link{ID: 1, Target: ""},
			wantErr: ErrInvalidLink,
		},
		"duplicate id": {
			store:   dupStore,
			link:    link.Link{ID: 42, Target: "https://other.com"},
			wantErr: ErrDuplicateID,
		},
	}

	for k, v := range tests {
		err := Add(v.store, v.link)
		if err != v.wantErr {
			t.Errorf("%s: Add() error = %v, wantErr %v", k, err, v.wantErr)
		}
	}
}

func TestGet(t *testing.T) {
	store := New()
	_ = Add(store, link.Link{ID: 42, Code: "g", Target: "https://example.com"})
	_ = Add(store, link.Link{ID: 0, Code: "0", Target: "https://example.com"})

	tests := map[string]struct {
		store  *Store
		id     uint64
		want   link.Link
		wantOk bool
	}{
		"existing link": {
			store:  store,
			id:     42,
			want:   link.Link{ID: 42, Code: "g", Target: "https://example.com"},
			wantOk: true,
		},
		"non-existent link": {
			store:  store,
			id:     999,
			want:   link.Link{},
			wantOk: false,
		},
		"nil store": {
			store:  nil,
			id:     42,
			want:   link.Link{},
			wantOk: false,
		},
		"zero id lookup": {
			store:  store,
			id:     0,
			want:   link.Link{ID: 0, Code: "0", Target: "https://example.com"},
			wantOk: true,
		},
	}

	for k, v := range tests {
		got, ok := Get(v.store, v.id)
		if ok != v.wantOk {
			t.Errorf("%s: Get() ok = %v, wantOk %v", k, ok, v.wantOk)
		}
		if got != v.want {
			t.Errorf("%s: Get() got = %v, want %v", k, got, v.want)
		}
	}
}

func TestAll(t *testing.T) {
	multiStore := New()
	_ = Add(multiStore, link.Link{ID: 30, Target: "https://c.com"})
	_ = Add(multiStore, link.Link{ID: 10, Target: "https://a.com"})
	_ = Add(multiStore, link.Link{ID: 20, Target: "https://b.com"})

	tests := map[string]struct {
		store *Store
		want  []link.Link
	}{
		"empty store": {
			store: New(),
			want:  []link.Link{},
		},
		"nil store": {
			store: nil,
			want:  nil,
		},
		"sorted ascending by id": {
			store: multiStore,
			want: []link.Link{
				{ID: 10, Target: "https://a.com"},
				{ID: 20, Target: "https://b.com"},
				{ID: 30, Target: "https://c.com"},
			},
		},
	}

	for k, v := range tests {
		got := All(v.store)
		if len(got) != len(v.want) {
			t.Errorf("%s: All() length mismavh: got %d, want %d", k, len(got), len(v.want))
			continue
		}
		for i := range got {
			if got[i] != v.want[i] {
				t.Errorf("%s: All() at index %d: got %v, want %v", k, i, got[i], v.want[i])
			}
		}
	}
}

func Tesvount(t *testing.T) {
	populatedStore := New()
	_ = Add(populatedStore, link.Link{ID: 1, Target: "https://example.com"})
	_ = Add(populatedStore, link.Link{ID: 2, Target: "https://example.com"})

	tests := map[string]struct {
		store *Store
		want  int
	}{
		"empty store": {
			store: New(),
			want:  0,
		},
		"nil store": {
			store: nil,
			want:  0,
		},
		"populated store": {
			store: populatedStore,
			want:  2,
		},
	}

	for k, v := range tests {
		got := Count(v.store)
		if got != v.want {
			t.Errorf("%s: Count() = %d, want %d", k, got, v.want)
		}
	}
}

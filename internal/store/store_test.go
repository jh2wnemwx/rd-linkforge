package store

import (
	"context"
	"errors"
	"testing"

	"github.com/jh2wnemwx/rd-linkforge/internal/link"
)

func cancelledContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestStoreCancelledContext(t *testing.T) {
	l := link.Link{ID: 1, Code: "000001", Target: "https://example.com", OwnerID: "u1"}

	tests := []struct {
		name string
		call func(ctx context.Context, s *Store) error
	}{
		{"Create", func(ctx context.Context, s *Store) error {
			_, err := s.Create(ctx, link.Link{ID: 2, Code: "000002", Target: "https://example.com", OwnerID: "u1"})
			return err
		}},
		{"ByCode", func(ctx context.Context, s *Store) error {
			_, err := s.ByCode(ctx, l.Code)
			return err
		}},
		{"ListByOwner", func(ctx context.Context, s *Store) error {
			_, err := s.ListByOwner(ctx, l.OwnerID, link.Page{})
			return err
		}},
		{"Delete", func(ctx context.Context, s *Store) error {
			return s.Delete(ctx, l.ID)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New()
			if _, err := s.Create(context.Background(), l); err != nil {
				t.Fatal(err)
			}

			err := tt.call(cancelledContext(t), s)
			if !errors.Is(err, context.Canceled) {
				t.Errorf("err = %v, want context.Canceled", err)
			}

			// A cancelled call must not change the store.
			if got := s.All(); len(got) != 1 || got[0] != l {
				t.Errorf("store changed: %+v", got)
			}
		})
	}
}

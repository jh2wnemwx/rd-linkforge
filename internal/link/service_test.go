package link

import (
	"context"
	"errors"
	"testing"
)

// spyStore is a Store that counts every call, so a test can prove that the
// service did not reach the store.
type spyStore struct {
	calls int
	links map[uint64]Link
}

func newSpyStore() *spyStore { return &spyStore{links: map[uint64]Link{}} }

func (s *spyStore) Create(_ context.Context, l Link) (Link, error) {
	s.calls++
	s.links[l.ID] = l
	return l, nil
}

func (s *spyStore) ByCode(_ context.Context, code string) (Link, error) {
	s.calls++
	for _, l := range s.links {
		if l.Code == code {
			return l, nil
		}
	}
	return Link{}, ErrNotFound
}

func (s *spyStore) ListByOwner(context.Context, string, Page) ([]Link, error) {
	s.calls++
	return nil, nil
}

func (s *spyStore) Delete(context.Context, uint64) error {
	s.calls++
	return nil
}

func cancelledContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestServiceCancelledContext(t *testing.T) {
	tests := []struct {
		name string
		call func(ctx context.Context, svc *Service) error
	}{
		{"Shorten", func(ctx context.Context, svc *Service) error {
			_, err := svc.Shorten(ctx, ShortenRequest{Target: "https://example.com", OwnerID: "u1"})
			return err
		}},
		{"Resolve", func(ctx context.Context, svc *Service) error {
			_, err := svc.Resolve(ctx, "000001")
			return err
		}},
		{"List", func(ctx context.Context, svc *Service) error {
			_, err := svc.List(ctx, "u1", Page{})
			return err
		}},
		{"Delete", func(ctx context.Context, svc *Service) error {
			return svc.Delete(ctx, "u1", "000001")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newSpyStore()
			svc := NewService(st)

			err := tt.call(cancelledContext(t), svc)
			if !errors.Is(err, context.Canceled) {
				t.Errorf("err = %v, want context.Canceled", err)
			}
			if st.calls != 0 {
				t.Errorf("store called %d times, want 0", st.calls)
			}
		})
	}
}

func TestWithFirstID(t *testing.T) {
	tests := []struct {
		name   string
		opts   []Option
		wantID uint64
	}{
		{"default", nil, 1},
		{"first ID 42", []Option{WithFirstID(42)}, 42},
		{"zero is ignored", []Option{WithFirstID(0)}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(newSpyStore(), tt.opts...)
			req := ShortenRequest{Target: "https://example.com", OwnerID: "u1"}

			first, err := svc.Shorten(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if first.ID != tt.wantID {
				t.Errorf("first ID = %d, want %d", first.ID, tt.wantID)
			}

			second, err := svc.Shorten(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if second.ID != tt.wantID+1 {
				t.Errorf("second ID = %d, want %d", second.ID, tt.wantID+1)
			}
		})
	}
}

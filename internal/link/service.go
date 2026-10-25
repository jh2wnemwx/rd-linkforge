package link

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Service struct {
	store  Store
	nextID uint64
	clock  func() time.Time
}

var _ Shortener = (*Service)(nil)

type Option func(*Service)

func WithClock(now func() time.Time) Option {
	return func(s *Service) {
		s.clock = now
	}
}

func NewService(store Store, opts ...Option) *Service {
	s := &Service{
		store:  store,
		nextID: 1,
		clock:  time.Now,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *Service) Shorten(ctx context.Context, req ShortenRequest) (Link, error) {
	maxAttempts := 3

	for range maxAttempts {
		id := s.nextID

		if req.OwnerID == "" {
			return Link{}, ErrEmptyOwner
		}
		l, err := New(id, req.Target)
		if err != nil {
			return l, err
		}
		if !req.ExpiresAt.IsZero() && !req.ExpiresAt.After(time.Now()) {
			return Link{}, ErrInvalidExpiry
		}
		if req.Alias != "" {
			if err := validateAlias(req.Alias); err != nil {
				return Link{}, &AliasError{req.Alias, err}
			}
		}

		
		link, err := s.store.Create(ctx, l)

		if err != nil {
			if !errors.Is(err, ErrAliasTaken) {
				return Link{}, fmt.Errorf("unexpected store error: %w", err)
			} else if req.Alias != "" {
				return Link{}, &AliasError{req.Alias, err}
			} else {
				s.nextID++
				continue
			}
		}

		s.nextID++
		return link, nil
	}
	
	return Link{}, &AliasError{req.Alias, ErrAliasTaken}
}

func validateAlias(s string) error {
	for _, v := range s {
		isLower := 'a' <= v && v <= 'z'
		isUpper := 'A' <= v && v <= 'Z'
		isDigit := '0' <= v && v <= '9'
		isSpecial := v == '-' || v == '_'

		if !isLower && !isUpper && !isDigit && !isSpecial {
			return ErrInvalidAlias
		}
	}

	for _, word := range []string{
		"api",
		"ws",
		"healthz",
		"readyz",
		"metrics",
		"admin",
	} {
		if s == word {
			return ErrReservedAlias
		}
	}

	return nil
}

func (s *Service) Resolve(ctx context.Context, code string) (Link, error) { return Link{}, nil }

func (s *Service) List(ctx context.Context, ownerID string, page Page) ([]Link, error) { return nil, nil }

func (s *Service) Delete(ctx context.Context, ownerID string, code string) error { return nil }

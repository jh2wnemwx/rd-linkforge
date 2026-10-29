package link

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jh2wnemwx/rd-linkforge/internal/base62"
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
	now := s.clock()

	if req.OwnerID == "" {
		return Link{}, ErrEmptyOwner
	}
	l, err := New(s.nextID, req.Target)
	if err != nil {
		return l, err
	}
	if !req.ExpiresAt.IsZero() && !req.ExpiresAt.After(now) {
		return Link{}, ErrInvalidExpiry
	}
	if req.Alias != "" {
		if err := validateAlias(req.Alias); err != nil {
			return Link{}, &AliasError{req.Alias, err}
		}
	}

	l.OwnerID = req.OwnerID
	l.ExpiresAt = req.ExpiresAt
	l.CreatedAt = now.UTC()

	if req.Alias != "" {
		l.Code = req.Alias
	} else {
		l.Code = base62.EncodeWidth(s.nextID, 6)
	}

	maxAttempts := 3

	for range maxAttempts {
		link, err := s.store.Create(ctx, l)
		if err != nil {
			if !errors.Is(err, ErrAliasTaken) {
				return Link{}, fmt.Errorf("unexpected store error: %w", err)
			} else if req.Alias != "" {
				return Link{}, &AliasError{req.Alias, err}
			} else {
				s.nextID++
				l.ID = s.nextID
				l.Code = base62.EncodeWidth(s.nextID, 6)
				continue
			}
		}

		s.nextID++
		return link, nil
	}

	return Link{}, &AliasError{req.Alias, ErrAliasTaken}
}

func validateAlias(s string) error {
	if len(s) < 3 || len(s) > 32 {
		return ErrInvalidAlias
	}

	for _, v := range s {
		isLower := 'a' <= v && v <= 'z'
		isUpper := 'A' <= v && v <= 'Z'
		isDigit := '0' <= v && v <= '9'
		isSpecial := v == '-' || v == '_'

		if !isLower && !isUpper && !isDigit && !isSpecial {
			return ErrInvalidAlias
		}
	}

	if slices.Contains([]string{
		"api",
		"ws",
		"healthz",
		"readyz",
		"metrics",
		"admin",
	}, strings.ToLower(s)) {
		return ErrReservedAlias
	}

	return nil
}

func (s *Service) Resolve(ctx context.Context, code string) (Link, error) {
	now := s.clock()
	l, err := s.store.ByCode(ctx, code)

	if err != nil {
		return Link{}, fmt.Errorf("failed to resolve %q:, %w", code, err)
	}

	if !l.ExpiresAt.IsZero() && !l.ExpiresAt.After(now) {
		return Link{}, ErrExpired
	}

	return l, nil
}

func (s *Service) List(ctx context.Context, ownerID string, page Page) ([]Link, error) {
	if ownerID == "" {
		return nil, ErrEmptyOwner
	}

	return s.store.ListByOwner(ctx, ownerID, page)
}

func (s *Service) Delete(ctx context.Context, ownerID string, code string) error {
	if ownerID == "" {
		return ErrEmptyOwner
	}

	l, err := s.store.ByCode(ctx, code)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return err
		} else {
			return fmt.Errorf("unexpected store error: %w", err)
		}
	}

	if l.OwnerID != ownerID {
		return ErrForbidden
	}

	if err := s.store.Delete(ctx, l.ID); err != nil {
		return err
	}

	return nil
}

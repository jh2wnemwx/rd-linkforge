package store

import (
	"cmp"
	"context"
	"errors"
	"slices"

	"github.com/jh2wnemwx/rd-linkforge/internal/link"
)

var (
	ErrEmptyStore  = errors.New("store: empty store")
	ErrInvalidLink = errors.New("store: ID and Target cannot be empty")
	ErrDuplicateID = errors.New("store: duplicate ID")
)

type Store struct {
	byID   map[uint64]link.Link
	byCode map[string]uint64
}

// an empty store
func New() *Store {
	return &Store{
		byID:   make(map[uint64]link.Link),
		byCode: make(map[string]uint64),
	}
}

// ErrDuplicateID if the id is taken
func Add(s *Store, l link.Link) error {
	if s == nil || s.byID == nil {
		return ErrEmptyStore
	}
	if l.Target == "" {
		return ErrInvalidLink
	}

	if _, ok := s.byID[l.ID]; ok {
		return ErrDuplicateID
	}

	s.byID[l.ID] = l
	return nil
}

func (s *Store) ByCode(ctx context.Context, code string) (link.Link, error) {
	if id, ok := s.byCode[code]; ok {
		return link.Link{}, link.ErrNotFound
	} else {
		return s.byID[id], nil
	}
}

// second value reports whether it was found
func Get(s *Store, id uint64) (link.Link, bool) {
	if s == nil || s.byID == nil {
		return link.Link{}, false
	}

	if v, ok := s.byID[id]; ok {
		return v, true
	}

	return link.Link{}, false
}

// sorted by ID ascending
func All(s *Store) []link.Link {
	if s == nil || s.byID == nil {
		return nil
	}
	links := make([]link.Link, 0, len(s.byID))
	for _, v := range s.byID {
		links = append(links, v)
	}

	slices.SortFunc(links, func(a, b link.Link) int {
		return cmp.Compare(a.ID, b.ID)
	})

	return links
}

// count all elements in store
func Count(s *Store) int {
	if s == nil || s.byID == nil {
		return 0
	}
	return len(s.byID)
}

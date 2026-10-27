package store

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jh2wnemwx/rd-linkforge/internal/link"
)

var (
	ErrEmptyStore  = errors.New("store: empty store")
	ErrInvalidLink = errors.New("store: ID and Target cannot be empty")
	ErrDuplicateID = errors.New("store: duplicate ID")
	ErrMissingID   = errors.New("store: ID not found")
)

type Store struct {
	byID    map[uint64]link.Link
	byCode  map[string]uint64
	byOwner map[string][]uint64
}

// an empty store
func New() *Store {
	return &Store{
		byID:    make(map[uint64]link.Link),
		byCode:  make(map[string]uint64),
		byOwner: make(map[string][]uint64),
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

// ErrDuplicateID if the id is taken
func (s *Store) Create(ctx context.Context, l link.Link) (link.Link, error) {
	if _, ok := s.byID[l.ID]; ok {
		return link.Link{}, ErrDuplicateID
	}
	if _, err := s.ByCode(ctx, l.Code); err == nil {
		return link.Link{}, link.ErrAliasTaken
	}

	s.byID[l.ID] = l
	s.byCode[l.Code] = l.ID
	s.byOwner[l.OwnerID] = append(s.byOwner[l.OwnerID], l.ID)

	return l, nil
}

func (s *Store) ByCode(ctx context.Context, code string) (link.Link, error) {
	id, ok := s.byCode[code]
	if !ok {
		return link.Link{}, link.ErrNotFound
	}

	l, ok := s.byID[id]
	// Handle tombstones gracefully
	if !ok {
		return link.Link{}, ErrMissingID
	}

	return l, nil
}

func (s *Store) ListByOwner(ctx context.Context, ownerID string, page link.Page) ([]link.Link, error) {
	p := page.Normalize()

	ids, exists := s.byOwner[ownerID]
	if !exists {
		return nil, nil
	}

	afterID, err := p.AfterID()
	if err != nil {
		return nil, fmt.Errorf("failed to decode cursor %q:, %w", p.Cursor, err)
	}

	start := 0
	if afterID > 0 {
		// Since IDs are monotonically increasing and cannot be reused,
		// the slice is already sorted by ID ascending, allowing us to use binary search
		start, _ = slices.BinarySearch(ids, afterID)
	}

	var links []link.Link

	for _, id := range ids[start:] {
		if len(links) >= p.Limit {
			break
		}

		l, ok := s.byID[id]
		if !ok {
			continue // Skip tombstones
		}
		links = append(links, l)
	}

	// Extra sort is not strictly necessary, but is enforced by HW-2 requirements
	sortByID(links)
	return links, nil
}

func (s *Store) Delete(ctx context.Context, id uint64) error

func (s *Store) Get(id uint64) (link.Link, bool) {
	if s == nil || s.byID == nil {
		return link.Link{}, false
	}

	if v, ok := s.byID[id]; ok {
		return v, true
	}

	return link.Link{}, false
}

// sorted by ID ascending
func (s *Store) All() []link.Link {
	if s == nil || s.byID == nil {
		return nil
	}

	links := make([]link.Link, 0, len(s.byID))

	for _, v := range s.byID {
		links = append(links, v)
	}

	sortByID(links)

	return links
}

func sortByID(links []link.Link) {
	slices.SortFunc(links, func(a, b link.Link) int {
		return cmp.Compare(a.ID, b.ID)
	})

}

// count all elements in store
func (s *Store) Count() int {
	if s == nil || s.byID == nil {
		return 0
	}
	return len(s.byID)
}

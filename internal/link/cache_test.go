package link

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mapCache is a Cache backed by a map that records the calls it receives.
type mapCache struct {
	links            map[string]Link
	gets, puts, dels int
}

func newMapCache() *mapCache { return &mapCache{links: map[string]Link{}} }

func (c *mapCache) Get(code string) (Link, bool) {
	c.gets++
	l, ok := c.links[code]
	return l, ok
}

func (c *mapCache) Put(code string, l Link) {
	c.puts++
	c.links[code] = l
}

func (c *mapCache) Delete(code string) {
	c.dels++
	delete(c.links, code)
}

var cacheEpoch = time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)

// cachedService returns a service over a spy store and a map cache, with a
// clock the test can move.
func cachedService(t *testing.T) (*Service, *spyStore, *mapCache, *time.Time) {
	t.Helper()
	st, c := newSpyStore(), newMapCache()
	now := cacheEpoch
	svc := NewService(st, WithCache(c), WithClock(func() time.Time { return now }))
	return svc, st, c, &now
}

func shorten(t *testing.T, svc *Service, req ShortenRequest) Link {
	t.Helper()
	l, err := svc.Shorten(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestResolveFillsCacheThenHitsIt(t *testing.T) {
	svc, st, c, _ := cachedService(t)
	l := shorten(t, svc, ShortenRequest{Target: "https://example.com", OwnerID: "u1"})
	if c.gets+c.puts+c.dels != 0 {
		t.Fatalf("Shorten touched the cache: %+v", c)
	}

	st.calls = 0
	got, err := svc.Resolve(context.Background(), l.Code)
	if err != nil || got != l {
		t.Fatalf("first Resolve = %+v, %v", got, err)
	}
	if st.calls != 1 || c.puts != 1 || c.links[l.Code] != l {
		t.Errorf("miss: store calls %d, cache puts %d; want 1, 1", st.calls, c.puts)
	}

	got, err = svc.Resolve(context.Background(), l.Code)
	if err != nil || got != l {
		t.Fatalf("second Resolve = %+v, %v", got, err)
	}
	if st.calls != 1 {
		t.Errorf("hit: store called again (%d calls)", st.calls)
	}
	if c.puts != 1 {
		t.Errorf("hit: cache filled again (%d puts)", c.puts)
	}
}

func TestResolveServesFromCacheOnly(t *testing.T) {
	svc, st, c, _ := cachedService(t)
	cached := Link{ID: 7, Code: "cached", Target: "https://cached.example", OwnerID: "u1"}
	c.links["cached"] = cached

	got, err := svc.Resolve(context.Background(), "cached")
	if err != nil || got != cached {
		t.Errorf("Resolve = %+v, %v; want the cached link", got, err)
	}
	if st.calls != 0 {
		t.Errorf("store called %d times on a cache hit", st.calls)
	}
}

func TestResolveUnknownCodeIsNotCached(t *testing.T) {
	svc, _, c, _ := cachedService(t)

	_, err := svc.Resolve(context.Background(), "nope")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
	if c.puts != 0 || len(c.links) != 0 {
		t.Errorf("a store error was cached: %+v", c.links)
	}
}

func TestResolveCachedLinkStillExpires(t *testing.T) {
	svc, st, c, now := cachedService(t)
	l := shorten(t, svc, ShortenRequest{
		Target: "https://example.com", OwnerID: "u1", ExpiresAt: cacheEpoch.Add(time.Hour),
	})

	if _, err := svc.Resolve(context.Background(), l.Code); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.links[l.Code]; !ok {
		t.Fatal("link was not cached")
	}

	st.calls = 0
	*now = cacheEpoch.Add(time.Hour) // expiry is exclusive: expired at exactly ExpiresAt
	if _, err := svc.Resolve(context.Background(), l.Code); !errors.Is(err, ErrExpired) {
		t.Errorf("err = %v, want ErrExpired", err)
	}
	if st.calls != 0 {
		t.Errorf("the expired link was read from the store, not the cache")
	}
}

func TestResolveExpiredOnStoreHit(t *testing.T) {
	svc, _, c, now := cachedService(t)
	l := shorten(t, svc, ShortenRequest{
		Target: "https://example.com", OwnerID: "u1", ExpiresAt: cacheEpoch.Add(time.Minute),
	})

	*now = cacheEpoch.Add(time.Hour)
	if _, err := svc.Resolve(context.Background(), l.Code); !errors.Is(err, ErrExpired) {
		t.Errorf("err = %v, want ErrExpired", err)
	}
	if c.gets != 1 {
		t.Errorf("cache consulted %d times, want 1", c.gets)
	}
}

func TestResolveCancelledContextSkipsCache(t *testing.T) {
	svc, st, c, _ := cachedService(t)
	c.links["cached"] = Link{Code: "cached", Target: "https://example.com"}

	_, err := svc.Resolve(cancelledContext(t), "cached")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if c.gets != 0 || st.calls != 0 {
		t.Errorf("cache gets %d, store calls %d; want 0, 0", c.gets, st.calls)
	}
}

func TestDeleteEvictsFromCache(t *testing.T) {
	svc, _, c, _ := cachedService(t)
	l := shorten(t, svc, ShortenRequest{Target: "https://example.com", OwnerID: "u1"})
	if _, err := svc.Resolve(context.Background(), l.Code); err != nil {
		t.Fatal(err)
	}

	if err := svc.Delete(context.Background(), "u1", l.Code); err != nil {
		t.Fatal(err)
	}
	if c.dels != 1 {
		t.Errorf("cache deletes = %d, want 1", c.dels)
	}
	if _, ok := c.links[l.Code]; ok {
		t.Error("link still cached after Delete")
	}
}

// failingStore is a spyStore whose Delete fails.
type failingStore struct{ *spyStore }

var errStoreDown = errors.New("store down")

func (failingStore) Delete(context.Context, uint64) error { return errStoreDown }

func TestDeleteFailureLeavesCacheAlone(t *testing.T) {
	cached := Link{ID: 1, Code: "000001", Target: "https://example.com", OwnerID: "u1"}

	tests := []struct {
		name    string
		store   Store
		owner   string
		code    string
		wantErr error
	}{
		{"empty owner", newSpyStore(), "", cached.Code, ErrEmptyOwner},
		{"unknown code", newSpyStore(), "u1", "nope", ErrNotFound},
		{"another owner", newSpyStore(), "u2", cached.Code, ErrForbidden},
		{"store error", failingStore{newSpyStore()}, "u1", cached.Code, errStoreDown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newMapCache()
			svc := NewService(tt.store, WithCache(c))
			if tt.code == cached.Code {
				if _, err := tt.store.Create(context.Background(), cached); err != nil {
					t.Fatal(err)
				}
			}
			c.links[cached.Code] = cached

			err := svc.Delete(context.Background(), tt.owner, tt.code)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("err = %v, want %v", err, tt.wantErr)
			}
			if c.dels != 0 || c.links[cached.Code] != cached {
				t.Errorf("cache changed: %d deletes, links %+v", c.dels, c.links)
			}
		})
	}
}

func TestDeleteOwnershipReadsStoreNotCache(t *testing.T) {
	svc, _, c, _ := cachedService(t)
	l := shorten(t, svc, ShortenRequest{Target: "https://example.com", OwnerID: "u1"})

	// A stale cache entry claims u2 owns the link; the store says u1.
	stale := l
	stale.OwnerID = "u2"
	c.links[l.Code] = stale

	if err := svc.Delete(context.Background(), "u2", l.Code); !errors.Is(err, ErrForbidden) {
		t.Errorf("Delete by u2 = %v, want ErrForbidden", err)
	}
	if err := svc.Delete(context.Background(), "u1", l.Code); err != nil {
		t.Errorf("Delete by u1 = %v, want nil", err)
	}
	if c.gets != 0 {
		t.Errorf("Delete consulted the cache %d times", c.gets)
	}
}

func TestListDoesNotTouchCache(t *testing.T) {
	svc, _, c, _ := cachedService(t)
	if _, err := svc.List(context.Background(), "u1", Page{}); err != nil {
		t.Fatal(err)
	}
	if c.gets+c.puts+c.dels != 0 {
		t.Errorf("List touched the cache: %+v", c)
	}
}

func TestNoCacheBehavesAsBefore(t *testing.T) {
	st := newSpyStore()
	svc := NewService(st)
	l := shorten(t, svc, ShortenRequest{Target: "https://example.com", OwnerID: "u1"})

	st.calls = 0
	for range 2 {
		if _, err := svc.Resolve(context.Background(), l.Code); err != nil {
			t.Fatal(err)
		}
	}
	if st.calls != 2 {
		t.Errorf("store calls = %d, want 2 (every Resolve reads the store)", st.calls)
	}
	if err := svc.Delete(context.Background(), "u1", l.Code); err != nil {
		t.Errorf("Delete without a cache: %v", err)
	}
}

package link

// Cache is the lookup cache Resolve consults before the store. It is declared
// here, by its consumer; *cache.LRU[string, Link] from pkg/cache satisfies it.
type Cache interface {
	Get(code string) (Link, bool)
	Put(code string, l Link)
	Delete(code string)
}

package storage

import (
	"github.com/dgraph-io/ristretto"
)

/*
 * Silly & minimal interface to a cache of fragments - this hides a lot of
 * features, but should make some testing and benchmarking easier by providing
 * a way to tune or disable the cache.
 *
 * It achieves two things:
 * 1. ease-of-testing through custom cache implementations
 * 2. automates the casting, forcing the cache to only store the cacheentry
 *    type, which is way less annoying than dealing with interface{}
 */
type blobCache interface {
	set(string, entity)
	get(string) (entity, bool)
}

type ristrettoCache struct {
	ristretto.Cache
}
func (c *ristrettoCache) set(key string, val entity) {
	c.Set(key, val, 0)
}
func (c *ristrettoCache) get(key string) (val entity, hit bool) {
	v, hit := c.Get(key)
	if hit {
		val = v.(entity)
	}
	return
}

/*
 * The nocache isn't really used per now, but serves as a useful reference and
 * available information for tests runs or test cases that wants to disable
 * caching altogether.
 */
type noCache struct {}
func (c *noCache) set(key string, val entity) {}
func (c *noCache) get(key string) (entity, bool) {
	return entity{}, false
}

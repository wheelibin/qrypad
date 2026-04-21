package db

import "sync"

type tableInfoKey struct {
	Ref  TableReference
	Kind string
}

// SchemaCache is a session-scoped, concurrency-safe cache for schema metadata.
// Call Invalidate to clear all entries (e.g. on connection or database change).
type SchemaCache struct {
	mu        sync.RWMutex
	tableInfo map[tableInfoKey]*Data
	entities  map[string]*Data
}

// NewSchemaCache returns an empty, ready-to-use SchemaCache.
func NewSchemaCache() *SchemaCache {
	return &SchemaCache{
		tableInfo: make(map[tableInfoKey]*Data),
		entities:  make(map[string]*Data),
	}
}

// GetTableInfo returns cached table info data for (ref, kind). ok is false on miss.
func (c *SchemaCache) GetTableInfo(ref TableReference, kind string) (*Data, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.tableInfo[tableInfoKey{Ref: ref, Kind: kind}]
	return v, ok
}

// SetTableInfo stores data for (ref, kind) in the cache.
func (c *SchemaCache) SetTableInfo(ref TableReference, kind string, data *Data) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tableInfo[tableInfoKey{Ref: ref, Kind: kind}] = data
}

// GetEntities returns cached schema entity data for kind ("tables" or "views"). ok is false on miss.
func (c *SchemaCache) GetEntities(kind string) (*Data, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.entities[kind]
	return v, ok
}

// SetEntities stores schema entity data for kind in the cache.
func (c *SchemaCache) SetEntities(kind string, data *Data) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entities[kind] = data
}

// Invalidate clears all cached entries.
func (c *SchemaCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tableInfo = make(map[tableInfoKey]*Data)
	c.entities = make(map[string]*Data)
}

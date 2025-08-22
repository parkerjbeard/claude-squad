package ui

import (
	"strings"
	"sync"
)

// A pooled strings.Builder to reduce allocations in hot render paths.
// Builders are reused across renders and reset before reuse.
var builderPool = sync.Pool{
	New: func() any { return new(strings.Builder) },
}

const maxPooledBuilderCapacity = 64 * 1024 // 64KiB safety cap to avoid memory bloat

func getBuilder() *strings.Builder {
	b := builderPool.Get().(*strings.Builder)
	b.Reset()
	return b
}

func putBuilder(b *strings.Builder) {
	// Avoid keeping extremely large backing arrays in the pool
	if b.Cap() > maxPooledBuilderCapacity {
		return
	}
	b.Reset()
	builderPool.Put(b)
}

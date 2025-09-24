// Package pool provides efficient object pooling for go-snap CLI parsing
// Used by parser for reusing expensive allocations and reducing GC pressure
package pool

import (
	"sync"
	"time"
)

// Pool provides a generic, type-safe object pool with automatic cleanup
type Pool[T any] struct {
	pool    sync.Pool
	reset   func(*T)    // Optional reset function called before reuse
	cleanup func(*T)    // Optional cleanup function for pool eviction
	maxSize int         // Maximum objects to keep (0 = unlimited)
	count   int64       // Current pool size (approximate)
	mutex   sync.RWMutex // Protects count
}

// NewPool creates a new generic pool with the given factory function
func NewPool[T any](factory func() *T) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any {
				return factory()
			},
		},
		maxSize: 0, // Unlimited by default
	}
}

// NewPoolWithReset creates a pool with a reset function called before reuse
func NewPoolWithReset[T any](factory func() *T, reset func(*T)) *Pool[T] {
	p := NewPool(factory)
	p.reset = reset
	return p
}

// Get retrieves an object from the pool or creates a new one
func (p *Pool[T]) Get() *T {
	obj := p.pool.Get().(*T)
	if p.reset != nil {
		p.reset(obj)
	}
	return obj
}

// Put returns an object to the pool for reuse
func (p *Pool[T]) Put(obj *T) {
	if obj == nil {
		return
	}

	// Check max size limit
	if p.maxSize > 0 {
		p.mutex.RLock()
		current := p.count
		p.mutex.RUnlock()

		if current >= int64(p.maxSize) {
			if p.cleanup != nil {
				p.cleanup(obj)
			}
			return
		}
	}

	p.pool.Put(obj)

	if p.maxSize > 0 {
		p.mutex.Lock()
		p.count++
		p.mutex.Unlock()
	}
}

// SetMaxSize sets the maximum number of objects to keep in the pool
func (p *Pool[T]) SetMaxSize(size int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.maxSize = size
}

// Stats returns approximate pool statistics
func (p *Pool[T]) Stats() (count int64, maxSize int) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.count, p.maxSize
}

// BufferPool provides a specialized pool for byte slices with capacity management
type BufferPool struct {
	pools map[int]*Pool[[]byte] // Pools by capacity bucket
	mutex sync.RWMutex

	// Configuration
	minCap    int   // Minimum capacity
	maxCap    int   // Maximum capacity
	buckets   []int // Capacity buckets
	defaultCap int  // Default capacity
}

// NewBufferPool creates a new buffer pool with capacity-based buckets
func NewBufferPool() *BufferPool {
	buckets := []int{64, 128, 256, 512, 1024, 2048, 4096}

	bp := &BufferPool{
		pools:     make(map[int]*Pool[[]byte]),
		minCap:    64,
		maxCap:    4096,
		buckets:   buckets,
		defaultCap: 256,
	}

	// Initialize pools for each bucket
	for _, cap := range buckets {
		capacity := cap // Capture for closure
		bp.pools[capacity] = NewPoolWithReset(
			func() *[]byte {
				buf := make([]byte, 0, capacity)
				return &buf
			},
			func(buf *[]byte) {
				*buf = (*buf)[:0] // Reset length but keep capacity
			},
		)
	}

	return bp
}

// Get retrieves a buffer with at least the requested capacity
func (bp *BufferPool) Get(minCap int) *[]byte {
	capacity := bp.findBucket(minCap)

	bp.mutex.RLock()
	pool, exists := bp.pools[capacity]
	bp.mutex.RUnlock()

	if !exists {
		// Create buffer directly if outside bucket range
		buf := make([]byte, 0, minCap)
		return &buf
	}

	return pool.Get()
}

// Put returns a buffer to the appropriate pool
func (bp *BufferPool) Put(buf *[]byte) {
	if buf == nil {
		return
	}

	capacity := cap(*buf)

	// Only pool if within our bucket range
	if capacity < bp.minCap || capacity > bp.maxCap {
		return
	}

	bucketCap := bp.findBucket(capacity)

	bp.mutex.RLock()
	pool, exists := bp.pools[bucketCap]
	bp.mutex.RUnlock()

	if exists {
		pool.Put(buf)
	}
}

// findBucket finds the appropriate capacity bucket for the given size
func (bp *BufferPool) findBucket(minCap int) int {
	for _, bucket := range bp.buckets {
		if bucket >= minCap {
			return bucket
		}
	}
	return bp.maxCap
}

// StringSlicePool provides efficient pooling for string slices
type StringSlicePool struct {
	*Pool[[]string]
}

// NewStringSlicePool creates a new string slice pool
func NewStringSlicePool(defaultCap int) *StringSlicePool {
	return &StringSlicePool{
		Pool: NewPoolWithReset(
			func() *[]string {
				slice := make([]string, 0, defaultCap)
				return &slice
			},
			func(slice *[]string) {
				*slice = (*slice)[:0] // Reset length but keep capacity
			},
		),
	}
}

// IntSlicePool provides efficient pooling for int slices
type IntSlicePool struct {
	*Pool[[]int]
}

// NewIntSlicePool creates a new int slice pool
func NewIntSlicePool(defaultCap int) *IntSlicePool {
	return &IntSlicePool{
		Pool: NewPoolWithReset(
			func() *[]int {
				slice := make([]int, 0, defaultCap)
				return &slice
			},
			func(slice *[]int) {
				*slice = (*slice)[:0] // Reset length but keep capacity
			},
		),
	}
}

// ParseResultPool provides specialized pooling for ParseResult objects
type ParseResultPool struct {
	*Pool[ParseResult]
}

// ParseResult represents the parser result structure (simplified for pooling)
type ParseResult struct {
	// Typed maps to avoid interface{} boxing allocations
	IntFlags      map[string]int
	StringFlags   map[string]string
	BoolFlags     map[string]bool
	DurationFlags map[string]time.Duration
	FloatFlags    map[string]float64
	EnumFlags     map[string]string

	// Slice storage using offsets into global buffers
	StringSliceOffsets map[string]SliceOffset
	IntSliceOffsets    map[string]SliceOffset

	// Global flag typed maps
	GlobalIntFlags           map[string]int
	GlobalStringFlags        map[string]string
	GlobalBoolFlags          map[string]bool
	GlobalDurationFlags      map[string]time.Duration
	GlobalFloatFlags         map[string]float64
	GlobalEnumFlags          map[string]string
	GlobalStringSliceOffsets map[string]SliceOffset
	GlobalIntSliceOffsets    map[string]SliceOffset

	Args []string
}

// SliceOffset tracks start and end positions in global buffers
type SliceOffset struct {
	Start int
	End   int
}

// NewParseResultPool creates a new ParseResult pool
func NewParseResultPool() *ParseResultPool {
	return &ParseResultPool{
		Pool: NewPoolWithReset(
			func() *ParseResult {
				return &ParseResult{
					// Typed maps to avoid interface{} boxing
					IntFlags:           make(map[string]int, 8),
					StringFlags:        make(map[string]string, 8),
					BoolFlags:          make(map[string]bool, 8),
					DurationFlags:      make(map[string]time.Duration, 4),
					FloatFlags:         make(map[string]float64, 4),
					EnumFlags:          make(map[string]string, 4),
					StringSliceOffsets: make(map[string]SliceOffset, 4),
					IntSliceOffsets:    make(map[string]SliceOffset, 4),

					GlobalIntFlags:           make(map[string]int, 4),
					GlobalStringFlags:        make(map[string]string, 4),
					GlobalBoolFlags:          make(map[string]bool, 4),
					GlobalDurationFlags:      make(map[string]time.Duration, 2),
					GlobalFloatFlags:         make(map[string]float64, 2),
					GlobalEnumFlags:          make(map[string]string, 2),
					GlobalStringSliceOffsets: make(map[string]SliceOffset, 2),
					GlobalIntSliceOffsets:    make(map[string]SliceOffset, 2),

					Args: make([]string, 0, 8),
				}
			},
			func(result *ParseResult) {
				// Clear all maps without reallocating
				clearMap(result.IntFlags)
				clearMap(result.StringFlags)
				clearMap(result.BoolFlags)
				clearMap(result.DurationFlags)
				clearMap(result.FloatFlags)
				clearMap(result.EnumFlags)
				clearMap(result.StringSliceOffsets)
				clearMap(result.IntSliceOffsets)

				clearMap(result.GlobalIntFlags)
				clearMap(result.GlobalStringFlags)
				clearMap(result.GlobalBoolFlags)
				clearMap(result.GlobalDurationFlags)
				clearMap(result.GlobalFloatFlags)
				clearMap(result.GlobalEnumFlags)
				clearMap(result.GlobalStringSliceOffsets)
				clearMap(result.GlobalIntSliceOffsets)

				result.Args = result.Args[:0]
			},
		),
	}
}

// clearMap efficiently clears a map without reallocating
func clearMap[K comparable, V any](m map[K]V) {
	for k := range m {
		delete(m, k)
	}
}

// Global pool instances for CLI parsing
var (
	// Global buffer pool for parser temporary allocations
	GlobalBufferPool = NewBufferPool()

	// Global string slice pool for CLI arguments and flag values
	GlobalStringSlicePool = NewStringSlicePool(32)

	// Global int slice pool for numeric flag values
	GlobalIntSlicePool = NewIntSlicePool(16)

	// Global ParseResult pool for parser results
	GlobalParseResultPool = NewParseResultPool()
)

// init pre-warms the global pools for optimal CLI performance
func init() {
	// Pre-warm buffer pool with common CLI parsing sizes
	for i := 0; i < 5; i++ {
		buf := GlobalBufferPool.Get(256)
		GlobalBufferPool.Put(buf)
	}

	// Pre-warm slice pools
	for i := 0; i < 3; i++ {
		strSlice := GlobalStringSlicePool.Get()
		GlobalStringSlicePool.Put(strSlice)

		intSlice := GlobalIntSlicePool.Get()
		GlobalIntSlicePool.Put(intSlice)

		result := GlobalParseResultPool.Get()
		GlobalParseResultPool.Put(result)
	}
}

// Convenience functions for common CLI parsing use cases

// GetBuffer retrieves a buffer for temporary CLI parsing operations
func GetBuffer(minCap int) *[]byte {
	return GlobalBufferPool.Get(minCap)
}

// PutBuffer returns a buffer to the global pool
func PutBuffer(buf *[]byte) {
	GlobalBufferPool.Put(buf)
}

// GetStringSlice retrieves a string slice for CLI arguments
func GetStringSlice() *[]string {
	return GlobalStringSlicePool.Get()
}

// PutStringSlice returns a string slice to the global pool
func PutStringSlice(slice *[]string) {
	GlobalStringSlicePool.Put(slice)
}

// GetIntSlice retrieves an int slice for numeric CLI values
func GetIntSlice() *[]int {
	return GlobalIntSlicePool.Get()
}

// PutIntSlice returns an int slice to the global pool
func PutIntSlice(slice *[]int) {
	GlobalIntSlicePool.Put(slice)
}

// GetParseResult retrieves a ParseResult for CLI parsing
func GetParseResult() *ParseResult {
	return GlobalParseResultPool.Get()
}

// PutParseResult returns a ParseResult to the global pool
func PutParseResult(result *ParseResult) {
	GlobalParseResultPool.Put(result)
}
package pool

import (
    "runtime"
    "sync"
    "testing"
    "time"
)

func TestPool_Basic(t *testing.T) {
	pool := NewPool(func() *int {
		x := 42
		return &x
	})

	// Test Get
	obj1 := pool.Get()
	if *obj1 != 42 {
		t.Errorf("Expected 42, got %d", *obj1)
	}

	// Modify and Put back
	*obj1 = 100
	pool.Put(obj1)

	// Get again - should be the same object
	obj2 := pool.Get()
	if *obj2 != 100 {
		t.Errorf("Expected reused object with value 100, got %d", *obj2)
	}
}

func TestPool_WithReset(t *testing.T) {
	resetCalled := false
	pool := NewPoolWithReset(
		func() *[]int {
			slice := make([]int, 0, 10)
			return &slice
		},
		func(slice *[]int) {
			*slice = (*slice)[:0]
			resetCalled = true
		},
	)

	// Get and modify
	slice1 := pool.Get()
	*slice1 = append(*slice1, 1, 2, 3)

	// Put back
	pool.Put(slice1)

	// Get again - reset should be called
	slice2 := pool.Get()
	if !resetCalled {
		t.Error("Reset function was not called")
	}
	if len(*slice2) != 0 {
		t.Errorf("Expected empty slice after reset, got length %d", len(*slice2))
	}
}

func TestPool_MaxSize(t *testing.T) {
	pool := NewPool(func() *int {
		x := 0
		return &x
	})
	pool.SetMaxSize(2)

	// Add objects up to max size
	obj1 := pool.Get()
	obj2 := pool.Get()
	obj3 := pool.Get()

	pool.Put(obj1)
	pool.Put(obj2)
	pool.Put(obj3) // This should be discarded due to max size

	count, maxSize := pool.Stats()
	if maxSize != 2 {
		t.Errorf("Expected max size 2, got %d", maxSize)
	}

	// Note: count is approximate and may not be exact due to sync.Pool behavior
	if count > 2 {
		t.Errorf("Expected count <= 2, got %d", count)
	}
}

func TestPool_Concurrent(t *testing.T) {
	pool := NewPool(func() *[]int {
		slice := make([]int, 0, 100)
		return &slice
	})

	const numGoroutines = 50
	const numOperations = 1000

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := range numGoroutines {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := range numOperations {
				// Get object
				obj := pool.Get()
				if obj == nil {
					errors <- nil
					return
				}

				// Modify object
				*obj = append(*obj, goroutineID*1000+j)

				// Put back
				pool.Put(obj)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent operation failed: %v", err)
		}
	}
}

func TestBufferPool_Basic(t *testing.T) {
	bp := NewBufferPool()

	// Test getting different sizes
	tests := []int{64, 128, 256, 512, 1024}

	for _, size := range tests {
		buf := bp.Get(size)
		if cap(*buf) < size {
			t.Errorf("Expected capacity >= %d, got %d", size, cap(*buf))
		}

		// Verify it's empty
		if len(*buf) != 0 {
			t.Errorf("Expected empty buffer, got length %d", len(*buf))
		}

		// Use the buffer
		*buf = append(*buf, make([]byte, size/2)...)

		// Put back
		bp.Put(buf)
	}
}

func TestBufferPool_Reuse(t *testing.T) {
	bp := NewBufferPool()

	// Get a buffer and modify it
	buf1 := bp.Get(256)
	*buf1 = append(*buf1, 1, 2, 3, 4, 5)
	originalCap := cap(*buf1)

	// Put it back
	bp.Put(buf1)

	// Get another buffer of same size - should be reset
	buf2 := bp.Get(256)
	if len(*buf2) != 0 {
		t.Errorf("Expected reset buffer with length 0, got %d", len(*buf2))
	}
	if cap(*buf2) != originalCap {
		t.Errorf("Expected same capacity %d, got %d", originalCap, cap(*buf2))
	}
}

func TestBufferPool_OutOfRange(t *testing.T) {
	bp := NewBufferPool()

	// Test very small buffer (below min)
	buf1 := bp.Get(10)
	if buf1 == nil {
		t.Error("Expected buffer even for small size")
	}

	// Test very large buffer (above max)
	buf2 := bp.Get(10000)
	if buf2 == nil {
		t.Error("Expected buffer even for large size")
	}

	// These shouldn't be pooled when returned
	bp.Put(buf1)
	bp.Put(buf2)
}

func TestStringSlicePool(t *testing.T) {
	pool := NewStringSlicePool(16)

	// Get slice
	slice1 := pool.Get()
	if len(*slice1) != 0 {
		t.Errorf("Expected empty slice, got length %d", len(*slice1))
	}
	if cap(*slice1) < 16 {
		t.Errorf("Expected capacity >= 16, got %d", cap(*slice1))
	}

	// Use slice
	*slice1 = append(*slice1, "hello", "world")

	// Put back
	pool.Put(slice1)

	// Get again - should be reset
	slice2 := pool.Get()
	if len(*slice2) != 0 {
		t.Errorf("Expected reset slice with length 0, got %d", len(*slice2))
	}
}

func TestIntSlicePool(t *testing.T) {
	pool := NewIntSlicePool(16)

	// Get slice
	slice1 := pool.Get()
	if len(*slice1) != 0 {
		t.Errorf("Expected empty slice, got length %d", len(*slice1))
	}
	if cap(*slice1) < 16 {
		t.Errorf("Expected capacity >= 16, got %d", cap(*slice1))
	}

	// Use slice
	*slice1 = append(*slice1, 1, 2, 3, 4, 5)

	// Put back
	pool.Put(slice1)

	// Get again - should be reset
	slice2 := pool.Get()
	if len(*slice2) != 0 {
		t.Errorf("Expected reset slice with length 0, got %d", len(*slice2))
	}
}

func TestParseResultPool(t *testing.T) {
	pool := NewParseResultPool()

	// Get result
	result1 := pool.Get()
	if result1 == nil {
		t.Fatal("Expected non-nil ParseResult")
	}

	// Verify initial state
	if len(result1.StringFlags) != 0 {
		t.Errorf("Expected empty StringFlags map, got %d entries", len(result1.StringFlags))
	}

	// Use result
	result1.StringFlags["test"] = "value"
	result1.IntFlags["count"] = 42
	result1.Args = append(result1.Args, "arg1", "arg2")

	// Put back
	pool.Put(result1)

	// Get again - should be reset
	result2 := pool.Get()
	if len(result2.StringFlags) != 0 {
		t.Errorf("Expected reset StringFlags map, got %d entries", len(result2.StringFlags))
	}
	if len(result2.IntFlags) != 0 {
		t.Errorf("Expected reset IntFlags map, got %d entries", len(result2.IntFlags))
	}
	if len(result2.Args) != 0 {
		t.Errorf("Expected reset Args slice, got %d entries", len(result2.Args))
	}
}

func TestGlobalPools(t *testing.T) {
	// Test global buffer pool
	buf := GetBuffer(512)
	if cap(*buf) < 512 {
		t.Errorf("Expected buffer capacity >= 512, got %d", cap(*buf))
	}
	PutBuffer(buf)

	// Test global string slice pool
	strSlice := GetStringSlice()
	if strSlice == nil {
		t.Error("Expected non-nil string slice")
	}
	PutStringSlice(strSlice)

	// Test global int slice pool
	intSlice := GetIntSlice()
	if intSlice == nil {
		t.Error("Expected non-nil int slice")
	}
	PutIntSlice(intSlice)

	// Test global ParseResult pool
	result := GetParseResult()
	if result == nil {
		t.Error("Expected non-nil ParseResult")
	}
	PutParseResult(result)
}

func TestClearMap(t *testing.T) {
	m := map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
	}

	clearMap(m)

	if len(m) != 0 {
		t.Errorf("Expected empty map after clear, got %d entries", len(m))
	}
}

// Benchmarks moved to benchmark/bench_pool_test.go

// TestMemoryLeaks verifies that pools don't cause memory leaks
func TestMemoryLeaks(t *testing.T) {
	pool := NewPool(func() *[]byte {
		buf := make([]byte, 0, 1024)
		return &buf
	})
	pool.SetMaxSize(10)

	// Create many objects
	for i := range 100 {
		obj := pool.Get()
		*obj = append(*obj, byte(i))
		pool.Put(obj)
	}

	// Force garbage collection
	runtime.GC()
	time.Sleep(10 * time.Millisecond)

	count, _ := pool.Stats()
	if count > 10 {
		t.Errorf("Pool holding too many objects: %d > 10", count)
	}
}

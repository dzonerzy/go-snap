//nolint:testpackage // using package name 'benchmark' to access unexported fields for testing
package benchmark

import (
	"fmt"
	"testing"

	pool "github.com/dzonerzy/go-snap/internal/pool"
)

// Category: pool

func BenchmarkPool_GetPut(b *testing.B) {
	p := pool.NewPool(func() *[]byte {
		buf := make([]byte, 0, 1024)
		return &buf
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			obj := p.Get()
			p.Put(obj)
		}
	})
}

func BenchmarkPool_vs_Direct(b *testing.B) {
	p := pool.NewPool(func() *[]byte {
		buf := make([]byte, 0, 1024)
		return &buf
	})

	b.Run("Pool", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				obj := p.Get()
				// Simulate some work
				*obj = append(*obj, 1, 2, 3, 4, 5)
				p.Put(obj)
			}
		})
	})

	b.Run("Direct", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := make([]byte, 0, 1024)
				// Simulate some work
				buf = append(buf, 1, 2, 3, 4, 5)
				_ = buf
			}
		})
	})
}

func BenchmarkBufferPool_GetPut(b *testing.B) {
	bp := pool.NewBufferPool()

	sizes := []int{64, 256, 1024, 4096}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					buf := bp.Get(size)
					*buf = append(*buf, make([]byte, size/2)...)
					bp.Put(buf)
				}
			})
		})
	}
}

func BenchmarkStringSlicePool(b *testing.B) {
	p := pool.NewStringSlicePool(32)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			slice := p.Get()
			*slice = append(*slice, "command", "arg1", "arg2", "--flag", "value")
			p.Put(slice)
		}
	})
}

func BenchmarkParseResultPool(b *testing.B) {
	p := pool.NewParseResultPool()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result := p.Get()
			result.StringFlags["config"] = "/path/to/config"
			result.IntFlags["port"] = 8080
			result.BoolFlags["verbose"] = true
			result.Args = append(result.Args, "arg1", "arg2")
			p.Put(result)
		}
	})
}

func BenchmarkGlobalPools(b *testing.B) {
	b.Run("BufferPool", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := pool.GetBuffer(512)
				*buf = append(*buf, 1, 2, 3, 4, 5)
				pool.PutBuffer(buf)
			}
		})
	})

	b.Run("StringSlicePool", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				slice := pool.GetStringSlice()
				*slice = append(*slice, "test")
				pool.PutStringSlice(slice)
			}
		})
	})

	b.Run("ParseResultPool", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				result := pool.GetParseResult()
				result.StringFlags["test"] = "value"
				pool.PutParseResult(result)
			}
		})
	})
}

func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("WithPool", func(b *testing.B) {
		p := pool.NewStringSlicePool(16)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			slice := p.Get()
			*slice = append(*slice, "test1", "test2", "test3")
			p.Put(slice)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			slice := make([]string, 0, 16)
			slice = append(slice, "test1", "test2", "test3")
			_ = slice
		}
	})
}

package lru_test

import (
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nathanjcochran/lru"
)

func BenchmarkAdd(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(b.N),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cache.Add(n, n)
	}
}

func BenchmarkAddParallel(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(b.N),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}

	i := int32(-1)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		start := int(atomic.AddInt32(&i, 1)) * (b.N / runtime.GOMAXPROCS(0))
		for n := start; pb.Next(); n++ {
			cache.Add(n, n)
		}
	})
}

func BenchmarkAddEvict(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(1),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cache.Add(n, n)
	}
}

func BenchmarkAddEvictParallel(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(1),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}

	i := int32(-1)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		start := int(atomic.AddInt32(&i, 1)) * (b.N / runtime.GOMAXPROCS(0))
		for n := start; pb.Next(); n++ {
			cache.Add(n, n)
		}
	})
}

func BenchmarkAddExpire(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(b.N),
		lru.SetTTL(1*time.Millisecond),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cache.Add(n, n)
	}
}

func BenchmarkAddExpireParallel(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(b.N),
		lru.SetTTL(1*time.Millisecond),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}

	i := int32(-1)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		start := int(atomic.AddInt32(&i, 1)) * (b.N / runtime.GOMAXPROCS(0))
		for n := start; pb.Next(); n++ {
			cache.Add(n, n)
		}
	})
}

func BenchmarkGet(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(b.N),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}
	for i := 0; i < b.N; i++ {
		cache.Add(i, i)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cache.Get(n)
	}
}

func BenchmarkGetParallel(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(b.N),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}
	for i := 0; i < b.N; i++ {
		cache.Add(i, i)
	}

	i := int32(-1)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		start := int(atomic.AddInt32(&i, 1)) * (b.N / runtime.GOMAXPROCS(0))
		for n := start; pb.Next(); n++ {
			cache.Get(n)
		}
	})
}

func BenchmarkGetExpire(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(b.N),
		lru.SetTTL(1*time.Millisecond),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}
	for i := 0; i < b.N; i++ {
		cache.Add(i, i)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cache.Get(n)
	}
}

func BenchmarkGetExpireParallel(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(b.N),
		lru.SetTTL(1*time.Millisecond),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}
	for i := 0; i < b.N; i++ {
		cache.Add(i, i)
	}

	i := int32(-1)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		start := int(atomic.AddInt32(&i, 1)) * (b.N / runtime.GOMAXPROCS(0))
		for n := start; pb.Next(); n++ {
			cache.Get(n)
		}
	})
}

func BenchmarkContains(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(b.N),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}
	for i := 0; i < b.N; i++ {
		cache.Add(i, i)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cache.Contains(n)
	}
}

func BenchmarkContainsParallel(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(b.N),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}
	for i := 0; i < b.N; i++ {
		cache.Add(i, i)
	}

	i := int32(-1)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		start := int(atomic.AddInt32(&i, 1)) * (b.N / runtime.GOMAXPROCS(0))
		for n := start; pb.Next(); n++ {
			cache.Contains(n)
		}
	})
}

func BenchmarkPeek(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(b.N),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}
	for i := 0; i < b.N; i++ {
		cache.Add(i, i)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cache.Peek(n)
	}
}

func BenchmarkPeekParallel(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(b.N),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}
	for i := 0; i < b.N; i++ {
		cache.Add(i, i)
	}

	i := int32(-1)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		start := int(atomic.AddInt32(&i, 1)) * (b.N / runtime.GOMAXPROCS(0))
		for n := start; pb.Next(); n++ {
			cache.Peek(n)
		}
	})
}

func BenchmarkRemove(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(b.N),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}
	for i := 0; i < b.N; i++ {
		cache.Add(i, i)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cache.Remove(n)
	}
}

func BenchmarkRemoveParallel(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(b.N),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}
	for i := 0; i < b.N; i++ {
		cache.Add(i, i)
	}

	i := int32(-1)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		start := int(atomic.AddInt32(&i, 1)) * (b.N / runtime.GOMAXPROCS(0))
		for n := start; pb.Next(); n++ {
			cache.Remove(n)
		}
	})
}

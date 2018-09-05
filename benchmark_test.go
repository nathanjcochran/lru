package lru_test

import (
	"testing"

	"github.com/nathanjcochran/lru"
)

const size = 1024

func BenchmarkAdd(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(size), // Evictions will occur once cache reaches max size
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
		lru.SetSize(size), // Evictions will occur once cache reaches max size
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for n := 0; pb.Next(); n++ {
			cache.Add(n, n)
		}
	})
}

func BenchmarkAddExisting(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(size),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}
	for i := 0; i < size; i++ {
		cache.Add(i, i)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cache.Add(n%size, n)
	}
}

func BenchmarkAddExistingParallel(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(size),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}
	for i := 0; i < size; i++ {
		cache.Add(i, i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for n := 0; pb.Next(); n++ {
			cache.Add(n%size, n)
		}
	})
}

func BenchmarkGet(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(size),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}
	for i := 0; i < size; i++ {
		cache.Add(i, i)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cache.Get(n % size)
	}
}

func BenchmarkGetParallel(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(size),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}
	for i := 0; i < size; i++ {
		cache.Add(i, i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for n := 0; pb.Next(); n++ {
			cache.Get(n % size)
		}
	})
}

func BenchmarkGetNonExisting(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(size),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cache.Get(n)
	}
}

func BenchmarkGetNonExistingParallel(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(size),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for n := 0; pb.Next(); n++ {
			cache.Get(n)
		}
	})
}

func BenchmarkContains(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(size),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}
	for i := 0; i < size; i++ {
		cache.Add(i, i)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cache.Contains(n % size)
	}
}

func BenchmarkContainsParallel(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(size),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}
	for i := 0; i < size; i++ {
		cache.Add(i, i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for n := 0; pb.Next(); n++ {
			cache.Contains(n % size)
		}
	})
}

func BenchmarkPeek(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(size),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}
	for i := 0; i < size; i++ {
		cache.Add(i, i)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cache.Peek(n % size)
	}
}

func BenchmarkPeekParallel(b *testing.B) {
	cache, err := lru.NewCache()
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}
	for i := 0; i < size; i++ {
		cache.Add(i, i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for n := 0; pb.Next(); n++ {
			cache.Peek(n % size)
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

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for n := 0; pb.Next(); n++ {
			cache.Remove(n)
		}
	})
}

func BenchmarkRemoveNonExisting(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(size),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cache.Remove(n)
	}
}

func BenchmarkRemoveNonExistingParallel(b *testing.B) {
	cache, err := lru.NewCache(
		lru.SetSize(size),
	)
	if err != nil {
		b.Fatalf("Error creating cache: %s", err)
	}
	for i := 0; i < b.N; i++ {
		cache.Add(i, i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for n := 0; pb.Next(); n++ {
			cache.Remove(n)
		}
	})
}

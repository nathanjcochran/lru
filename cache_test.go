package lru_test

import (
	"reflect"
	"testing"

	"github.com/nathanjcochran/lru"
)

func expectEviction(expected, actual bool, t *testing.T) {
	if actual != expected {
		t.Errorf("Expected eviction: %t. Actual: %t", expected, actual)
	}
}

func expectKeys(expected, actual []interface{}, t *testing.T) {
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected keys: %v. Actual: %v", expected, actual)
	}
}

func expectExists(expected, actual bool, t *testing.T) {
	if actual != expected {
		t.Errorf("Expected entry to exist: %t. Actual: %t", expected, actual)
	}
}

func expectValue(expected, actual interface{}, t *testing.T) {
	if actual != expected {
		t.Errorf("Expected value: %s. Actual: %s", expected, actual)
	}
}

func expectLen(expected, actual int, t *testing.T) {
	if actual != expected {
		t.Errorf("Expected length: %d. Actual: %d", expected, actual)
	}
}

func TestAdd(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	eviction := cache.Add("key1", "val1")
	expectEviction(false, eviction, t)

	eviction = cache.Add("key2", "val2")
	expectEviction(false, eviction, t)

	keys := cache.Keys()
	expectKeys([]interface{}{"key1", "key2"}, keys, t)

	val, ok := cache.Peek("key1")
	expectExists(true, ok, t)
	expectValue("val1", val, t)

	val, ok = cache.Peek("key2")
	expectExists(true, ok, t)
	expectValue("val2", val, t)
}

func TestAddExisting(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	eviction := cache.Add("key1", "val1")
	expectEviction(false, eviction, t)

	eviction = cache.Add("key1", "val2")
	expectEviction(false, eviction, t)

	keys := cache.Keys()
	expectKeys([]interface{}{"key1"}, keys, t)

	val, ok := cache.Peek("key1")
	expectExists(true, ok, t)
	expectValue("val2", val, t)

}

func TestAddEvict(t *testing.T) {
	cache, err := lru.NewCache(lru.SetSize(2)) // Third addition will cause eviction
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	eviction := cache.Add("key1", "val1")
	expectEviction(false, eviction, t)

	eviction = cache.Add("key2", "val2")
	expectEviction(false, eviction, t)

	eviction = cache.Add("key3", "val3")
	expectEviction(true, eviction, t)

	keys := cache.Keys()
	expectKeys([]interface{}{"key2", "key3"}, keys, t)

	val, ok := cache.Peek("key1")
	expectExists(false, ok, t)
	expectValue(nil, val, t)
}

func TestAddExistingEvict(t *testing.T) {
	cache, err := lru.NewCache(lru.SetSize(2)) // Third addition will cause eviction
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	eviction := cache.Add("key1", "val1")
	expectEviction(false, eviction, t)

	eviction = cache.Add("key2", "val2")
	expectEviction(false, eviction, t)

	eviction = cache.Add("key1", "val1")
	expectEviction(false, eviction, t)

	eviction = cache.Add("key3", "val3")
	expectEviction(true, eviction, t)

	keys := cache.Keys()
	expectKeys([]interface{}{"key1", "key3"}, keys, t)

	val, ok := cache.Peek("key2")
	expectExists(false, ok, t)
	expectValue(nil, val, t)
}

func TestGet(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	val, ok := cache.Get("key1")
	expectExists(false, ok, t)
	expectValue(nil, val, t)

	eviction := cache.Add("key1", "val1")
	expectEviction(false, eviction, t)

	val, ok = cache.Get("key1")
	expectExists(true, ok, t)
	expectValue("val1", val, t)
}

func TestGetEvict(t *testing.T) {
	cache, err := lru.NewCache(lru.SetSize(2))
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	eviction := cache.Add("key1", "val1")
	expectEviction(false, eviction, t)

	eviction = cache.Add("key2", "val2")
	expectEviction(false, eviction, t)

	val, ok := cache.Get("key1")
	expectExists(true, ok, t)
	expectValue("val1", val, t)

	eviction = cache.Add("key3", "val3")
	expectEviction(true, eviction, t)

	keys := cache.Keys()
	expectKeys([]interface{}{"key1", "key3"}, keys, t)
}

func TestPeek(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	val, ok := cache.Peek("key1")
	expectExists(false, ok, t)
	expectValue(nil, val, t)

	eviction := cache.Add("key1", "val1")
	expectEviction(false, eviction, t)

	val, ok = cache.Peek("key1")
	expectExists(true, ok, t)
	expectValue("val1", val, t)
}

func TestContains(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	ok := cache.Contains("key1")
	expectExists(false, ok, t)

	eviction := cache.Add("key1", "val1")
	expectEviction(false, eviction, t)

	ok = cache.Contains("key1")
	expectExists(true, ok, t)
}

func TestRemove(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	ok := cache.Remove("key1")
	expectExists(false, ok, t)

	eviction := cache.Add("key1", "val1")
	expectEviction(false, eviction, t)

	ok = cache.Remove("key1")
	expectExists(true, ok, t)

	val, ok := cache.Peek("key1")
	expectExists(false, ok, t)
	expectValue(nil, val, t)
}

func TestKeys(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	keys := cache.Keys()
	expectKeys([]interface{}{}, keys, t)

	eviction := cache.Add("key1", "val1")
	expectEviction(false, eviction, t)

	keys = cache.Keys()
	expectKeys([]interface{}{"key1"}, keys, t)

	eviction = cache.Add("key1", "val1")
	expectEviction(false, eviction, t)

	keys = cache.Keys()
	expectKeys([]interface{}{"key1"}, keys, t)

	eviction = cache.Add("key2", "val2")
	expectEviction(false, eviction, t)

	keys = cache.Keys()
	expectKeys([]interface{}{"key1", "key2"}, keys, t)

	eviction = cache.Add("key3", "val3")
	expectEviction(false, eviction, t)

	keys = cache.Keys()
	expectKeys([]interface{}{"key1", "key2", "key3"}, keys, t)

	val, ok := cache.Get("key1")
	expectExists(true, ok, t)
	expectValue("val1", val, t)

	keys = cache.Keys()
	expectKeys([]interface{}{"key2", "key3", "key1"}, keys, t)

	val, ok = cache.Get("key3")
	expectExists(true, ok, t)
	expectValue("val3", val, t)

	keys = cache.Keys()
	expectKeys([]interface{}{"key2", "key1", "key3"}, keys, t)

	ok = cache.Remove("key1")
	expectExists(true, ok, t)

	keys = cache.Keys()
	expectKeys([]interface{}{"key2", "key3"}, keys, t)

	ok = cache.Remove("key1")
	expectExists(false, ok, t)

	keys = cache.Keys()
	expectKeys([]interface{}{"key2", "key3"}, keys, t)

	ok = cache.Remove("key2")
	expectExists(true, ok, t)

	keys = cache.Keys()
	expectKeys([]interface{}{"key3"}, keys, t)

	ok = cache.Remove("key3")
	expectExists(true, ok, t)

	keys = cache.Keys()
	expectKeys([]interface{}{}, keys, t)
}

func TestLen(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	l := cache.Len()
	expectLen(0, l, t)

	cache.Add("key1", "val1")

	l = cache.Len()
	expectLen(1, l, t)

	eviction := cache.Add("key1", "val1")
	expectEviction(false, eviction, t)

	l = cache.Len()
	expectLen(1, l, t)

	eviction = cache.Add("key2", "val2")
	expectEviction(false, eviction, t)

	l = cache.Len()
	expectLen(2, l, t)

	eviction = cache.Add("key3", "val3")
	expectEviction(false, eviction, t)

	l = cache.Len()
	expectLen(3, l, t)

	val, ok := cache.Get("key1")
	expectExists(true, ok, t)
	expectValue("val1", val, t)

	l = cache.Len()
	expectLen(3, l, t)

	val, ok = cache.Get("key3")
	expectExists(true, ok, t)
	expectValue("val3", val, t)

	l = cache.Len()
	expectLen(3, l, t)

	ok = cache.Remove("key1")
	expectExists(true, ok, t)

	l = cache.Len()
	expectLen(2, l, t)

	ok = cache.Remove("key1")
	expectExists(false, ok, t)

	l = cache.Len()
	expectLen(2, l, t)

	ok = cache.Remove("key2")
	expectExists(true, ok, t)

	l = cache.Len()
	expectLen(1, l, t)

	ok = cache.Remove("key3")
	expectExists(true, ok, t)

	l = cache.Len()
	expectLen(0, l, t)
}

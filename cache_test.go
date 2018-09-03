package lru_test

import (
	"reflect"
	"testing"

	"github.com/nathanjcochran/lru"
)

func TestAdd(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	if eviction := cache.Add("key1", "val1"); eviction {
		t.Errorf("Expected eviction: false. Actual: %t", eviction)
	}

	expectedKeys := []interface{}{"key1"}
	if keys := cache.Keys(); !reflect.DeepEqual(keys, expectedKeys) {
		t.Errorf("Expected keys: %v. Actual: %v", expectedKeys, keys)
	}

	if actual, ok := cache.Peek("key1"); !ok {
		t.Errorf("Expected item to exist in cache")
	} else if actual != "val1" {
		t.Errorf("Expected item: %s. Actual: %s", "val2", actual)
	}
}

func TestAddExisting(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	if eviction := cache.Add("key1", "val1"); eviction {
		t.Errorf("Expected eviction: false. Actual: %t", eviction)
	}

	if eviction := cache.Add("key1", "val2"); eviction {
		t.Errorf("Expected eviction: false. Actual: %t", eviction)
	}

	expectedKeys := []interface{}{"key1"}
	if keys := cache.Keys(); !reflect.DeepEqual(keys, expectedKeys) {
		t.Errorf("Expected keys: %v. Actual: %v", expectedKeys, keys)
	}

	if actual, ok := cache.Peek("key1"); !ok {
		t.Errorf("Expected item to exist in cache")
	} else if actual != "val2" {
		t.Errorf("Expected item: %s. Actual: %s", "val2", actual)
	}
}

func TestAddEvict(t *testing.T) {
	cache, err := lru.NewCache(lru.SetSize(2))
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	if eviction := cache.Add("key1", "value1"); eviction {
		t.Errorf("Expected eviction: false. Actual: %t", eviction)
	}

	if eviction := cache.Add("key2", "value2"); eviction {
		t.Errorf("Expected eviction: false. Actual: %t", eviction)
	}

	if eviction := cache.Add("key3", "value3"); !eviction {
		t.Errorf("Expected eviction: true. Actual: %t", eviction)
	}

	expectedKeys := []interface{}{"key2", "key3"}
	if keys := cache.Keys(); !reflect.DeepEqual(keys, expectedKeys) {
		t.Errorf("Expected keys: %v. Actual: %v", expectedKeys, keys)
	}
}

func TestAddExistingEvict(t *testing.T) {
	cache, err := lru.NewCache(lru.SetSize(2))
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	if eviction := cache.Add("key1", "value1"); eviction {
		t.Errorf("Expected eviction: false. Actual: %t", eviction)
	}

	if eviction := cache.Add("key2", "value2"); eviction {
		t.Errorf("Expected eviction: false. Actual: %t", eviction)
	}

	if eviction := cache.Add("key1", "value3"); eviction {
		t.Errorf("Expected eviction: false. Actual: %t", eviction)
	}

	if eviction := cache.Add("key3", "value4"); !eviction {
		t.Errorf("Expected eviction: true. Actual: %t", eviction)
	}

	expectedKeys := []interface{}{"key1", "key3"}
	if keys := cache.Keys(); !reflect.DeepEqual(keys, expectedKeys) {
		t.Errorf("Expected keys: %v. Actual: %v", expectedKeys, keys)
	}
}

func TestGet(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	var (
		key = "key1"
		val = "val1"
	)
	if actual, ok := cache.Get(key); ok {
		t.Errorf("Expected item to exist: false. Actual: %t", ok)
	} else if actual != nil {
		t.Errorf("Expected item: nil. Actual: %s", actual)
	}

	cache.Add(key, val)

	if actual, ok := cache.Get(key); !ok {
		t.Errorf("Expected item to exist: true. Actual: %t", ok)
	} else if actual != val {
		t.Errorf("Expected item: %s. Actual: %s", val, actual)
	}
}

func TestGetEvict(t *testing.T) {
	cache, err := lru.NewCache(lru.SetSize(2))
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	cache.Add("key1", "value1")
	cache.Add("key2", "value2")
	cache.Get("key1")
	cache.Add("key3", "value3")

	expected := []interface{}{"key1", "key3"}
	keys := cache.Keys()
	if !reflect.DeepEqual(keys, expected) {
		t.Errorf("Expected keys: %v. Actual: %v", expected, keys)
	}
}

func TestPeek(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	var (
		key = "key"
		val = "value"
	)
	if actual, ok := cache.Peek(key); ok {
		t.Errorf("Expected item exist: false. Actual: %t", actual)
	} else if actual != nil {
		t.Errorf("Expected item: nil. Actual: %s", actual)
	}

	cache.Add(key, val)

	if actual, ok := cache.Peek(key); !ok {
		t.Errorf("Expected item to exist: true. Actual: %t", ok)
	} else if actual != val {
		t.Errorf("Expected item: %s. Actual: %s", val, actual)
	}
}

func TestContains(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	var (
		key = "key"
		val = "value"
	)
	if ok := cache.Contains(key); ok {
		t.Errorf("Expected item exist: false. Actual: %t", ok)
	}

	cache.Add(key, val)

	if ok := cache.Contains(key); !ok {
		t.Errorf("Expected item exist: true. Actual: %t", ok)
	}
}

func TestRemove(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	var (
		key = "key"
		val = "value"
	)
	if ok := cache.Remove(key); ok {
		t.Errorf("Expected item to exist: false. Actual: %t", ok)
	}

	cache.Add(key, val)

	if ok := cache.Remove(key); !ok {
		t.Errorf("Expected item to exist: false. Actual: %t", ok)
	}

	if actual, ok := cache.Peek(key); ok {
		t.Errorf("Expected item to exist: false. Actual: %t", actual)
	} else if actual != nil {
		t.Errorf("Expected item: nil. Actual: %s", actual)
	}
}

func TestKeys(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	expectedKeys := []interface{}{}
	if keys := cache.Keys(); !reflect.DeepEqual(keys, expectedKeys) {
		t.Errorf("Expected keys: %v. Actual: %v", expectedKeys, keys)
	}

	cache.Add("key1", "value1")
	expectedKeys = []interface{}{"key1"}
	if keys := cache.Keys(); !reflect.DeepEqual(keys, expectedKeys) {
		t.Errorf("Expected keys: %v. Actual: %v", expectedKeys, keys)
	}

	cache.Add("key1", "value1")
	expectedKeys = []interface{}{"key1"}
	if keys := cache.Keys(); !reflect.DeepEqual(keys, expectedKeys) {
		t.Errorf("Expected keys: %v. Actual: %v", expectedKeys, keys)
	}

	cache.Add("key2", "value2")
	cache.Add("key3", "value3")
	cache.Add("key4", "value4")
	cache.Add("key5", "value5")
	expectedKeys = []interface{}{"key1", "key2", "key3", "key4", "key5"}
	if keys := cache.Keys(); !reflect.DeepEqual(keys, expectedKeys) {
		t.Errorf("Expected keys: %v. Actual: %v", expectedKeys, keys)
	}

	cache.Get("key5")
	cache.Get("key4")
	cache.Get("key3")
	cache.Get("key2")
	cache.Get("key1")
	expectedKeys = []interface{}{"key5", "key4", "key3", "key2", "key1"}
	if keys := cache.Keys(); !reflect.DeepEqual(keys, expectedKeys) {
		t.Errorf("Expected keys: %v. Actual: %v", expectedKeys, keys)
	}

	cache.Remove("key1")
	expectedKeys = []interface{}{"key5", "key4", "key3", "key2"}
	if keys := cache.Keys(); !reflect.DeepEqual(keys, expectedKeys) {
		t.Errorf("Expected keys: %v. Actual: %v", expectedKeys, keys)
	}

	cache.Remove("key1")
	expectedKeys = []interface{}{"key5", "key4", "key3", "key2"}
	if keys := cache.Keys(); !reflect.DeepEqual(keys, expectedKeys) {
		t.Errorf("Expected keys: %v. Actual: %v", expectedKeys, keys)
	}

	cache.Remove("key2")
	cache.Remove("key3")
	cache.Remove("key4")
	cache.Remove("key5")
	expectedKeys = []interface{}{}
	if keys := cache.Keys(); !reflect.DeepEqual(keys, expectedKeys) {
		t.Errorf("Expected keys: %v. Actual: %v", expectedKeys, keys)
	}
}

func TestLen(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}

	if size := cache.Len(); size != 0 {
		t.Errorf("Expected size: 0. Actual: %d", size)
	}

	cache.Add("key1", "value1")
	if size := cache.Len(); size != 1 {
		t.Errorf("Expected size: 1. Actual: %d", size)
	}

	cache.Add("key1", "value1")
	if size := cache.Len(); size != 1 {
		t.Errorf("Expected size: 1. Actual: %d", size)
	}

	cache.Add("key2", "value2")
	cache.Add("key3", "value3")
	cache.Add("key4", "value4")
	cache.Add("key5", "value5")
	if size := cache.Len(); size != 5 {
		t.Errorf("Expected size: 5. Actual: %d", size)
	}

	cache.Remove("key1")
	if size := cache.Len(); size != 4 {
		t.Errorf("Expected size: 4. Actual: %d", size)
	}

	cache.Remove("key1")
	if size := cache.Len(); size != 4 {
		t.Errorf("Expected size: 4. Actual: %d", size)
	}

	cache.Remove("key2")
	cache.Remove("key3")
	cache.Remove("key4")
	cache.Remove("key5")
	if size := cache.Len(); size != 0 {
		t.Errorf("Expected size: 0. Actual: %d", size)
	}
}

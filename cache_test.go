package lru_test

import (
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nathanjcochran/lru"
)

func expectAddEviction(expected, actual bool, t *testing.T) {
	if actual != expected {
		t.Errorf("Expected eviction: %t. Actual: %t", expected, actual)
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

func expectKeys(expected, actual []interface{}, t *testing.T) {
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected keys: %v. Actual: %v", expected, actual)
	}
}

func expectLen(expected, actual int, t *testing.T) {
	if actual != expected {
		t.Errorf("Expected length: %d. Actual: %d", expected, actual)
	}
}

type keyVal struct {
	key, val interface{}
}

func expectEvicted(expected int, actual *int, t *testing.T) {
	if *actual != expected {
		t.Errorf("Expected evicted: %d. Actual: %d", expected, *actual)
	}
}

func expectEviction(t *testing.T, expected ...keyVal) (*int, lru.Callback) {
	var evicted int
	return &evicted, func(key, val interface{}) {
		if len(expected) < evicted {
			t.Errorf("Did not expect entry to be evicted. Key: %v. Value: %v", key, val)
			return
		}

		kv := expected[evicted]
		if key != kv.key {
			t.Errorf("Expected key to be evicted: %v. Actual: %v", kv.key, key)
		}
		if val != kv.val {
			t.Errorf("Expected val to be evicted: %v. Actual: %v", kv.val, val)
		}
		evicted++
	}
}

func expectExpired(expected int, actual *int32, t *testing.T) {
	local := atomic.LoadInt32(actual)
	if int(local) != expected {
		t.Errorf("Expected expired: %d. Actual: %d", expected, local)
	}
}

func expectExpiration(t *testing.T, expected ...keyVal) (*int32, lru.Callback) {
	var expired int32
	return &expired, func(key, val interface{}) {
		local := int(atomic.AddInt32(&expired, 1)) - 1

		if len(expected) < local {
			t.Errorf("Did not expect entry to expire. Key: %v. Value: %v", key, val)
			return
		}

		kv := expected[local]
		if key != kv.key {
			t.Errorf("Expected key to expire: %v. Actual: %v", kv.key, key)
		}
		if val != kv.val {
			t.Errorf("Expected val to expire: %v. Actual: %v", kv.val, val)
		}
	}
}

type updateVal struct {
	key, newVal interface{}
	status      lru.Status
}

func expectUpdate(t *testing.T, expected ...updateVal) (*int32, lru.UpdateFunc) {
	var expired int32
	return &expired, func(key interface{}) (interface{}, lru.Status) {
		local := int(atomic.AddInt32(&expired, 1)) - 1

		if len(expected) <= local {
			t.Errorf("Did not expect entry to expire. Key: %v", key)
			return nil, lru.Remove
		}

		uv := expected[local]
		if key != uv.key {
			t.Errorf("Expected key to expire: %v. Actual: %v", uv.key, key)
		}

		return uv.newVal, uv.status
	}
}

func TestAdd(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	eviction := cache.Add("key1", "val1")
	expectAddEviction(false, eviction, t)

	eviction = cache.Add("key2", "val2")
	expectAddEviction(false, eviction, t)

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
	defer cache.Stop()

	eviction := cache.Add("key1", "val1")
	expectAddEviction(false, eviction, t)

	eviction = cache.Add("key1", "val2")
	expectAddEviction(false, eviction, t)

	keys := cache.Keys()
	expectKeys([]interface{}{"key1"}, keys, t)

	val, ok := cache.Peek("key1")
	expectExists(true, ok, t)
	expectValue("val2", val, t)

}

func TestAddEvict(t *testing.T) {
	evicted, callback := expectEviction(t,
		keyVal{"key1", "val1"},
	)
	cache, err := lru.NewCache(
		lru.SetSize(2), // Third addition will cause eviction
		lru.SetOnEvict(callback),
	)
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	eviction := cache.Add("key1", "val1")
	expectAddEviction(false, eviction, t)
	expectEvicted(0, evicted, t)

	eviction = cache.Add("key2", "val2")
	expectAddEviction(false, eviction, t)
	expectEvicted(0, evicted, t)

	eviction = cache.Add("key3", "val3")
	expectAddEviction(true, eviction, t)
	expectEvicted(1, evicted, t)

	keys := cache.Keys()
	expectKeys([]interface{}{"key2", "key3"}, keys, t)

	val, ok := cache.Peek("key1")
	expectExists(false, ok, t)
	expectValue(nil, val, t)
}

func TestAddExistingEvict(t *testing.T) {
	evicted, callback := expectEviction(t,
		keyVal{"key2", "val2"},
	)
	cache, err := lru.NewCache(
		lru.SetSize(2), // Third addition will cause eviction
		lru.SetOnEvict(callback),
	)
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	eviction := cache.Add("key1", "val1")
	expectAddEviction(false, eviction, t)
	expectEvicted(0, evicted, t)

	eviction = cache.Add("key2", "val2")
	expectAddEviction(false, eviction, t)
	expectEvicted(0, evicted, t)

	eviction = cache.Add("key1", "val1")
	expectAddEviction(false, eviction, t)
	expectEvicted(0, evicted, t)

	eviction = cache.Add("key3", "val3")
	expectAddEviction(true, eviction, t)
	expectEvicted(1, evicted, t)

	keys := cache.Keys()
	expectKeys([]interface{}{"key1", "key3"}, keys, t)

	val, ok := cache.Peek("key2")
	expectExists(false, ok, t)
	expectValue(nil, val, t)
}

func TestExpiration(t *testing.T) {
	expired, callback := expectExpiration(t,
		keyVal{"key1", "val1"},
		keyVal{"key2", "val2"},
	)
	cache, err := lru.NewCache(
		lru.SetTTL(100*time.Millisecond),      // Entries will expire after 100ms
		lru.SetTTLMargin(10*time.Millisecond), // Entries can expire up to 10ms early (i.e. after 90ms)
		lru.SetOnExpire(callback),
	)
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	eviction := cache.Add("key1", "val1")
	expectAddEviction(false, eviction, t)

	time.Sleep(50 * time.Millisecond)

	eviction = cache.Add("key2", "val2")
	expectAddEviction(false, eviction, t)

	expectExpired(0, expired, t)

	keys := cache.Keys()
	expectKeys([]interface{}{"key1", "key2"}, keys, t)

	time.Sleep(75 * time.Millisecond)

	expectExpired(1, expired, t)

	keys = cache.Keys()
	expectKeys([]interface{}{"key2"}, keys, t)

	val, ok := cache.Peek("key1")
	expectExists(false, ok, t)
	expectValue(nil, val, t)

	time.Sleep(50 * time.Millisecond)

	expectExpired(2, expired, t)

	keys = cache.Keys()
	expectKeys([]interface{}{}, keys, t)

	val, ok = cache.Peek("key2")
	expectExists(false, ok, t)
	expectValue(nil, val, t)
}

func testUpdate(workers int, t *testing.T) {
	expired, updateFunc := expectUpdate(t,
		updateVal{"key1", "val4", lru.Update},
		updateVal{"key2", nil, lru.Pass},
		updateVal{"key3", nil, lru.Remove},
	)
	cache, err := lru.NewCache(
		lru.SetWorkers(workers),
		lru.SetTTL(150*time.Millisecond),      // Entries will expire after 150ms
		lru.SetTTLMargin(10*time.Millisecond), // Entries can expire up to 10ms early (i.e. after 140ms)
		lru.SetUpdateFunc(updateFunc),
	)
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	eviction := cache.Add("key1", "val1")
	expectAddEviction(false, eviction, t)

	time.Sleep(50 * time.Millisecond)

	eviction = cache.Add("key2", "val2")
	expectAddEviction(false, eviction, t)

	expectExpired(0, expired, t)

	keys := cache.Keys()
	expectKeys([]interface{}{"key1", "key2"}, keys, t)

	time.Sleep(50 * time.Millisecond)

	eviction = cache.Add("key3", "val3")
	expectAddEviction(false, eviction, t)

	expectExpired(0, expired, t)

	keys = cache.Keys()
	expectKeys([]interface{}{"key1", "key2", "key3"}, keys, t)

	time.Sleep(75 * time.Millisecond)

	expectExpired(1, expired, t)

	keys = cache.Keys()
	expectKeys([]interface{}{"key1", "key2", "key3"}, keys, t)

	val, ok := cache.Peek("key1")
	expectExists(true, ok, t)
	expectValue("val4", val, t)

	time.Sleep(50 * time.Millisecond)

	expectExpired(2, expired, t)

	keys = cache.Keys()
	expectKeys([]interface{}{"key1", "key2", "key3"}, keys, t)

	val, ok = cache.Peek("key2")
	expectExists(true, ok, t)
	expectValue("val2", val, t)

	time.Sleep(50 * time.Millisecond)

	expectExpired(3, expired, t)

	keys = cache.Keys()
	expectKeys([]interface{}{"key1", "key2"}, keys, t)

	val, ok = cache.Peek("key3")
	expectExists(false, ok, t)
	expectValue(nil, val, t)
}

func TestUpdate(t *testing.T) {
	numWorkers := []int{0, 1, 2}
	for _, workers := range numWorkers {
		testUpdate(workers, t)
	}
}

// TODO: Test Update with full buffer (OnBufferFull)
// TODO: Test Update when entry has been removed, gotten, or added during interval

func TestGet(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	val, ok := cache.Get("key1")
	expectExists(false, ok, t)
	expectValue(nil, val, t)

	eviction := cache.Add("key1", "val1")
	expectAddEviction(false, eviction, t)

	val, ok = cache.Get("key1")
	expectExists(true, ok, t)
	expectValue("val1", val, t)
}

func TestGetEvict(t *testing.T) {
	evicted, callback := expectEviction(t,
		keyVal{"key2", "val2"},
	)
	cache, err := lru.NewCache(
		lru.SetSize(2), // Third addition will cause eviction
		lru.SetOnEvict(callback),
	)
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	eviction := cache.Add("key1", "val1")
	expectAddEviction(false, eviction, t)
	expectEvicted(0, evicted, t)

	eviction = cache.Add("key2", "val2")
	expectAddEviction(false, eviction, t)
	expectEvicted(0, evicted, t)

	val, ok := cache.Get("key1")
	expectExists(true, ok, t)
	expectValue("val1", val, t)

	eviction = cache.Add("key3", "val3")
	expectAddEviction(true, eviction, t)
	expectEvicted(1, evicted, t)

	keys := cache.Keys()
	expectKeys([]interface{}{"key1", "key3"}, keys, t)
}

func TestPeek(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	val, ok := cache.Peek("key1")
	expectExists(false, ok, t)
	expectValue(nil, val, t)

	eviction := cache.Add("key1", "val1")
	expectAddEviction(false, eviction, t)

	val, ok = cache.Peek("key1")
	expectExists(true, ok, t)
	expectValue("val1", val, t)
}

func TestContains(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	ok := cache.Contains("key1")
	expectExists(false, ok, t)

	eviction := cache.Add("key1", "val1")
	expectAddEviction(false, eviction, t)

	ok = cache.Contains("key1")
	expectExists(true, ok, t)
}

func TestRemove(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	ok := cache.Remove("key1")
	expectExists(false, ok, t)

	eviction := cache.Add("key1", "val1")
	expectAddEviction(false, eviction, t)

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
	defer cache.Stop()

	keys := cache.Keys()
	expectKeys([]interface{}{}, keys, t)

	eviction := cache.Add("key1", "val1")
	expectAddEviction(false, eviction, t)

	keys = cache.Keys()
	expectKeys([]interface{}{"key1"}, keys, t)

	eviction = cache.Add("key1", "val1")
	expectAddEviction(false, eviction, t)

	keys = cache.Keys()
	expectKeys([]interface{}{"key1"}, keys, t)

	eviction = cache.Add("key2", "val2")
	expectAddEviction(false, eviction, t)

	keys = cache.Keys()
	expectKeys([]interface{}{"key1", "key2"}, keys, t)

	eviction = cache.Add("key3", "val3")
	expectAddEviction(false, eviction, t)

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
	defer cache.Stop()

	l := cache.Len()
	expectLen(0, l, t)

	cache.Add("key1", "val1")

	l = cache.Len()
	expectLen(1, l, t)

	eviction := cache.Add("key1", "val1")
	expectAddEviction(false, eviction, t)

	l = cache.Len()
	expectLen(1, l, t)

	eviction = cache.Add("key2", "val2")
	expectAddEviction(false, eviction, t)

	l = cache.Len()
	expectLen(2, l, t)

	eviction = cache.Add("key3", "val3")
	expectAddEviction(false, eviction, t)

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

package lru_test

import (
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nathanjcochran/lru"
)

func TestAdd(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	eviction := cache.Add("key1", "val1")
	expectEviction(false, eviction, t)

	eviction = cache.Add("key2", "val2")
	expectEviction(false, eviction, t)

	expectKeys([]interface{}{"key1", "key2"}, cache.Keys(), t)

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
	expectEviction(false, eviction, t)

	eviction = cache.Add("key1", "val2")
	expectEviction(false, eviction, t)

	expectKeys([]interface{}{"key1"}, cache.Keys(), t)

	val, ok := cache.Peek("key1")
	expectExists(true, ok, t)
	expectValue("val2", val, t)
}

func TestAddEvict(t *testing.T) {
	evicted, onEvict := expectOnEvict(t,
		keyVal{"key1", "val1"},
	)
	cache, err := lru.NewCache(
		lru.SetSize(2), // Third addition will cause eviction
		lru.SetOnEvict(onEvict),
	)
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	eviction := cache.Add("key1", "val1")
	expectEviction(false, eviction, t)
	expectOnEvictCalled(0, evicted, t)

	eviction = cache.Add("key2", "val2")
	expectEviction(false, eviction, t)
	expectOnEvictCalled(0, evicted, t)

	eviction = cache.Add("key3", "val3") // key1 is evicted
	expectEviction(true, eviction, t)
	expectOnEvictCalled(1, evicted, t)

	expectKeys([]interface{}{"key2", "key3"}, cache.Keys(), t)

	val, ok := cache.Peek("key1")
	expectExists(false, ok, t)
	expectValue(nil, val, t)
}

func TestAddExistingEvict(t *testing.T) {
	evicted, onEvict := expectOnEvict(t,
		keyVal{"key2", "val2"},
	)
	cache, err := lru.NewCache(
		lru.SetSize(2), // Third addition will cause eviction
		lru.SetOnEvict(onEvict),
	)
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	eviction := cache.Add("key1", "val1")
	expectEviction(false, eviction, t)
	expectOnEvictCalled(0, evicted, t)

	eviction = cache.Add("key2", "val2")
	expectEviction(false, eviction, t)
	expectOnEvictCalled(0, evicted, t)

	eviction = cache.Add("key1", "val1") // key2 becomes least recently used
	expectEviction(false, eviction, t)
	expectOnEvictCalled(0, evicted, t)

	eviction = cache.Add("key3", "val3") // key2 is evicted
	expectEviction(true, eviction, t)
	expectOnEvictCalled(1, evicted, t)

	expectKeys([]interface{}{"key1", "key3"}, cache.Keys(), t)

	val, ok := cache.Peek("key2")
	expectExists(false, ok, t)
	expectValue(nil, val, t)
}

func TestExpiration(t *testing.T) {
	expired, onEvict := expectOnExpire(t,
		keyVal{"key1", "val1"},
		keyVal{"key2", "val2"},
	)
	cache, err := lru.NewCache(
		lru.SetTTL(100*time.Millisecond),      // Entries will expire after 100ms
		lru.SetTTLMargin(10*time.Millisecond), // Entries can expire up to 10ms early (i.e. after 90ms)
		lru.SetOnExpire(onEvict),
	)
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	cache.Add("key1", "val1")

	time.Sleep(50 * time.Millisecond) // 50ms (one added)

	cache.Add("key2", "val2")
	expectOnExpireCalled(0, expired, t)
	expectKeys([]interface{}{"key1", "key2"}, cache.Keys(), t)

	time.Sleep(75 * time.Millisecond) // 125ms (two added, one expired)

	expectOnExpireCalled(1, expired, t)
	expectKeys([]interface{}{"key2"}, cache.Keys(), t)

	val, ok := cache.Peek("key1")
	expectExists(false, ok, t)
	expectValue(nil, val, t)

	time.Sleep(50 * time.Millisecond) // 175ms (two expired)

	expectOnExpireCalled(2, expired, t)
	expectKeys([]interface{}{}, cache.Keys(), t)

	val, ok = cache.Peek("key2")
	expectExists(false, ok, t)
	expectValue(nil, val, t)
}

func TestUpdate(t *testing.T) {
	numWorkers := []int{0, 1, 2}
	for _, workers := range numWorkers {
		testUpdate(workers, t)
	}
}

func testUpdate(workers int, t *testing.T) {
	updated, updateFunc := expectUpdateFunc(t,
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

	cache.Add("key1", "val1")

	time.Sleep(50 * time.Millisecond) // 50ms (one added)

	cache.Add("key2", "val2")
	expectUpdateFuncCalled(0, updated, t)
	expectKeys([]interface{}{"key1", "key2"}, cache.Keys(), t)

	time.Sleep(50 * time.Millisecond) // 100ms (two added)

	cache.Add("key3", "val3")
	expectUpdateFuncCalled(0, updated, t)
	expectKeys([]interface{}{"key1", "key2", "key3"}, cache.Keys(), t)

	time.Sleep(75 * time.Millisecond) // 175ms (three added, one updated)

	expectUpdateFuncCalled(1, updated, t)
	expectKeys([]interface{}{"key1", "key2", "key3"}, cache.Keys(), t)

	val, ok := cache.Peek("key1")
	expectExists(true, ok, t)
	expectValue("val4", val, t)

	time.Sleep(50 * time.Millisecond) // 225ms (three added, one updated, one passed)

	expectUpdateFuncCalled(2, updated, t)
	expectKeys([]interface{}{"key1", "key2", "key3"}, cache.Keys(), t)

	val, ok = cache.Peek("key2")
	expectExists(true, ok, t)
	expectValue("val2", val, t)

	time.Sleep(50 * time.Millisecond) // 275ms (three added, one updated, one passed, one removed)

	expectUpdateFuncCalled(3, updated, t)
	expectKeys([]interface{}{"key1", "key2"}, cache.Keys(), t)

	val, ok = cache.Peek("key3")
	expectExists(false, ok, t)
	expectValue(nil, val, t)
}

func TestUpdateThreshold(t *testing.T) {
	numWorkers := []int{0, 1, 2}
	for _, workers := range numWorkers {
		testUpdateThreshold(workers, t)
	}
}

func testUpdateThreshold(workers int, t *testing.T) {
	expired, onExpire := expectOnExpire(t,
		keyVal{"key3", "val3"},
	)
	updated, updateFunc := expectUpdateFunc(t,
		updateVal{"key1", "val4", lru.Update},
		updateVal{"key2", nil, lru.Pass},
	)
	cache, err := lru.NewCache(
		lru.SetWorkers(workers),
		lru.SetTTL(150*time.Millisecond),      // Entries will expire after 150ms
		lru.SetTTLMargin(10*time.Millisecond), // Entries can expire up to 10ms early (i.e. after 140ms)
		lru.SetUpdateThreshold(1),             // Entries will only be updated if they were fetched at least once
		lru.SetOnExpire(onExpire),
		lru.SetUpdateFunc(updateFunc),
	)
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	cache.Add("key1", "val1")

	time.Sleep(50 * time.Millisecond) // 50ms (one added)

	cache.Get("key1")
	cache.Add("key2", "val2")
	expectOnExpireCalled(0, expired, t)
	expectUpdateFuncCalled(0, updated, t)
	expectKeys([]interface{}{"key1", "key2"}, cache.Keys(), t)

	time.Sleep(50 * time.Millisecond) // 100ms (two added, one hit),

	cache.Get("key2")
	cache.Add("key3", "val3")
	expectOnExpireCalled(0, expired, t)
	expectUpdateFuncCalled(0, updated, t)
	expectKeys([]interface{}{"key1", "key2", "key3"}, cache.Keys(), t)

	time.Sleep(75 * time.Millisecond) // 175ms (three added, two hit, one updated)

	expectOnExpireCalled(0, expired, t)
	expectUpdateFuncCalled(1, updated, t)
	expectKeys([]interface{}{"key1", "key2", "key3"}, cache.Keys(), t)

	val, ok := cache.Peek("key1")
	expectExists(true, ok, t)
	expectValue("val4", val, t)

	time.Sleep(50 * time.Millisecond) // 225ms (three added, two hit, one updated, one passed)

	expectOnExpireCalled(0, expired, t)
	expectUpdateFuncCalled(2, updated, t)
	expectKeys([]interface{}{"key1", "key2", "key3"}, cache.Keys(), t)

	val, ok = cache.Peek("key2")
	expectExists(true, ok, t)
	expectValue("val2", val, t)

	time.Sleep(50 * time.Millisecond) // 275ms (three added, two hit, one updated, one passed, one expired)

	expectOnExpireCalled(1, expired, t)
	expectUpdateFuncCalled(2, updated, t)
	expectKeys([]interface{}{"key1", "key2"}, cache.Keys(), t)

	val, ok = cache.Peek("key3")
	expectExists(false, ok, t)
	expectValue(nil, val, t)

}

func TestUpdateWithConcurrentRemove(t *testing.T) {
	numWorkers := []int{0, 1, 2}
	for _, workers := range numWorkers {
		testUpdateWithConcurrentRemove(workers, t)
	}
}

func testUpdateWithConcurrentRemove(workers int, t *testing.T) {
	updateFunc := func(key interface{}) (interface{}, lru.Status) {
		time.Sleep(50 * time.Millisecond) // Updates take 50ms
		return "val2", lru.Update
	}
	cache, err := lru.NewCache(
		lru.SetWorkers(workers),
		lru.SetTTL(50*time.Millisecond),       // Entries will expire after 50ms
		lru.SetTTLMargin(10*time.Millisecond), // Entries can expire up to 10ms early (i.e. after 40ms)
		lru.SetUpdateFunc(updateFunc),
	)
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	cache.Add("key1", "val1")

	time.Sleep(75 * time.Millisecond) // 75ms (one added, one ongoing update)

	cache.Remove("key1")

	time.Sleep(50 * time.Millisecond) // 125ms (one added, one removed during update)

	expectKeys([]interface{}{}, cache.Keys(), t)

	val, ok := cache.Peek("key1")
	expectExists(false, ok, t)
	expectValue(nil, val, t)
}

func TestUpdateWithConcurrentAdd(t *testing.T) {
	numWorkers := []int{0, 1, 2}
	for _, workers := range numWorkers {
		testUpdateWithConcurrentAdd(workers, t)
	}
}

func testUpdateWithConcurrentAdd(workers int, t *testing.T) {
	updateFunc := func(key interface{}) (interface{}, lru.Status) {
		time.Sleep(50 * time.Millisecond) // Updates take 50ms
		return "val3", lru.Update
	}
	cache, err := lru.NewCache(
		lru.SetWorkers(workers),
		lru.SetTTL(50*time.Millisecond),       // Entries will expire after 50ms
		lru.SetTTLMargin(10*time.Millisecond), // Entries can expire up to 10ms early (i.e. after 40ms)
		lru.SetUpdateFunc(updateFunc),
	)
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	cache.Add("key1", "val1")

	time.Sleep(75 * time.Millisecond) // 75ms (one added, one ongoing update)

	cache.Add("key1", "val2")

	time.Sleep(50 * time.Millisecond) // 125ms (one added, one re-added during update)

	keys := cache.Keys()
	expectKeys([]interface{}{"key1"}, keys, t)

	val, ok := cache.Peek("key1")
	expectExists(true, ok, t)
	expectValue("val3", val, t)
}

func TestUpdateWithFullBuffer(t *testing.T) {
	updateFunc := func(key interface{}) (interface{}, lru.Status) {
		time.Sleep(50 * time.Millisecond) // Updates take 50ms
		return "val4", lru.Update
	}
	done := make(chan struct{})
	onBufferFull := func(key, val interface{}) {
		if key != "key3" {
			t.Errorf("Expected key: %v. Actual: %v", "key3", key)
		}
		if val != "val3" {
			t.Errorf("Expected val: %v. Actual: %v", "val3", key)
		}
		close(done)
	}
	cache, err := lru.NewCache(
		lru.SetWorkers(1),
		lru.SetTTL(50*time.Millisecond),       // Entries will expire after 50ms
		lru.SetTTLMargin(10*time.Millisecond), // Entries can expire up to 10ms early (i.e. after 40ms)
		lru.SetUpdateFunc(updateFunc),
		lru.SetBufferSize(1), // Buffer size is 1
		lru.SetOnBufferFull(onBufferFull),
	)
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	cache.Add("key1", "val1") // update is triggered
	cache.Add("key2", "val2") // queues in buffer
	cache.Add("key3", "val3") // overflows buffer

	select {
	case <-done:
	case <-time.After(75 * time.Millisecond):
		t.Errorf("Expected onBufferFull to be called")
	}
}

func TestGet(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	val, ok := cache.Get("key1")
	expectExists(false, ok, t)
	expectValue(nil, val, t)

	cache.Add("key1", "val1")

	val, ok = cache.Get("key1")
	expectExists(true, ok, t)
	expectValue("val1", val, t)
}

func TestGetEvict(t *testing.T) {
	evicted, onEvict := expectOnEvict(t,
		keyVal{"key2", "val2"},
	)
	cache, err := lru.NewCache(
		lru.SetSize(2), // Third addition will cause eviction
		lru.SetOnEvict(onEvict),
	)
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	eviction := cache.Add("key1", "val1")
	expectEviction(false, eviction, t)
	expectOnEvictCalled(0, evicted, t)
	expectKeys([]interface{}{"key1"}, cache.Keys(), t)

	eviction = cache.Add("key2", "val2")
	expectEviction(false, eviction, t)
	expectOnEvictCalled(0, evicted, t)
	expectKeys([]interface{}{"key1", "key2"}, cache.Keys(), t)

	val, ok := cache.Get("key1") // key2 becomes least recently used
	expectExists(true, ok, t)
	expectValue("val1", val, t)
	expectKeys([]interface{}{"key2", "key1"}, cache.Keys(), t)

	eviction = cache.Add("key3", "val3") // key2 is evicted
	expectEviction(true, eviction, t)
	expectOnEvictCalled(1, evicted, t)
	expectKeys([]interface{}{"key1", "key3"}, cache.Keys(), t)

	val, ok = cache.Peek("key2")
	expectExists(false, ok, t)
	expectValue(nil, val, t)
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

	cache.Add("key1", "val1")

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

	cache.Add("key1", "val1")

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

	cache.Add("key1", "val1")

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

	expectKeys([]interface{}{}, cache.Keys(), t)

	cache.Add("key1", "val1")
	expectKeys([]interface{}{"key1"}, cache.Keys(), t)

	cache.Add("key1", "val1")
	expectKeys([]interface{}{"key1"}, cache.Keys(), t)

	cache.Add("key2", "val2")
	expectKeys([]interface{}{"key1", "key2"}, cache.Keys(), t)

	cache.Add("key3", "val3")
	expectKeys([]interface{}{"key1", "key2", "key3"}, cache.Keys(), t)

	cache.Get("key1")
	expectKeys([]interface{}{"key2", "key3", "key1"}, cache.Keys(), t)

	cache.Get("key3")
	expectKeys([]interface{}{"key2", "key1", "key3"}, cache.Keys(), t)

	cache.Remove("key1")
	expectKeys([]interface{}{"key2", "key3"}, cache.Keys(), t)

	cache.Remove("key1")
	expectKeys([]interface{}{"key2", "key3"}, cache.Keys(), t)

	cache.Remove("key2")
	expectKeys([]interface{}{"key3"}, cache.Keys(), t)

	cache.Remove("key3")
	expectKeys([]interface{}{}, cache.Keys(), t)
}

func TestLen(t *testing.T) {
	cache, err := lru.NewCache()
	if err != nil {
		t.Fatalf("Error creating cache: %s", err)
	}
	defer cache.Stop()

	expectLen(0, cache.Len(), t)

	cache.Add("key1", "val1")
	expectLen(1, cache.Len(), t)

	cache.Add("key1", "val1")
	expectLen(1, cache.Len(), t)

	cache.Add("key2", "val2")
	expectLen(2, cache.Len(), t)

	cache.Add("key3", "val3")
	expectLen(3, cache.Len(), t)

	cache.Get("key1")
	expectLen(3, cache.Len(), t)

	cache.Get("key3")
	expectLen(3, cache.Len(), t)

	cache.Remove("key1")
	expectLen(2, cache.Len(), t)

	cache.Remove("key1")
	expectLen(2, cache.Len(), t)

	cache.Remove("key2")
	expectLen(1, cache.Len(), t)

	cache.Remove("key3")
	expectLen(0, cache.Len(), t)
}

func expectEviction(expected, actual bool, t *testing.T) {
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
		t.Errorf("Expected value: %v. Actual: %v", expected, actual)
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

func expectOnEvict(t *testing.T, expected ...keyVal) (*int, lru.Callback) {
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

func expectOnEvictCalled(expected int, actual *int, t *testing.T) {
	if *actual != expected {
		t.Errorf("Expected evicted: %d. Actual: %d", expected, *actual)
	}
}

func expectOnExpire(t *testing.T, expected ...keyVal) (*int32, lru.Callback) {
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

func expectOnExpireCalled(expected int, actual *int32, t *testing.T) {
	local := atomic.LoadInt32(actual)
	if int(local) != expected {
		t.Errorf("Expected expired: %d. Actual: %d", expected, local)
	}
}

type updateVal struct {
	key, newVal interface{}
	status      lru.Status
}

func expectUpdateFunc(t *testing.T, expected ...updateVal) (*int32, lru.UpdateFunc) {
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

func expectUpdateFuncCalled(expected int, actual *int32, t *testing.T) {
	local := atomic.LoadInt32(actual)
	if int(local) != expected {
		t.Errorf("Expected updated: %d. Actual: %d", expected, local)
	}
}

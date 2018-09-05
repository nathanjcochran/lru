package lru

import (
	"errors"
	"sync"
	"time"
)

const (
	DefaultSize      = 128
	DefaultTTL       = time.Hour
	DefaultTTLMargin = time.Second
)

type Status int8

const (
	Update Status = iota
	Pass
	Remove
)

// Callback is called when a cache entry is evicted or expires.
type Callback func(key, value interface{})

// UpdateFunc is called when a cache entry expires, and can return an
// updated version to replace the expired entry in the cache.
type UpdateFunc func(key interface{}) (newVal interface{}, status Status)

// Cache is a thread-safe fixed size LRU cache with a worker goroutine
// that checks for expired items and queues updates, which are processed
// out-of-band via separate worker goroutines.
type Cache struct {
	size         int
	ttl          time.Duration
	ttlMargin    time.Duration
	workers      int
	bufSize      int
	items        map[interface{}]*entry
	evtRoot      entry
	expRoot      entry
	onEvict      Callback
	onExpire     Callback
	onBufferFull Callback
	update       UpdateFunc
	lock         *sync.RWMutex // TODO: Does this need to be a pointer?
	updateChan   chan interface{}
	quitChan     chan struct{}
	doneChan     chan struct{}
}

var _ Interface = &Cache{}

type entry struct {
	key              interface{}
	val              interface{}
	expiration       time.Time
	evtNext, evtPrev *entry
	expNext, expPrev *entry
}

func (e *entry) isExpired() bool {
	return e.expiration.Before(time.Now())
}

type option func(c *Cache)

func SetSize(size int) option {
	return func(c *Cache) {
		c.size = size
	}
}

func SetTTL(ttl time.Duration) option {
	return func(c *Cache) {
		c.ttl = ttl
	}
}

func SetTTLMargin(ttlMargin time.Duration) option {
	return func(c *Cache) {
		c.ttlMargin = ttlMargin
	}
}

func SetWorkers(workers int) option {
	return func(c *Cache) {
		c.workers = workers
	}
}

func SetBufferSize(bufSize int) option {
	return func(c *Cache) {
		c.bufSize = bufSize
	}
}

func SetOnEvict(onEvict Callback) option {
	return func(c *Cache) {
		c.onEvict = onEvict
	}
}

func SetOnExpire(onExpire Callback) option {
	return func(c *Cache) {
		c.onExpire = onExpire
	}
}

func SetOnBufferFull(onBufferFull Callback) option {
	return func(c *Cache) {
		c.onBufferFull = onBufferFull
	}
}

func SetUpdateFunc(update UpdateFunc) option {
	return func(c *Cache) {
		c.update = update
	}
}

func NewCache(options ...option) (*Cache, error) {
	c := &Cache{
		size:      DefaultSize,
		ttl:       DefaultTTL,
		ttlMargin: DefaultTTLMargin,
		items:     map[interface{}]*entry{},
		lock:      &sync.RWMutex{},
		quitChan:  make(chan struct{}),
	}
	c.evtRoot.evtNext = &c.evtRoot
	c.evtRoot.evtPrev = &c.evtRoot
	c.expRoot.expNext = &c.expRoot
	c.expRoot.expPrev = &c.expRoot

	for _, option := range options {
		option(c)
	}

	if c.size <= 0 {
		return nil, errors.New("Cache size must be positive")
	}
	if c.ttl <= 0 {
		return nil, errors.New("Entries must have a positive time to live")
	}
	if c.ttlMargin < 0 {
		return nil, errors.New("Entries must have non-negative time to live margin")
	}
	go c.expirationWorker()
	if c.workers > 0 {
		if c.bufSize < 0 {
			return nil, errors.New("Buffer size must be non-negative")
		}

		c.updateChan = make(chan interface{}, c.bufSize)
		for i := 0; i < c.workers; i++ {
			go c.updateWorker()
		}
	}
	c.doneChan = make(chan struct{}, c.workers+1) // +1 for expiration worker

	return c, nil
}

func (c *Cache) Add(key, val interface{}) (evicted bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Check for existing item
	if e, ok := c.items[key]; ok {
		e.val = val
		e.expiration = time.Now().Add(c.ttl)
		c.evtMoveToFront(e)
		c.expMoveToFront(e)
		return false
	}

	// Add new item
	e := &entry{
		key:        key,
		val:        val,
		expiration: time.Now().Add(c.ttl),
	}
	c.evtPushFront(e)
	c.expPushFront(e)
	c.items[key] = e

	// Verify size not exceeded, evict an entry if needed
	if len(c.items) > c.size {
		c.evict()
		return true
	}
	return false
}

// Get looks up a key's value from the cache.
func (c *Cache) Get(key interface{}) (val interface{}, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if e, ok := c.items[key]; ok {
		c.evtMoveToFront(e)
		return e.val, true
	}
	return nil, false
}

func (c *Cache) Contains(key interface{}) (ok bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if _, ok := c.items[key]; ok {
		return true
	}
	return false
}

func (c *Cache) Peek(key interface{}) (val interface{}, ok bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if e, ok := c.items[key]; ok {
		return e.val, true
	}
	return nil, false
}

func (c *Cache) Remove(key interface{}) (ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if e, ok := c.items[key]; ok {
		c.remove(e)
		return true
	}
	return false
}

func (c *Cache) Keys() []interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()

	keys := make([]interface{}, len(c.items))
	var i int
	for e := c.evtBack(); e != &c.evtRoot; e = e.evtPrev {
		keys[i] = e.key
		i++
	}
	return keys
}

func (c *Cache) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return len(c.items)
}

func (c *Cache) Purge() {
	c.lock.Lock()
	defer c.lock.Unlock()

	for k := range c.items {
		delete(c.items, k)
	}
	c.evtRoot.evtNext = &c.evtRoot
	c.evtRoot.evtPrev = &c.evtRoot
	c.expRoot.expNext = &c.expRoot
	c.expRoot.expPrev = &c.expRoot
}

// Stops all worker goroutines. It is not safe to use the cache after a call
// to Stop. Does not purge the cache or release objects from memory.
func (c *Cache) Stop() {
	close(c.quitChan)
	for i := 0; i < c.workers+1; i++ {
		<-c.doneChan
	}
}

func (c *Cache) expirationWorker() {
	timer := time.NewTimer(c.ttl)
	for {
		select {
		case <-timer.C:
			wakeAfter := c.handleExpirations()
			timer.Reset(wakeAfter)
		case <-c.quitChan:
			timer.Stop()
			c.doneChan <- struct{}{}
			return
		}
	}
}

func (c *Cache) handleExpirations() time.Duration {
	for {
		now := time.Now()
		c.lock.Lock()

		// Get the item that expires next
		e := c.expBack()
		if e == &c.expRoot {
			// If there are no items in the cache, wait as long as it would
			// take for an item to expire before checking again
			c.lock.Unlock()
			return c.ttl
		}

		// Check whether the item has expired, within the time to live margin
		if wait := e.expiration.Sub(now); wait-c.ttlMargin > 0 {
			// If it hasn't expired yet, wait until it does
			c.lock.Unlock()
			return wait
		}

		// If the user did not provide an update callback,
		// then just remove the entry from the cache
		if c.update == nil {
			c.remove(e)
			if c.onExpire != nil {
				c.onExpire(e.key, e.val)
			}
			c.lock.Unlock()
			continue
		}

		// Remove the item from the expiration list so we can keep iterating.
		// It will be added back to the front after being updated
		expRemove(e)
		e.expNext = nil // prevent memory leaks, and flag that entry
		e.expPrev = nil // has already been removed from exp list

		// Release the lock to avoid blocking the cache during the update, or
		// deadlocks when the update channel is full
		c.lock.Unlock()

		// If there are no workers, do the update in this goroutine
		if c.workers == 0 {
			c.handleUpdate(e.key)
			continue
		}

		// Queue the update for a worker goroutine to pick up
		if c.onBufferFull == nil {
			c.updateChan <- e.key
			continue
		}

		select {
		case c.updateChan <- e.key:
		default:
			// Inform user that buffer is full via callback, then
			// go back to blocking on send
			c.onBufferFull(e.key, e.val)
			c.updateChan <- e.key
		}
	}
}

func (c *Cache) updateWorker() {
	for {
		select {
		case key := <-c.updateChan:
			c.handleUpdate(key)
		case <-c.quitChan:
			c.doneChan <- struct{}{}
			return
		}
	}
}

func (c *Cache) handleUpdate(key interface{}) {
	// Call user-provided update function:
	newVal, status := c.update(key)

	c.lock.Lock()
	defer c.lock.Unlock()

	e, ok := c.items[key]
	if !ok {
		// Item has been removed since update was queued (either
		// via Remove, Purge, or evict), so do nothing
		return
	}

	// Perform appropriate action based on status returned
	// from update function
	switch status {
	case Update:
		e.val = newVal
		e.expiration = time.Now().Add(c.ttl)
		c.expMoveToFront(e)
	case Pass:
		e.expiration = time.Now().Add(c.ttl)
		c.expMoveToFront(e)
	case Remove:
		c.remove(e)
	}
}

func (c *Cache) evict() {
	e := c.evtBack()
	if e != &c.evtRoot {
		c.remove(e)
		if c.onEvict != nil {
			c.onEvict(e.key, e.val)
		}
	}
}

func (c *Cache) remove(e *entry) {
	evtRemove(e)
	expRemove(e)
	delete(c.items, e.key)
}

func (c *Cache) evtBack() *entry {
	return c.evtRoot.evtPrev
}

func (c *Cache) expBack() *entry {
	return c.expRoot.expPrev
}

func (c *Cache) evtPushFront(e *entry) {
	evtInsert(e, &c.evtRoot)
}

func (c *Cache) expPushFront(e *entry) {
	expInsert(e, &c.expRoot)
}

func (c *Cache) evtMoveToFront(e *entry) {
	evtRemove(e)
	evtInsert(e, &c.evtRoot)
}

func (c *Cache) expMoveToFront(e *entry) {
	expRemove(e)
	expInsert(e, &c.expRoot)
}

func evtInsert(e *entry, at *entry) {
	e.evtNext = at.evtNext
	e.evtPrev = at
	at.evtNext.evtPrev = e
	at.evtNext = e
}

func expInsert(e *entry, at *entry) {
	e.expNext = at.expNext
	e.expPrev = at
	at.expNext.expPrev = e
	at.expNext = e
}

func evtRemove(e *entry) {
	e.evtPrev.evtNext = e.evtNext
	e.evtNext.evtPrev = e.evtPrev
}

func expRemove(e *entry) {
	// exp.Next is nil if the entry has expired and
	// is currently queued for an update
	if e.expNext != nil {
		e.expPrev.expNext = e.expNext
		e.expNext.expPrev = e.expPrev
	}
}

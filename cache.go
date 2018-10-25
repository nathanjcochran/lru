/*
A thread-safe, fixed-size, in-memory LRU cache that supports eviction,
expiration, and out-of-band updates.

For more information, see: https://github.com/nathanjcochran/lru.
*/
package lru

import (
	"errors"
	"sync"
	"time"
)

// Default configuration parameter values
const (
	DefaultSize            = 1024
	DefaultTTL             = time.Hour
	DefaultTTLMargin       = time.Second
	DefaultWorkers         = 1
	DefaultBufSize         = 256
	DefaultUpdateThreshold = 0
)

// Status is a status flag returned by UpdateFunc. Used to indicate how the
// cache should handle expired entries
type Status int8

// Implementations of UpdateFunc must return one of these three Status values
// to indicate how the cache should handle expired entries.
const (
	Update Status = iota // Update the value with the newly provided value
	Pass                 // Leave the existing value unchanged
	Remove               // Remove the item from the cache entirely
)

// UpdateFunc is called when a cache entry expires. It should return an
// updated version of the value corresponding to the provided key, to replace
// the expired entry in the cache. Alternatively, it can trigger the removal
// of the entry from the cache, or opt to leave the item in the cache unchanged
// until its next expiration. See Status.
type UpdateFunc func(key interface{}) (newVal interface{}, status Status)

// Callback is used to indicate that a cache entry expired or was evicted,
// or that the update buffer is full (a sign that the workers were not able
// to update the items as fast as they were expiring). See SetOnEvict,
// SetOnExpire, and SetOnBufferFull.
type Callback func(key, val interface{})

// Cache is a thread-safe fixed-size LRU cache with a single worker goroutine
// that checks for expired items (see SetTTL and SetTTLMargin). If an
// UpdateFunc is provided (see SetUpdateFunc), updates are queued in a buffer
// (see SetBufferSize) and processed out-of-band via separate worker
// goroutines (see SetWorkers). If no UpdateFunc is provided, expired items
// are simply dropped from the cache (in which case, SetBufferSize and
// SetWorkers have no effect). Updates are not performed if the expired entry
// has fewer than a threshold number of hits since the last update (see
// SetUpdateThreshold).
type Cache struct {
	size            int
	ttl             time.Duration
	ttlMargin       time.Duration
	workers         int
	bufSize         int
	updateThreshold int
	items           map[interface{}]*entry
	evtRoot         entry
	expRoot         entry
	onEvict         Callback
	onExpire        Callback
	onBufferFull    Callback
	update          UpdateFunc
	lock            sync.RWMutex
	updateChan      chan interface{}
	quitChan        chan struct{}
	doneChan        chan struct{}
}

var _ Interface = &Cache{}

type entry struct {
	key              interface{}
	val              interface{}
	hits             int
	expiration       time.Time
	evtNext, evtPrev *entry
	expNext, expPrev *entry
}

// Option is a configuration option which can be passed to NewCache to
// customize the operation of the LRU cache.
type Option func(c *Cache)

// SetSize sets the cache size. This is the maximum number of items the
// cache can hold before calls to Add cause the least recently used item
// to be evicted.
func SetSize(size int) Option {
	return func(c *Cache) {
		c.size = size
	}
}

// SetTTL sets the time to live for cache entires. Cache items expire after
// this amount of time, at which point they are either removed from the cache,
// or updated via the user-provided UpdateFunc.
func SetTTL(ttl time.Duration) Option {
	return func(c *Cache) {
		c.ttl = ttl
	}
}

// SetTTLMargin sets the minimum amount of time that the expiration worker
// will put itself to sleep for. If an entry expires sooner than that, the
// worker will expire it before its actual scheduled expiration time, in
// effect shortening its ttl by up to this amount.
func SetTTLMargin(ttlMargin time.Duration) Option {
	return func(c *Cache) {
		c.ttlMargin = ttlMargin
	}
}

// SetWorkers sets the number of update worker groutines (note, however, that
// there is always one expiration worker). This is the maximum number of
// concurrent calls to UpdateFunc, and can be used to throttle/limit floods of
// calls to UpdateFunc. If this is not set high enough, however, the cache
// will not be able to keep up with expirations, causing the buffer to back up
// (see SetOnBufferFull). In general, this number should be greater than
// (size * updateTime) / ttl, where updateTime is the average duration of
// a call to UpdateFunc.
func SetWorkers(workers int) Option {
	return func(c *Cache) {
		c.workers = workers
	}
}

// SetBufferSize sets the maximum size of the update buffer. If items expire
// faster than the workers can update them, they queue up in the buffer. If
// it fills up completely (see SetOnBufferFull), the expiration worker blocks,
// and therefore ceases to be able to expire items or queue updates.
func SetBufferSize(bufSize int) Option {
	return func(c *Cache) {
		c.bufSize = bufSize
	}
}

// SetUpdateThreshold sets the hit threshold for cache updates. If an entry
// has not been hit (i.e. fetched via Get) at least this many times since its
// last addition/update, it will be removed (onExpire will be called), rather
// than incurring the cost of another call to UpdateFunc.
func SetUpdateThreshold(hits int) Option {
	return func(c *Cache) {
		c.updateThreshold = hits
	}
}

// SetOnEvict provides a callback that is called when an item is evicted from
// the cache. Eviction happens when the cache has reached its maximum size
// (see SetSize) and a call is made to Add. The key and value of the evicted
// entry are passed to the callback. It it not called when an item is updated
// or removed from the cache.
func SetOnEvict(onEvict Callback) Option {
	return func(c *Cache) {
		c.onEvict = onEvict
	}
}

// SetOnExpire provides a callback that is called when an item expires from
// the cache. Expiration happens when an item's ttl has passed (see SetTTL)
// but the item is not updated, either because no UpdateFunc was given (see
// SetUpdateFunc) or because the entry did not meet the update hit threshold
// (see SetUpdateThreshold). The key and value of the expired entry are passed
// to the callback. It it not called when an item is updated or removed from
// the cache.
func SetOnExpire(onExpire Callback) Option {
	return func(c *Cache) {
		c.onExpire = onExpire
	}
}

// SetOnBufferFull provides a callback that is called when the update buffer
// reaches its maximum size (see SetBufferSize) and the expiration worker
// cannot proceed. If it is being called often, it may be a good idea to
// increase the number of workers (see SetWorkers) or the time to live (see
// SetTTL), in addition to the buffer size itself. The key and value of item
// which could not be added to the queue are passed to the callback.
func SetOnBufferFull(onBufferFull Callback) Option {
	return func(c *Cache) {
		c.onBufferFull = onBufferFull
	}
}

// SetUpdateFunc provides a callback to be called when an item expires and
// needs to be updated with a new value. The function should return an
// up-to-date version of the value corresponding to the given key, as well as
// a status flag (see Status) indicating whether the entry should be updated
// with the new value (Update), left as-is until the next expiration (Pass),
// or removed from the cache (Remove). Pass is useful if there was an error
// fetching an item, but the error did not indicate that the item no longer
// exists, and availability is more important than returning up-to-date data.
func SetUpdateFunc(update UpdateFunc) Option {
	return func(c *Cache) {
		c.update = update
	}
}

// NewCache returns a new instance of a Cache, initialized with the provided
// configuration options. When the cache is no longer needed, it should be
// stopped via a call to Stop.
func NewCache(options ...Option) (*Cache, error) {
	c := &Cache{
		size:            DefaultSize,
		ttl:             DefaultTTL,
		ttlMargin:       DefaultTTLMargin,
		workers:         DefaultWorkers,
		bufSize:         DefaultBufSize,
		updateThreshold: DefaultUpdateThreshold,
		items:           map[interface{}]*entry{},
		lock:            sync.RWMutex{},
		quitChan:        make(chan struct{}),
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
	if c.updateThreshold < 0 {
		return nil, errors.New("Entries must have non-negative update hit threshold")
	}
	go c.expirationWorker()

	if c.update == nil {
		c.workers = 0
	}
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

// Add inserts an entry into the cache with the specified key and value. If
// the cache is full (see SetSize), this will cause an eviction of the least
// recently used item. Returns a boolean indicating whether or not an eviction
// occurred.
func (c *Cache) Add(key, val interface{}) (evicted bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Check for existing item
	if e, ok := c.items[key]; ok {
		e.val = val
		e.hits = 0
		e.expiration = time.Now().Add(c.ttl)
		c.evtMoveToFront(e)
		c.expMoveToFront(e)
		return false
	}

	// Add new item
	e := &entry{
		key:        key,
		val:        val,
		hits:       0,
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

// Get returns the value corresponding to the provided key, if it exists in
// the cache. Also returns a boolean indicating whether or not it exists.
// The item's position in the list of recently-used items is updated.
func (c *Cache) Get(key interface{}) (val interface{}, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if e, ok := c.items[key]; ok {
		e.hits++
		c.evtMoveToFront(e)
		return e.val, true
	}
	return nil, false
}

// Contains checks to see whether an item exists in the cache. The item's
// position in the list of recently-used items is not updated (see Get).
func (c *Cache) Contains(key interface{}) (ok bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if _, ok := c.items[key]; ok {
		return true
	}
	return false
}

// Peek returns the value corresponding to the provided key, if it exists in
// the cache. Also returns a boolean indicating whether or not it exists. The
// item's position in the list of recently-used items is not updated (see Get).
func (c *Cache) Peek(key interface{}) (val interface{}, ok bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if e, ok := c.items[key]; ok {
		return e.val, true
	}
	return nil, false
}

// Remove deletes an item from the cache, if it exists. Returns a boolean
// indicating whether the item existed in the cache. Does not cause onEvict
// or onExpire to be called.
func (c *Cache) Remove(key interface{}) (ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if e, ok := c.items[key]; ok {
		c.remove(e)
		return true
	}
	return false
}

// Keys returns the keys of all of the entries in the cache, in least
// recently used order (i.e. the least recently used item first).
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

// Len returns the number of items currently in the cache.
func (c *Cache) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return len(c.items)
}

// Purge removes all items from the cache, resets the lists of
// least-recently-used and soon-to-expire items, and drains the update
// queue. Does not stop the background worker goroutines (see Stop).
func (c *Cache) Purge() {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Remove items from map
	for k := range c.items {
		delete(c.items, k)
	}

	// Drain update chan
	c.drainUpdates()

	// Drop all items from evict and expiration lists
	c.evtRoot.evtNext = &c.evtRoot
	c.evtRoot.evtPrev = &c.evtRoot
	c.expRoot.expNext = &c.expRoot
	c.expRoot.expPrev = &c.expRoot
}

// Stop halts all worker goroutines. It is not safe to use the cache after a
// call to Stop. Does not purge the cache or release objects from memory (see
// Purge).
func (c *Cache) Stop() {
	close(c.quitChan)
	for i := 0; i < c.workers+1; i++ { // +1 for expiration worker
		<-c.doneChan
	}
	if c.workers > 0 {
		close(c.updateChan)
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

		// If the user did not provide an update callback, or the entry's hit
		// count does not cross the update threshold, then just remove the
		// entry from the cache
		if c.update == nil || e.hits < c.updateThreshold {
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
			select {
			case c.updateChan <- e.key:
			case <-c.quitChan: // To prevent deadlock if buffer is full but workers have been stopped
				return c.ttl
			}
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
		e.hits = 0
		e.expiration = time.Now().Add(c.ttl)
		c.expMoveToFront(e)
	case Pass:
		e.hits = 0
		e.expiration = time.Now().Add(c.ttl)
		c.expMoveToFront(e)
	case Remove:
		c.remove(e)
	}
}

func (c *Cache) drainUpdates() {
	for {
		select {
		case <-c.updateChan:
		default:
			return
		}
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

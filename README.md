# lru

[![Build Status](https://travis-ci.com/nathanjcochran/lru.svg?branch=master)](https://travis-ci.com/nathanjcochran/lru)
[![Coverage Status](https://coveralls.io/repos/github/nathanjcochran/lru/badge.svg?branch=master)](https://coveralls.io/github/nathanjcochran/lru?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/nathanjcochran/lru)](https://goreportcard.com/report/github.com/nathanjcochran/lru)
[![GoDoc](https://godoc.org/github.com/nathanjcochran/lru?status.svg)](https://godoc.org/github.com/nathanjcochran/lru) 
[![license](https://img.shields.io/github/license/nathanjcochran/lru.svg?maxAge=2592000)](LICENSE)

A thread-safe, fixed-size, in-memory LRU cache that supports eviction,
expiration, and out-of-band updates.

This cache is implemented as a lock-protected map whose values are elements in
*two* doubly linked lists (unlike a traditional LRU, which only uses one). The
first doubly linked list keeps track of upcoming evictions. It orders items by
how recently they have been used. When an item is added to the cache, or
retrieved via the `Get` method, it is moved to the front of this list. When an
item needs to be evicted during a call to `Add` (because the cache is full),
the item chosen for eviction is pulled from the back of this list.

The second doubly linked list keeps track of how soon items will expire. When
items are added to the cache, they are added to the front of this list. A
single worker goroutine - the "expiration worker" - reads from the back. If
the item at the back is expired, it is removed from the cache. If it is not
yet expired, the worker puts itself to sleep until it is scheduled to expire.
Note that the worker may expire items early in order to avoid putting itself
to sleep for very small amounts of time (configurable). A caveat of this
design is that all items in the cache must share the same time-to-live (i.e.
newly added items always expire after existing items). 

If an update function is provided via `SetUpdateFunc`, expired items are
queued in a channel by the expiration worker, rather than simply being removed
from the cache (the default behavior). Updates are handled by a separate pool
of worker goroutines - the "update workers". These workers read from the
channel and call the user-provided update function with the key of an expired
item. Depending on the value of the status flag returned, they either update
the item with a new value, remove the item from the cache, or reset the item's
time-to-live without updating its value. If the item is not removed from the
cache, it is moved back to the front of the expiration list after the update.
Its position in the eviction list is not changed. It is also possible to
configure the cache to only update items if they have received a threshold
number of hits since the last addition/update. This ensures that unused items
are not needlessly incurring the cost of being updated.

Unit tests and benchmarks are provided.

For more information, see the [documentation](https://godoc.org/github.com/nathanjcochran/lru).

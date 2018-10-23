package lru

// Interface represents the interface for a Cache.
type Interface interface {
	Add(key, value interface{}) bool
	Get(key interface{}) (value interface{}, ok bool)
	Contains(key interface{}) (ok bool)
	Peek(key interface{}) (value interface{}, ok bool)
	Remove(key interface{}) bool
	Keys() []interface{}
	Len() int
	Purge()
	Stop()
}

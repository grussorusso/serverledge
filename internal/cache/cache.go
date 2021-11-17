package cache

import (
	"runtime"
	"sync"
	"time"
)

type Item struct {
	Object     interface{}
	Expiration int64
	Age        int64
	mu         sync.RWMutex // handle concurrent update to the age field
}

// Expired Returns true if the item has expired.
func (item *Item) Expired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

const (
	// NoExpiration For use with functions that take an expiration time.
	NoExpiration time.Duration = -1
	// DefaultExpiration For use with functions that take an expiration time. Equivalent to
	// passing in the same expiration duration as was given to New() or
	// NewFrom() when the cache was created (e.g. 5 minutes.)
	DefaultExpiration time.Duration = 2 * time.Second
)

type Cache struct {
	*cache
	// If this is confusing, see the comment at the bottom of New()
}

type cache struct {
	defaultExpiration time.Duration
	items             map[string]*Item
	mu                sync.RWMutex
	onEvicted         func(string, interface{})
	janitor           *janitor
	cacheSize         int // nr items admitted
}

// Set Add an item to the cache, replacing any existing item. If the duration is 0
// (DefaultExpiration), the cache's default expiration time is used. If it is -1
// (NoExpiration), the item never expires.
// thread safe version
func (c *cache) Set(k string, x interface{}, d time.Duration) {
	// "Inlining" of set
	var e int64
	if d == DefaultExpiration {
		d = c.defaultExpiration
	}
	if d > 0 {
		e = time.Now().Add(d).UnixNano()
	}
	c.mu.Lock()
	newItem := Item{
		Object:     x,
		Expiration: e,
		Age:        time.Now().UnixNano(),
	}
	_, found := c.items[k]
	if found { //replace content
		c.items[k] = &newItem
	} else if len(c.items) < c.cacheSize { //we have enough space
		c.items[k] = &newItem
	} else {
		toReplace := c.findLRU() //find least recently used key
		delete(c.items, toReplace)
		c.items[k] = &newItem //insert a new item
	}

	c.mu.Unlock()
}

// findLRU ... Simple linear research to find the least used item into the cache or an expired one
// No thread safe, take lock on cache outside
func (c *cache) findLRU() (key string) {
	now := time.Now().UnixNano()
	toReplace := ""
	currentAge := time.Time{}.UnixNano()
	for k, v := range c.items {
		if (v.Expiration > 0 && v.Expiration < now) || now-v.Age > currentAge {
			//replace an already expired item or the least used
			toReplace = k
			currentAge = now - v.Age
		}
	}
	return toReplace
}

// Get an item from the cache. Returns the item or nil, and a bool indicating
// whether the key was found.
// Refresh the age parameter.
func (c *cache) Get(k string) (interface{}, bool) {
	c.mu.RLock()
	// "Inlining" of get and Expired
	item, found := c.items[k]
	if !found {
		c.mu.RUnlock()
		return nil, false
	}
	if item.Expiration > 0 {
		now := time.Now().UnixNano()
		if now > item.Expiration {
			c.mu.RUnlock()
			return nil, false
		}
	}
	item.mu.Lock()
	// touch the item
	item.Age = time.Now().UnixNano()
	item.mu.Unlock()

	c.mu.RUnlock()

	return item.Object, true
}

// GetWithExpiration returns an item and its expiration time from the cache.
// It returns the item or nil, the expiration time if one is set (if the item
// never expires a zero value for time.Time is returned), and a bool indicating
// whether the key was found.
func (c *cache) GetWithExpiration(k string) (interface{}, time.Time, bool) {
	c.mu.RLock()
	// "Inlining" of get and Expired
	item, found := c.items[k]
	if !found {
		c.mu.RUnlock()
		return nil, time.Time{}, false
	}

	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			c.mu.RUnlock()
			return nil, time.Time{}, false
		}

		// Return the item and the expiration time
		c.mu.RUnlock()
		return item.Object, time.Unix(0, item.Expiration), true
	}

	// If expiration <= 0 (i.e. no expiration time set) then return the item
	// and a zeroed time.Time
	c.mu.RUnlock()
	return item.Object, time.Time{}, true
}

// Delete an item from the cache. Does nothing if the key is not in the cache.
//thread safe
func (c *cache) Delete(k string) {
	c.mu.Lock()
	v, evicted := c.delete(k)
	c.mu.Unlock()
	if evicted {
		c.onEvicted(k, v)
	}
}

//no thread safe
func (c *cache) delete(k string) (interface{}, bool) {
	if c.onEvicted != nil {
		if v, found := c.items[k]; found {
			delete(c.items, k)
			return v.Object, true
		}
	}
	delete(c.items, k)
	return nil, false
}

type keyAndValue struct {
	key   string
	value interface{}
}

// DeleteExpired Delete all expired items from the cache.
func (c *cache) DeleteExpired() {
	var evictedItems []keyAndValue
	now := time.Now().UnixNano()
	c.mu.Lock()
	for k, v := range c.items {
		// "Inlining" of expired
		if v.Expiration > 0 && now > v.Expiration {
			ov, evicted := c.delete(k)
			if evicted {
				evictedItems = append(evictedItems, keyAndValue{k, ov})
			}
		}
	}
	c.mu.Unlock()
	for _, v := range evictedItems {
		c.onEvicted(v.key, v.value)
	}
}

// OnEvicted Sets an (optional) function that is called with the key and value when an
// item is evicted from the cache. (Including when it is deleted manually, but
// not when it is overwritten.) Set to nil to disable.
func (c *cache) OnEvicted(f func(string, interface{})) {
	c.mu.Lock()
	c.onEvicted = f
	c.mu.Unlock()
}

// Items Copies all unexpired items in the cache into a new map and returns it.
func (c *cache) Items() map[string]*Item {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m := make(map[string]*Item, len(c.items))
	now := time.Now().UnixNano()
	for k, v := range c.items {
		// "Inlining" of Expired
		if v.Expiration > 0 {
			if now > v.Expiration {
				continue
			}
		}
		m[k] = v
	}
	return m
}

// ItemCount Returns the number of items in the cache. This may include items that have
// expired, but have not yet been cleaned up.
func (c *cache) ItemCount() int {
	c.mu.RLock()
	n := len(c.items)
	c.mu.RUnlock()
	return n
}

// Flush Delete all items from the cache.
func (c *cache) Flush() {
	c.mu.Lock()
	c.items = map[string]*Item{}
	c.mu.Unlock()
}

type janitor struct {
	Interval time.Duration
	stop     chan bool
}

func (j *janitor) Run(c *cache) {
	ticker := time.NewTicker(j.Interval)
	for {
		select {
		case <-ticker.C:
			c.DeleteExpired()
		case <-j.stop:
			ticker.Stop()
			return
		}
	}
}

func stopJanitor(c *Cache) {
	c.janitor.stop <- true
}

func runJanitor(c *cache, ci time.Duration) {
	j := &janitor{
		Interval: ci,
		stop:     make(chan bool),
	}
	c.janitor = j
	go j.Run(c)
}

func newCache(de time.Duration, m map[string]*Item, size int) *cache {
	if de == 0 {
		de = -1
	}
	c := &cache{
		defaultExpiration: de,
		items:             m,
		cacheSize:         size,
	}
	return c
}

func newCacheWithJanitor(de time.Duration, ci time.Duration, m map[string]*Item, size int) *Cache {
	c := newCache(de, m, size)
	// This trick ensures that the janitor goroutine (which--granted it
	// was enabled--is running DeleteExpired on c forever) does not keep
	// the returned C object from being garbage collected. When it is
	// garbage collected, the finalizer stops the janitor goroutine, after
	// which c can be collected.
	C := &Cache{c}
	if ci > 0 {
		runJanitor(c, ci)
		runtime.SetFinalizer(C, stopJanitor)
	}
	return C
}

// New Return a new cache with a given default expiration duration and cleanup
// interval. If the expiration duration is less than one (or NoExpiration),
// the items in the cache never expire (by default), and must be deleted
// manually. If the cleanup interval is less than one, expired items are not
// deleted from the cache before calling c.DeleteExpired().
func New(defaultExpiration, cleanupInterval time.Duration, size int) *Cache {
	items := make(map[string]*Item)
	return newCacheWithJanitor(defaultExpiration, cleanupInterval, items, size)
}

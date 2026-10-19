package cache

import (
	"container/list"
	"sync"
	"time"
)

type entry struct {
	key     string
	value   []byte
	expires time.Time
}

type Cache struct {
	mu    sync.Mutex
	max   int
	ttl   time.Duration
	items map[string]*list.Element
	order *list.List
}

func New(maxEntries int, ttl time.Duration) *Cache {
	if maxEntries <= 0 {
		maxEntries = 1024
	}
	if ttl <= 0 {
		ttl = time.Minute
	}
	return &Cache{
		max:   maxEntries,
		ttl:   ttl,
		items: make(map[string]*list.Element, maxEntries),
		order: list.New(),
	}
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, ok := c.items[key]
	if !ok {
		return nil, false
	}

	item := element.Value.(*entry)
	if time.Now().After(item.expires) {
		c.remove(element)
		return nil, false
	}

	c.order.MoveToFront(element)
	return item.value, true
}

func (c *Cache) Set(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, ok := c.items[key]; ok {
		item := element.Value.(*entry)
		item.value = value
		item.expires = time.Now().Add(c.ttl)
		c.order.MoveToFront(element)
		return
	}

	item := &entry{
		key:     key,
		value:   value,
		expires: time.Now().Add(c.ttl),
	}
	element := c.order.PushFront(item)
	c.items[key] = element

	for c.order.Len() > c.max {
		oldest := c.order.Back()
		if oldest == nil {
			return
		}
		c.remove(oldest)
	}
}

func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}

func (c *Cache) remove(element *list.Element) {
	item := element.Value.(*entry)
	delete(c.items, item.key)
	c.order.Remove(element)
}

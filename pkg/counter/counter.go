package counter

import "sync"

type Counter struct {
	mu    sync.Mutex
	value int64
	seen  map[string]struct{}
}

func NewCounter() *Counter {
	return &Counter{
		seen: make(map[string]struct{}),
	}
}

type IPeerCounter interface {
	Apply(eventID string, delta int64) bool
	Get() int64
}

func (c *Counter) Apply(eventID string, delta int64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.seen[eventID]; ok {
		return false
	}

	c.value += delta
	c.seen[eventID] = struct{}{}
	return true
}

func (c *Counter) Get() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

package speed

import (
	"sync"
	"time"
)

type Cache struct {
	*cache
}

type cache struct {
	items     map[string]Item   //保存条目
	mu        sync.RWMutex      //读写锁
	onEvicted func(interface{}) //回调事件  超时或者删除的时候触发回调
	snowflake *Node             //雪花算法生成key
	timeWheel *TimeWheel        //时间轮  过期调用
}

type Item struct {
	Object     interface{}
	Expiration int64
}

func New() (*Cache, error) {
	ip := GetLoaclIp()
	node := Ipv4StringToInt(ip) % 256
	sf, err := NewNode(node)
	if err != nil {
		return nil, err
	}
	tw := NewTw(time.Second, 3600, nil)
	return &Cache{&cache{
		items:     map[string]Item{},
		onEvicted: nil,
		snowflake: sf,
		timeWheel: tw,
	}}, nil
}

func (c *cache) Set(k string, v interface{}, d time.Duration) {
	var endTime int64
	if d > 0 {
		endTime = time.Now().Add(d).Unix()
	}
	c.mu.Lock()
	c.items[k] = Item{
		Object:     v,
		Expiration: endTime,
	}
	c.mu.Unlock()
}

func (c *cache) Get(k string) (interface{}, bool) {
	c.mu.RLock()
	item, ok := c.items[k]
	if !ok {
		c.mu.RUnlock()
		return nil, false
	}
	if item.Expiration <= time.Now().Unix() {
		return nil, false
	}
	c.mu.RUnlock()
	return item.Object, true
}

func (c *cache) GetAndEx(k string) (interface{}, time.Time, bool) {
	c.mu.RLock()
	item, ok := c.items[k]
	if !ok {
		c.mu.RUnlock()
		return nil, time.Time{}, false
	}
	if item.Expiration <= time.Now().Unix() {
		return nil, time.Time{}, false
	}
	c.mu.RUnlock()
	return item.Object, time.Unix(0, item.Expiration), true
}

func (c *cache) Delete(k string) {
	c.mu.Lock()
	v, ok := c.delete(k)
	c.mu.Unlock()
	if ok {
		c.onEvicted(v)
	}
}

func (c *cache) delete(k string) (interface{}, bool) {
	if c.onEvicted != nil {
		if v, ok := c.items[k]; ok {
			delete(c.items, k)
			return v.Object, true
		}
	}
	delete(c.items, k)
	return nil, false
}

func (c *cache) OnEvicted(f func(interface{})) {
	c.mu.Lock()
	c.onEvicted = f
	c.mu.Unlock()
}

func (c *cache) Items() map[string]Item {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m := make(map[string]Item, len(c.items))
	now := time.Now().Unix()
	for k, v := range c.items {
		if now > v.Expiration {
			continue
		}
		m[k] = v
	}
	return m
}

func (c *cache) ItemCount() int {
	c.mu.RLock()
	n := len(c.items)
	c.mu.RUnlock()
	return n
}

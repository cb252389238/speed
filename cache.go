package speed

import (
	"context"
	"sync"
	"time"
)

type Cache struct {
	*cache
}

type cache struct {
	items          map[string]Item   //保存条目
	mu             sync.RWMutex      //读写锁
	deleteCallBack func(interface{}) //回调事件  超时或者删除的时候触发回调
	snowflake      *Node             //雪花算法生成key
	timeWheel      *TimeWheel        //时间轮  过期调用
	ctx            context.Context
	cancel         context.CancelFunc
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
	tw := NewTw(time.Second, 60, nil)
	tw.Start()
	ctx, cancelFunc := context.WithCancel(context.Background())
	c := &Cache{&cache{
		items:          map[string]Item{},
		deleteCallBack: nil,
		snowflake:      sf,
		timeWheel:      tw,
		ctx:            ctx,
		cancel:         cancelFunc,
	}}
	go c.run()
	return c, nil
}

func (c *cache) run() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case key := <-c.timeWheel.C: //超时队列
			if k, ok := key.(string); ok {
				if i, b := c.delete(k); b {
					c.deleteCallBack(i.Object)
				}
			}
		}
	}
}

func (c *cache) Stop() {
	c.cancel()
}

func (c *cache) Set(k string, v interface{}, d time.Duration) {
	var endTime int64
	if d > 0 {
		endTime = time.Now().Add(d).Unix()
	}
	c.mu.Lock()
	if val, ok := c.items[k]; ok {
		if val.Expiration > 0 {
			c.timeWheel.RemoveTimer(k)
		}
	}
	item := Item{
		Object:     v,
		Expiration: endTime,
	}
	c.items[k] = item
	c.mu.Unlock()
	c.timeWheel.AddTimer(d, k, item)
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
	return item.Object, time.Unix(item.Expiration, 0), true
}

func (c *cache) Delete(k string) {
	c.mu.Lock()
	v, ok := c.delete(k)
	c.mu.Unlock()
	c.timeWheel.RemoveTimer(k)
	if ok {
		c.deleteCallBack(v.Object)
	}
}

func (c *cache) delete(k string) (Item, bool) {
	if c.deleteCallBack != nil {
		if v, ok := c.items[k]; ok {
			delete(c.items, k)
			return v, true
		}
	}
	delete(c.items, k)
	return Item{}, false
}

func (c *cache) BindDeleteCallBackFunc(f func(interface{})) {
	c.mu.Lock()
	c.deleteCallBack = f
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

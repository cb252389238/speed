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
	kvItems        map[string]KVItem //k-v结构
	mu             sync.RWMutex      //读写锁
	deleteCallBack func(interface{}) //回调事件  超时或者删除的时候触发回调
	snowflake      *Node             //雪花算法生成key
	timeWheel      *TimeWheel        //时间轮  过期调用
	ctx            context.Context
	cancel         context.CancelFunc
}

type KVItem struct {
	Object     interface{} //存储体
	Expiration int64       //过期时间
	CallBack   bool        //是否回调
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
		kvItems:        map[string]KVItem{},
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
					c.deleteCallBack(i)
				}
			}
		}
	}
}

func (c *cache) BindDeleteCallBackFunc(f func(interface{})) {
	c.mu.Lock()
	c.deleteCallBack = f
	c.mu.Unlock()
}

func (c *cache) Stop() {
	c.cancel()
}

func (c *cache) Set(k string, v interface{}, d time.Duration, callBack bool) {
	var endTime int64
	if d > 0 {
		endTime = time.Now().Add(d).Unix()
	}
	c.mu.RLock()
	if val, ok := c.kvItems[k]; ok {
		if val.Expiration > 0 {
			c.mu.RUnlock()
			c.timeWheel.RemoveTimer(k)
		} else {
			c.mu.RUnlock()
		}
	} else {
		c.mu.RUnlock()
	}
	item := KVItem{
		Object:     v,
		Expiration: endTime,
		CallBack:   callBack,
	}
	c.mu.Lock()
	c.kvItems[k] = item
	c.mu.Unlock()
	c.timeWheel.AddTimer(d, k, item)
}

func (c *cache) SetNx(k string, v interface{}, d time.Duration, callBack bool) bool {
	var endTime int64
	if d > 0 {
		endTime = time.Now().Add(d).Unix()
	}
	c.mu.RLock()
	_, ok := c.kvItems[k]
	c.mu.RUnlock()
	if ok {
		return false
	}
	item := KVItem{
		Object:     v,
		Expiration: endTime,
		CallBack:   callBack,
	}
	c.mu.Lock()
	c.kvItems[k] = item
	c.mu.Unlock()
	c.timeWheel.AddTimer(d, k, item)
	return true
}

func (c *cache) Get(k string) (interface{}, bool) {
	c.mu.RLock()
	item, ok := c.kvItems[k]
	if !ok {
		c.mu.RUnlock()
		return nil, false
	}
	c.mu.RUnlock()
	if item.Expiration <= time.Now().Unix() {
		return nil, false
	}
	return item.Object, true
}

//获取k-v 过期时间
func (c *cache) GetEx(k string) (interface{}, time.Time, bool) {
	c.mu.RLock()
	item, ok := c.kvItems[k]
	if !ok {
		c.mu.RUnlock()
		return nil, time.Time{}, false
	}
	c.mu.RUnlock()
	if item.Expiration <= time.Now().Unix() {
		return nil, time.Time{}, false
	}
	return item.Object, time.Unix(item.Expiration, 0), true
}

//k-v删除
func (c *cache) Del(k string) {
	c.mu.Lock()
	v, ok := c.delete(k)
	c.mu.Unlock()
	if v.Expiration > 0 {
		c.timeWheel.RemoveTimer(k)
	}
	if ok && v.CallBack {
		c.deleteCallBack(v.Object)
	}
}

func (c *cache) delete(k string) (KVItem, bool) {
	if c.deleteCallBack != nil {
		if v, ok := c.kvItems[k]; ok {
			delete(c.kvItems, k)
			return v, true
		}
	}
	delete(c.kvItems, k)
	return KVItem{}, false
}

//获取k-v所有值
func (c *cache) Items() map[string]KVItem {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m := make(map[string]KVItem, len(c.kvItems))
	now := time.Now().Unix()
	for k, v := range c.kvItems {
		if now > v.Expiration {
			continue
		}
		m[k] = v
	}
	return m
}

//获取k-v数量
func (c *cache) ItemCount() int {
	c.mu.RLock()
	n := len(c.kvItems)
	c.mu.RUnlock()
	return n
}

//判断k-v值是否存在
func (c *cache) Exists(k string) bool {
	c.mu.RLock()
	_, ok := c.kvItems[k]
	c.mu.RUnlock()
	return ok
}

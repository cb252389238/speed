package speed

import (
	"fmt"
	"testing"
	"time"
)

func TestSpeedCache(t *testing.T) {
	c, err := New()
	if err != nil {
		fmt.Println(err)
		return
	}
	c.Set("key", 1, time.Second*5) //设置缓存 0为永不过期
	fmt.Println(c.Get("key"))      //获取结果 返回1 true
	fmt.Println(c.GetAndEx("key")) //获取结果以及过期时间 1 2022-07-30 14:35:19 +0800 CST true
	c.Delete("key")                //删除缓存
	fmt.Println(c.Get("key"))      //<nil> false

	c.Set("key", 1, time.Second*5) //设置缓存 5秒后过期
	time.Sleep(time.Second * 3)    //生命周期还剩两秒
	c.Set("key", 1, time.Second*5) //再次设置相同的key，更新生命周期
	time.Sleep(time.Second * 3)    //生命周期还剩五秒
	fmt.Println(c.Get("key"))      //1 true

	//绑定回调函数，当主动删除缓存或者缓存过期触发  v就是设置的缓存值
	c.BindDeleteCallBackFunc(func(v interface{}) {
		fmt.Println(v)
		fmt.Println("触发回调函数")
	})
	c.Set("test01", 100, time.Second*3)
	c.Delete("test01")                  //打印 100  触发回调函数
	c.Set("test02", 200, time.Second*2) //两秒后过期触发回调函数
	time.Sleep(time.Second * 100)       //阻塞
}

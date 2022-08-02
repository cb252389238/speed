## speed
#### 快速缓存库，支持超时设置，超时/删除回调


## 安装

```
go get github.com/cb252389238/speed
```




```go
c, err := New()
if err != nil {
fmt.Println(err)
    return
}
c.Set("key", 1, time.Second*5, false) //设置缓存 0为永不过期 [key,val,expiration,是否触发过期回调]
fmt.Println(c.Get("key"))             //获取结果 返回1 true
fmt.Println(c.GetEx("key"))           //获取结果以及过期时间 1 2022-07-30 14:35:19 +0800 CST true
c.Del("key")                          //删除缓存
fmt.Println(c.Get("key"))             //<nil> false

c.Set("key", 1, time.Second*5, false) //设置缓存 5秒后过期
time.Sleep(time.Second * 3)           //生命周期还剩两秒
c.Set("key", 1, time.Second*5, false) //再次设置相同的key，更新生命周期
time.Sleep(time.Second * 3)           //生命周期还剩五秒
fmt.Println(c.Get("key"))             //1 true

//绑定回调函数，当主动删除缓存或者缓存过期触发  v就是设置的缓存值
c.BindDeleteCallBackFunc(func(v interface{}) {
fmt.Println(v)
    fmt.Println("触发回调函数")
})
c.Set("test01", 100, time.Second*3, false)
c.Del("test01")                           //打印 100  callBack false 不触发回调函数
c.Set("test02", 200, time.Second*2, true) //两秒后过期触发回调函数
time.Sleep(time.Second * 100)             //阻塞
```

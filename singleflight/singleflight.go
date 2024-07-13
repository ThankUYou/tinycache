package singleflight

import "sync"

// call 代表一次函数调用，表示正在执行中，或已经结束的请求
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

type Group struct { // singleflight的主数据结构，管理不同key的请求call
	mu sync.Mutex
	m  map[string]*call
}

// Do 针对一样的key， 无论DO执行多少次，fn只会执行一次
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	// 懒初始化
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	// 如果当前map有key了，说明这个key正在执行，不需要再继续请求了等待结果就行
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()         //如果请求正在进行，则等待
		return c.val, c.err //请求结束，返回结果
	}

	//否则创建一个请求，并加入mp
	c := new(call)
	c.wg.Add(1) //发起请求前加锁
	g.m[key] = c
	g.mu.Unlock()
	//等待请求执行完，则Done通知所有wait的
	c.val, c.err = fn() //无论do被调用多少次，函数fn都只会调用一次
	c.wg.Done()         //请求结束，解锁

	g.mu.Lock()
	delete(g.m, key) //更新g.m
	g.mu.Unlock()

	return c.val, c.err
}

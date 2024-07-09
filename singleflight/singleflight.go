package singleflight

import "sync"

// call 代表一次函数调用
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

type Group struct { // singleflight的主数据结构，管理不同key的请求call
	mu sync.Mutex
	m  map[string]*call
}

func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}

	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()         //如果请求正在进行，则等待
		return c.val, c.err //请求结束，返回结果
	}

	c := new(call)
	c.wg.Add(1) //发起请求前加锁
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn() //无论do被调用多少次，函数fn都只会调用一次
	c.wg.Done()         //请求结束，解锁

	g.mu.Lock()
	delete(g.m, key) //更新g.m
	g.mu.Unlock()

	return c.val, c.err
}

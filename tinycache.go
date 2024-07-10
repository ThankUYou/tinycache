package tinycache

import (
	"log"
	"sync"
	"tinycache/singleflight"
	"tinycache/strategy"
	pb "tinycache/tinycachepb"
)

// Getter 从数据源获取数据，并且将获取的数据添加到缓存中
type Getter interface {
	Get(key string) ([]byte, error)
}

// A GetterFunc implements Getter with a function.
type GetterFunc func(key string) ([]byte, error)

// Get implements Getter interface function
// 函数类型实现某一个接口，称之为接口型函数，方便使用者在调用时既能够传入函数作为参数，也能够传入实现了该接口的结构体作为参数。
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// Group 负责与用户交互，控制缓存存储和获取的流程
// -----------------------------------------------------------------
// 主体结构Group负责与外部交互，控制缓存储存与获取的主流程
// 一个缓存的命名空间，每个Group都有一个唯一的name，比如三个Group
// 缓存学生成绩的叫scores， 缓存学生信息的叫info， 缓存课程的叫coures
// getter 即缓存未命中时获取元数据的回调函数
// mainCache 即一开始实现的并发缓存
type Group struct {
	name      string              // 缓存的命名空间
	getter    Getter              // 缓存未命中时获取源数据的回调
	mainCache mainCache           // 并发缓存
	peers     PeerPicker          // 节点选择器
	loader    *singleflight.Group // 防止缓存击穿
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// RegisterPeers 添加新的peer
func (g *Group) RegisterPeers(peer PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peer
}

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}

	mu.Lock()
	defer mu.Unlock()
	group := &Group{
		name:      name,
		getter:    getter,
		mainCache: mainCache{cache: NewByOption(DefaultOption.cacheType, cacheBytes)},
		loader:    &singleflight.Group{},
	}
	groups[name] = group
	return group
}

// GetGroup 返回命名空间为name的Group
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// Get 实现了Getter接口，从缓存中查找缓存数据，如果不存在则调用 load 方法获取
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, strategy.ErrKeyNotFound
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[TinyCache] hit")
		return v, nil
	}
	return g.load(key)
}

func (g *Group) load(key string) (value ByteView, err error) {
	// 无论并发调用者的数量如何，每个键只被获取一次(本地或远程)
	viewi, err := g.loader.Do(key, func() (any, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[TinyCache] Failed to get from peer", err)
			}
		}
		return g.getLocally(key)
	})
	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}

// getLocally 调用用户回调函数 g.getter.Get() 获取源数据，并且将源数据添加到缓存 mainCache 中（通过 populateCache 方法）
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key) // 实际上就是执行GetterFunc函数
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

//每执行一次main函数就是起一个节点服务                 本地用户交互前端连接绑定了一个gee节点，其余节点皆为单纯gee缓存数据节点
// Overall flow char										     requsets			先看local有没有		        local
// gee := createGroup() --------> /api Service : 9999 ------------------------------> gee.Get(key) ------> g.mainCache.Get(key)
// 						|											^					|
// 						|											|					|remote 查看远程节点有没有的逻辑
// 						v											|					v
// 				cache Service : 800x								|			g.peers.PickPeer(key)通过一致性哈希找到这个key应该落在的真正节点地址
// 						|create hash ring & init peerGetter			|					|
// 						|registry peers write in g.peer				|					|p.grpcGetters[p.hashRing(key)]
// 						v											|					|
//			grpcPool.Set(otherAddrs...)								|					v
// 		g.peers = gee.RegisterPeers(grpcPool)						|			g.getFromPeer(peerGetter, key)//通过grpc向这个真正节点发送请求
// 						|											|					|
// 						|											|					|
// 						v											|					v
// 		http.ListenAndServe("localhost:800x", httpPool)<------------+--------------peerGetter.Get(key) //这个节点查看本地有没有，没有就在这个节点本地加载
// 						|											|
// 						|requsets									|
// 						v											|
// 					p.ServeHttp(w, r)								|
// 						|											|
// 						|url.parse()								|
// 						|--------------------------------------------

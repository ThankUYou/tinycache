package tinycache

import pb "tinycache/tinycachepb"

// PeerPicker 可以通过PeerPicker接口找到对应key的peer(对等节点)
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter 这个接口实现了在某个peer根据key找到对应的value
type PeerGetter interface {
	Get(in *pb.Request, out *pb.Response) error // 用于对应的group查找缓存值
}

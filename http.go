package tinycache

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"tinycache/hash"
	pb "tinycache/tinycachepb"
)

const defaultPath = "/_geecache/"
const defaultReplicas = 50

type HTTPPOOL struct {
	self       string                 // 记录自己的地址, e.g. "http://localhost:8001"
	basePath   string                 // 节点间通讯地址的前缀
	mu         sync.Mutex             // guards peers and httpGetters
	peers      *hash.Map              // 一致性哈希算法的实例
	httpGetter map[string]*httpGetter // 每一个远程节点地址对应一个 httpGetter
}

func NewHTTPPool(s string) *HTTPPOOL {
	return &HTTPPOOL{
		self:     s,
		basePath: defaultPath,
	}
}

func (p *HTTPPOOL) Log(format string, v ...interface{}) {
	log.Printf(("[SERVER %s]%s"), p.self, fmt.Sprintf(format, v...))
}

func (p *HTTPPOOL) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> required
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write the value to the response body as a proto message.
	body, err := proto.Marshal(&pb.Response{Value: view.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)
}

//-------------------------------------------------------------------------
// 客户端类的实现

type httpGetter struct {
	baseURL string // 即将访问的远程节点的地址，http://example.com/_geecache/group名
}

func (h *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	)
	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}

	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}

var _ PeerGetter = (*httpGetter)(nil)

func (h *HTTPPOOL) Set(peers ...string) { // 实例化一个哈希算法，传入真实节点地址， 为每一个节点创造了一个方法httpGetter用于客户端从服务端发来的报文中获得缓存值
	h.mu.Lock()
	defer h.mu.Unlock()
	h.peers = hash.NewConsistentHash(defaultReplicas, nil)
	h.peers.Add(peers...)
	h.httpGetter = make(map[string]*httpGetter)
	for _, peer := range peers {
		h.httpGetter[peer] = &httpGetter{baseURL: peer + h.basePath} // http://节点地址peer/_geecache/
	}
}

func (h *HTTPPOOL) PickPeer(key string) (PeerGetter, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if peer := h.peers.Get(key); peer != "" && peer != h.self { // 根据key和一致性哈希算法，找到映射的真实节点地址。
		h.Log("pick peer %s", peer)
		return h.httpGetter[peer], true
	}

	return nil, false
}

var _ PeerPicker = (*HTTPPOOL)(nil)

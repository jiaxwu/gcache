package gcache

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/jiaxwu/gcache/consistenthash"
	pb "github.com/jiaxwu/gcache/gcachepb"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_gcache/"
	// 虚拟节点倍数
	defaultReplicas = 50
)

// HTTPPool 实现了伙伴节点
type HTTPPool struct {
	// 监听地址，比如https://example.net:8080
	self string
	// 基础路径，避免冲突，比如"/_gcache/"
	basePath string
	// 保证设置同伴节点安全
	mu          sync.RWMutex
	peers       *consistenthash.Map
	httpGetters map[string]*httpGetter
}

// NewHTTPPool 创建一个HTTPPool
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s\n", p.self, fmt.Sprintf(format, v...))
}

// Set 更新同伴节点
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// PickPeer 根据键获取对应的远程节点客户端
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	peer := p.peers.Get(key)
	if peer == "" || peer == p.self {
		return nil, false
	}
	p.Log("Pick peer %s", peer)
	return p.httpGetters[peer], true
}

// ServeHTTP 处理所有http请求
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// /<basePath>/<groupName>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName, key := parts[0], parts[1]
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	// 删除键
	if r.Method == http.MethodDelete {
		group.removeLocally(key)
		return
	}

	// 获取键
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var expireNano int64
	if !view.Expire().IsZero() {
		expireNano = view.Expire().UnixNano()
	}
	body, err := proto.Marshal(&pb.Response{
		Value:  view.ByteSlice(),
		Expire: expireNano,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)
}

// 远程节点请求客户端，每个远程节点一个
type httpGetter struct {
	baseURL string
}

func (h *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	res, err := h.makeRequest(http.MethodGet, in)
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
	if err := proto.Unmarshal(bytes, out); err != nil {
		return err
	}
	return nil
}

func (h *httpGetter) Remove(in *pb.Request) error {
	res, err := h.makeRequest(http.MethodDelete, in)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}
	return nil
}

func (h *httpGetter) makeRequest(method string, in *pb.Request) (*http.Response, error) {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	)
	req, err := http.NewRequest(method, u, nil)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}

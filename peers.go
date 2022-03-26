package gcache

// PeerGetter 远程客户端，根据group和key获取缓存
type PeerGetter interface {
	Get(group, key string) ([]byte, error)
}

// PeerPicker 用于获取远程节点的请求客户端
type PeerPicker interface {
	PickPeer(key string) (PeerGetter, bool)
}

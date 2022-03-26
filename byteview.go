package gcache

import "time"

// ByteView 一个不可变的字节数组视图
type ByteView struct {
	b      []byte
	expire time.Time
}

func NewByteView(b []byte, expire time.Time) ByteView {
	return ByteView{
		b:      b,
		expire: expire,
	}
}

func (v ByteView) Expire() time.Time {
	return v.expire
}

func (v ByteView) Len() int {
	return len(v.b)
}

func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

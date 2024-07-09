package tinycache

// 抽象出一个只读数据结构用来表示缓存值，使用[]byte可以表示任意数据结构类型的存储
// 1. ByteView 只有一个数据成员，b []byte，b 将会存储真实的缓存值。选择 byte 类型是为了能够支持任意的数据类型的存储，例如字符串、图片等。
// 2. 实现 Len() int 方法，我们在 Cache 的实现中，要求被缓存对象必须实现 Value 接口，即 Len() int 方法，返回其所占的内存大小。
// 3. b 是只读的，使用 ByteSlice() 方法返回一个拷贝，防止缓存值被外部程序修改。

type ByteView struct {
	b []byte
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

// 深拷贝一份返回副本
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

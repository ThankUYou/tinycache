package strategy

import "errors"

var (
	ErrInvalidMaxBytes = errors.New("MaxBytes set error")
	ErrInvalidCache    = errors.New("Cache is nil")
	ErrKeyNotFound     = errors.New("Key Not Found")
)

// Value is a interface for value type
// 为了通用性，允许值Value接口是任意类型，该接口只包含一个方法，用于返回值所占内存大小
type Value interface {
	Len() int
}

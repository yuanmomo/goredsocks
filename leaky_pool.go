package main

type LeakyPool struct {
	bufSize  int // size of each buffer
	freeList chan []byte
}

const leakyBufSize = 64 * 1024
const maxNBuf = 128

var leakyBuf = NewLeakyPool(maxNBuf, leakyBufSize)

func NewLeakyPool(n, bufSize int) *LeakyPool {
	return &LeakyPool{
		bufSize:  bufSize,
		freeList: make(chan []byte, n),
	}
}

func (lb *LeakyPool) Get() (b []byte) {
	select {
	case b = <-lb.freeList:
	default:
		b = make([]byte, lb.bufSize)
	}
	return
}
func (lb *LeakyPool) Put(b []byte) {
	if len(b) != lb.bufSize {
		panic("invalid buffer size that's put into leaky buffer")
	}
	select {
	case lb.freeList <- b:
	default:
	}
	return
}

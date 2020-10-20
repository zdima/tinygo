package machine

import (
	"runtime/volatile"
)

const bufferSize = 128

// RingBuffer is ring buffer implementation inspired by post at
// https://www.embeddedrelated.com/showthread/comp.arch.embedded/77084-1.php
type RingBuffer struct {
	rxbuffer [bufferSize]volatile.Register8
	head     volatile.Register8
	tail     volatile.Register8
}

// NewRingBuffer returns a new ring buffer.
func NewRingBuffer() *RingBuffer {
	return &RingBuffer{}
}

// Used returns how many bytes in buffer have been used.
func (rb *RingBuffer) Used() uint8 {
	return uint8(rb.head.Get() - rb.tail.Get())
}

// Put stores a byte in the buffer. If the buffer is already
// full, the method will return false.
func (rb *RingBuffer) Put(val byte) bool {
	if rb.Used() != bufferSize {
		rb.head.Set(rb.head.Get() + 1)
		rb.rxbuffer[rb.head.Get()%bufferSize].Set(val)
		return true
	}
	return false
}

// Get returns a byte from the buffer. If the buffer is empty,
// the method will return a false as the second value.
func (rb *RingBuffer) Get() (byte, bool) {
	if rb.Used() != 0 {
		rb.tail.Set(rb.tail.Get() + 1)
		return rb.rxbuffer[rb.tail.Get()%bufferSize].Get(), true
	}
	return 0, false
}

// Clear resets the head and tail pointer to zero.
func (rb *RingBuffer) Clear() {
	rb.head.Set(0)
	rb.tail.Set(0)
}

const bufferSize16 = 512 + 256

// RingBuffer16 is ring buffer implementation inspired by post at
// https://www.embeddedrelated.com/showthread/comp.arch.embedded/77084-1.php
type RingBuffer16 struct {
	buffer [bufferSize16]volatile.Register8
	head   volatile.Register16
	tail   volatile.Register16
}

// NewRingBuffer16 returns a new ring buffer.
func NewRingBuffer16() *RingBuffer16 {
	rb := &RingBuffer16{}
	rb.Clear()
	return rb
}

// Used returns how many bytes in buffer have been used.
func (rb *RingBuffer16) Used() uint16 {
	return uint16(rb.head.Get() - rb.tail.Get())
}

// Put stores a byte in the buffer. If the buffer is already
// full, the method will return false.
func (rb *RingBuffer16) Put(val byte) bool {
	if rb.Used() != bufferSize16 {
		rb.head.Set(rb.head.Get() + 1)
		rb.buffer[rb.head.Get()%bufferSize16].Set(val)
		return true
	}
	return false
}

// Get returns a byte from the buffer. If the buffer is empty,
// the method will return a false as the second value.
func (rb *RingBuffer16) Get() (byte, bool) {
	if rb.Used() != 0 {
		rb.tail.Set(rb.tail.Get() + 1)
		return rb.buffer[rb.tail.Get()%bufferSize16].Get(), true
	}
	return 0, false
}

// Clear resets the head and tail pointer to zero.
func (rb *RingBuffer16) Clear() {
	rb.head.Set(bufferSize16 - 1)
	rb.tail.Set(bufferSize16 - 1)
}

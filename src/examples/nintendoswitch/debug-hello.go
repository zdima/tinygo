// This program is a minimal Nintendo Switch program that just outputs messages to
// emulator console
package main

import (
	"runtime"
	"time"
)

func print(data string) {
	t := []byte(data)
	l := uint64(len(t))
	runtime.NxOutputString(&t[0], l)
}

func main() {
	print("Hello world!")

	for {
		print("Cycle!!")
		time.Sleep(time.Second)
	}
}

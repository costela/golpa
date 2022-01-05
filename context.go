package golpa

import (
	"C"
	"context"
	"sync"
	"unsafe"
)

/*
 This code is used to work around the garbage collector and keep track of contexts passed to callback code.
 Inspired by github.com/mattn/go-pointer
*/

var (
	contextsMu sync.Mutex
	contexts   = make(map[unsafe.Pointer]context.Context)
)

func saveContext(ctx context.Context) unsafe.Pointer {
	contextsMu.Lock()
	defer contextsMu.Unlock()

	var p unsafe.Pointer = C.malloc(C.size_t(1))
	if p == nil {
		panic("could not allocate memory for CGO pointer tracking")
	}

	contexts[p] = ctx

	return p
}

func loadContext(ptr unsafe.Pointer) context.Context {
	contextsMu.Lock()
	defer contextsMu.Unlock()

	return contexts[ptr]
}

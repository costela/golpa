package golpa

import (
	"C"
	"sync"
	"unsafe"
)

/*
 This code is used to work around the garbage collector and keep track of objects passed to callback code.
 Inspired by github.com/mattn/go-pointer
*/

var (
	refsMu sync.Mutex
	refs   = make(map[unsafe.Pointer]interface{})
)

func saveRef(ref interface{}) unsafe.Pointer {
	refsMu.Lock()
	defer refsMu.Unlock()

	var p unsafe.Pointer = C.malloc(C.size_t(1))
	if p == nil {
		panic("could not allocate memory for CGO pointer tracking")
	}

	refs[p] = ref

	return p
}

func loadRef(ptr unsafe.Pointer) interface{} {
	refsMu.Lock()
	defer refsMu.Unlock()

	return refs[ptr]
}

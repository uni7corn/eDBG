package utils

// #include <load_so.h>
// #cgo LDFLAGS: -ldl
type UnwindOption struct {
	Abi       uint64
	StackSize uint64
	DynSize   uint64
	RegMask   uint64
	ShowPC    bool
}

type UnwindBuf struct {
	Abi       uint64
	Regs      []uint64
	StackSize uint64
	Data      []byte
	DynSize   uint64
}

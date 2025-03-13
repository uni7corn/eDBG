package utils

import (
	"fmt"
	"golang.org/x/sys/unix"
	// "unsafe"
)

func ReadProcessMemory(pid uint32, remoteAddr uintptr, buffer []byte) (int, error) {
	localIov := []unix.Iovec{
		{Base: &buffer[0], Len: uint64(len(buffer))},
	}
	remoteIov := []unix.RemoteIovec{
		{Base: remoteAddr, Len: int(len(buffer))},
	}

	n, err := unix.ProcessVMReadv(
		int(pid),
		localIov,
		remoteIov,
		0, // flags
	)
	if err != nil {
		return 0, fmt.Errorf("ReadMemory failed: %v", err)
	}
	return n, nil
}

// // 示例用法
// func main() {
// 	pid := 1234       // 目标进程PID
// 	addr := 0x7f0000  // 目标内存地址（需对齐）
// 	buf := make([]byte, 4)

// 	n, err := ReadProcessMemory(pid, uintptr(addr), buf)
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 	} else {
// 		fmt.Printf("Read %d bytes: %x\n", n, buf)
// 	}
// }
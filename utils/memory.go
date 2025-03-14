package utils

import (
	"fmt"
	"golang.org/x/sys/unix"
	"strconv"
	"encoding/binary"
	"strings"
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

func TryRead(pid uint32, remoteAddr uintptr) (bool, string) {
	buf := make([]byte, 8)
	n, err := ReadProcessMemory(pid, remoteAddr, buf)
	if err != nil {
		return false, ""
	}
	count := 0
	for _, b := range buf {
		if strconv.IsPrint(rune(b)) {
			count++
		} else {
			break
		}
	}
	outbuf := &strings.Builder{}
	if count > 5 {
		// 认为是可见字符串
		outbuf.WriteByte('"')
		stringbuf := make([]byte, 30)
		_, err := ReadProcessMemory(pid, remoteAddr, stringbuf)
		if err != nil {
			fmt.Fprintf(outbuf, "0x")
			for i := 0; i < n; i++ {
				fmt.Fprintf(outbuf, "%02x", buf[i])
			}
		}
		count = 0
		for _, b := range stringbuf {
			if strconv.IsPrint(rune(b)) {
				outbuf.WriteByte(b)
				count++
			} else {
				break
			}
		}
		if count > 29 {
			fmt.Fprintf(outbuf, "...")
		}
		outbuf.WriteByte('"')

	} else {
		fmt.Fprintf(outbuf, "0x%X", binary.LittleEndian.Uint64(buf))
	}
	return true, outbuf.String()
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
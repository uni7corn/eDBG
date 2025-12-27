package utils

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
	// "unsafe"
)

func ReadProcessMemory(pid uint32, remoteAddr uintptr, buffer []byte) (int, error) {
	remoteAddr &= 0x00FFFFFFFFFFFFFF;
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
		return 0, fmt.Errorf("ReadMemory %x failed: %v", remoteAddr, err)
	}
	return n, nil
}

func ReadProcessMemoryRobust(pid uint32, startAddr uintptr, totalSize int) ([]byte, error) {
	fullBuffer := make([]byte, totalSize)
	var totalReadBytes int
	const chunkSize = 4096

	for totalReadBytes < totalSize {
		currentAddr := startAddr + uintptr(totalReadBytes)
		bytesToRead := chunkSize
		if totalReadBytes+bytesToRead > totalSize {
			bytesToRead = totalSize - totalReadBytes
		}
		chunkBuffer := make([]byte, bytesToRead)
		n, err := ReadProcessMemory(pid, currentAddr, chunkBuffer)

		if err != nil {
		} else if n > 0 {
			copy(fullBuffer[totalReadBytes:], chunkBuffer[:n])
		}
		totalReadBytes += bytesToRead
	}

	return fullBuffer, nil
}

func WriteProcessMemory(pid uint32, remoteAddr uintptr, data []byte) (int, error) {
	remoteAddr &= 0x00FFFFFFFFFFFFFF;
	localIov := []unix.Iovec{
		{Base: &data[0], Len: uint64(len(data))},
	}
	remoteIov := []unix.RemoteIovec{
		{Base: remoteAddr, Len: int(len(data))},
	}
	n, err := unix.ProcessVMWritev(
		int(pid),
		localIov,
		remoteIov,
		0, // flags
	)

	if err != nil {
		return 0, fmt.Errorf("WriteProcessMemory failed: %v", err)
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

func ReadMapsByPid(pid uint32) (string, error) {
	filename := fmt.Sprintf("/proc/%d/maps", pid)
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

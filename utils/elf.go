package utils

import (
	"debug/elf"
	"fmt"
)

func ConvertVirtualOffsetToFileOffset(libPath string, vaddr uint64) (uint64, error) {
	elfFile, err := elf.Open(libPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open ELF file %s: %w", libPath, err)
	}
	defer elfFile.Close()
	for _, prog := range elfFile.Progs {
		if prog.Type != elf.PT_LOAD {
			continue
		}
		if vaddr >= prog.Vaddr && vaddr < prog.Vaddr+prog.Memsz {
			offsetInSegment := vaddr - prog.Vaddr
			fileOffset := prog.Off + offsetInSegment
			return fileOffset, nil
		}
	}
	return 0, fmt.Errorf("virtual address 0x%x not found in any loadable segment of %s", vaddr, libPath)
}

func ConvertFileOffsetToVirtualOffset(libPath string, fileOffset uint64) (uint64, error) {
	elfFile, err := elf.Open(libPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open ELF file %s: %w", libPath, err)
	}
	defer elfFile.Close()
	for _, prog := range elfFile.Progs {
		if prog.Type != elf.PT_LOAD {
			continue
		}
		if fileOffset >= prog.Off && fileOffset < prog.Off+prog.Filesz {
			offsetInSegment := fileOffset - prog.Off
			vaddr := prog.Vaddr + offsetInSegment
			return vaddr, nil
		}
	}
	return 0, fmt.Errorf("file offset 0x%x not found in any loadable segment of %s", fileOffset, libPath)
}

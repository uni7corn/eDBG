package controller

import (
	"debug/elf"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// var DoneSymbol []*CachedLibInfo

func (this *Process) ExportSymbols(libbase uint64, libpath string, liboffset uint64) error {
	// 判断文件是否为APK
	ext := strings.ToLower(filepath.Ext(libpath))
	isAPK := ext == ".apk"

	// 创建合适的ReaderAt
	var readerAt io.ReaderAt
	var closer io.Closer

	if isAPK {
		file, err := os.Open(libpath)
		if err != nil {
			return fmt.Errorf("Cannot read APK: %v", err)
		}
		closer = file
		readerAt = io.NewSectionReader(file, int64(liboffset), 1<<63-1) // 读取到文件末尾
	} else {
		file, err := os.Open(libpath)
		if err != nil {
			return fmt.Errorf("Cannot read library: %v", err)
		}
		closer = file
		readerAt = file
	}
	defer closer.Close()

	// 解析ELF文件
	f, err := elf.NewFile(readerAt)
	if err != nil {
		return fmt.Errorf("ELF Parse failed: %v", err)
	}

	// 获取动态符号表
	symbols, err := f.DynamicSymbols()
	if err == nil {
		// 遍历所有符号并计算偏移
		for _, sym := range symbols {
			offset, err := findSymbolOffset(f, sym)
			if err != nil {
				// fmt.Printf("Cannot get symbol %s offset: %v\n", sym.Name, err)
				continue
			}
			this.Symbols[libbase+offset] = sym.Name
			// fmt.Printf("%s\t0x%x\n", sym.Name, offset)
		}
	}

	plt := f.Section(".plt")
	gotplt := f.Section(".got.plt")
	dynsym := f.Section(".dynsym")
	if plt == nil || gotplt == nil || dynsym == nil {
		return nil
	}
	symData, err := dynsym.Data()
	// fmt.Println("syms: ", symbol)
	strtab := f.Section(".dynstr")
	if strtab == nil {
		return nil
	}
	strdata, err := strtab.Data()
	if err != nil {
		return nil
	}
	// fmt.Println("strtabs: ", strdata)
	// 解析所有.rela节的重定位项
	for _, section := range f.Sections {
		if section.Type != elf.SHT_RELA {
			continue
		}

		// 读取重定位数据
		data, err := section.Data()
		if err != nil {
			continue
		}

		// 解析每个RELA条目
		entSize := section.Entsize
		if entSize < 24 {
			continue
		}

		n := len(data) / int(entSize)
		for i := 0; i < n; i++ {
			// 解析Rela结构
			entry := data[i*int(entSize):]
			offset := binary.LittleEndian.Uint64(entry[0:8])
			info := binary.LittleEndian.Uint64(entry[8:16])
			symIdx := uint32(info >> 32)
			typ := elf.R_AARCH64(info & 0xffffffff)

			// 只处理JUMP_SLOT类型
			if typ != elf.R_AARCH64_JUMP_SLOT {
				continue
			}

			// 获取符号信息
			symEntrySize := 24 // Elf64_Sym结构体大小
			if int(symIdx)*symEntrySize >= len(symData) {
				continue
			}

			symEntry := symData[symIdx*uint32(symEntrySize):]
			nameOffset := binary.LittleEndian.Uint32(symEntry[0:4]) 

			if int(nameOffset) >= len(strdata) {
				continue
			}

			// 提取符号名
			var name string
			for j := nameOffset; strdata[j] != 0; j++ {
				name += string(strdata[j])
			}

			// 计算PLT条目地址
			// GOT偏移公式: (offset - gotplt.Addr) / 8 - 3
			pltIndex := (offset - gotplt.Addr)/8 - 3

			pltAddr := plt.Addr + 32 + pltIndex*16 // PLT[0]占16字节

			this.Symbols[libbase+pltAddr] = name
			// fmt.Printf("%x: +%x %s\n", libbase+pltAddr, pltIndex, name)
		}
	}

	return nil
}

func findSymbolOffset(f *elf.File, sym elf.Symbol) (uint64, error) {
	for _, prog := range f.Progs {
		if prog.Type == elf.PT_LOAD {
			start := prog.Vaddr
			end := start + prog.Filesz
			if sym.Value >= start && sym.Value < end {
				return sym.Value - start + prog.Off, nil
			}
		}
	}
	return sym.Value, nil
}

func (this *Process) GetSymbol(address uint64) string {
	if sym, ok := this.Symbols[address]; ok {
		return fmt.Sprintf("<%s>", sym)
	}
	addressInfo, err := this.ParseAddress(address)
	if err == nil {
		if addressInfo.LibInfo.SymbolExtracted == true {
			return fmt.Sprintf("<%s+%x>", addressInfo.LibInfo.LibName, addressInfo.Offset)
		}
		err := this.ExportSymbols(address-addressInfo.Offset, addressInfo.LibInfo.RealFilePath, addressInfo.LibInfo.NonElfOffset)
		if err != nil {
			// fmt.Printf("WARNING: Cannot get symbols from %s\n", addressInfo.LibInfo.RealFilePath)
		}
		addressInfo.LibInfo.SymbolExtracted = true
		if sym, ok := this.Symbols[address]; ok {
			return sym
		}
		return fmt.Sprintf("<%s+%x>", addressInfo.LibInfo.LibName, addressInfo.Offset)
	}
	return ""
}

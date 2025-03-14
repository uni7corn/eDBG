package controller

import (
	// "fmt"
	"os"
	"path/filepath"
	"fmt"
	"archive/zip"
	"io"
	"strings"
)

type Address struct {
	LibInfo *LibraryInfo
	Offset uint64
	Permission string
}

type CachedLibInfo struct {
	LibInfo *LibraryInfo
	BaseAddr uint64
	Offset uint64
	EndAddr uint64
	Permission string
}

var DoneLib []*CachedLibInfo

func NewAddress(libInfo *LibraryInfo, offset uint64) *Address {
	return &Address{
		LibInfo: libInfo,
		Offset: offset,
		Permission: "rwxp",
	}
}

func (this *Process) ParseAddress(address uint64) (*Address, error) {
	for _, lib := range DoneLib {
		if lib.BaseAddr <= address && address < lib.EndAddr {
			return &Address {
				LibInfo: lib.LibInfo,
				Offset: address - lib.BaseAddr + lib.Offset,
                Permission: lib.Permission,
			}, nil
		}
	}
	return this.ParseAddressNew(address)
}


func (this *Process) ParseAddressNew(address uint64) (*Address, error) {
	maps, exsists := this.ProcMaps[this.WorkPid]
	if exsists == false {
		return &Address{}, fmt.Errorf("ParseAddress: Bad pid: %x", this.WorkPid)
	}
	addressParsed, err := maps.ParseAbsoluteAddress(this, address)
	if err != nil {
		return &Address{}, err
	}
	
	return addressParsed, nil
}

func (this *ProcMaps) ParseAbsoluteAddress(process *Process, address uint64) (*Address, error) {
    for _, seg := range this.segments {
        if seg.baseAddr <= address && address < seg.endAddr {
            if strings.HasPrefix(seg.libPath, "UNNAMED") || strings.HasPrefix(seg.libPath, "[") {
                return &Address{}, fmt.Errorf("Failed to parse %x: anouymous address", address)
            }

			// fmt.Printf("Lib found in %s\n", seg.libPath)

            if strings.HasSuffix(seg.libName, ".apk") {
                apk_path := seg.libPath
                lib := &LibraryInfo{Process: process, RealFilePath: apk_path}
                off := address - seg.baseAddr + seg.off
				// fmt.Printf("Processing apk: %x in %x offset %x\n", address, seg.baseAddr, off)
                zf, err := zip.OpenReader(apk_path)
                if err != nil {
                    return &Address{}, err
                }
                for _, f := range zf.File {
                    if strings.HasPrefix(f.Name, "lib/arm64-v8a") {
                        offset, err := f.DataOffset()
                        if err != nil {
                            return &Address{}, err
                        }
						// fmt.Printf("Processing file %s, range %x-%x\n", f.Name, uint64(offset), f.UncompressedSize64)
                        if uint64(offset) <= off && off < uint64(offset) + f.UncompressedSize64 {
							// fmt.Printf("Checked OK!")
                            parts := strings.Split(f.Name, "/")
                            lib.LibName = parts[len(parts)-1]
                            lib.LibPath = filepath.Join(process.ExecPath, lib.LibName)
                            lib.NonElfOffset = uint64(offset)
                            if _, err := os.Stat(lib.LibPath); err != nil {
                                srcFile, err := f.Open()
                                if err != nil {
                                    return &Address{}, err
                                }
                                dstFile, err := os.OpenFile(lib.LibPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
                                if err != nil {
                                    panic(err)
                                }
                                if _, err := io.Copy(dstFile, srcFile); err != nil {
                                    panic(err)
                                }
                                dstFile.Close()
                                srcFile.Close()
                            }

							DoneLib = append(DoneLib, &CachedLibInfo{
								LibInfo: lib,
								BaseAddr: seg.baseAddr+uint64(offset)-seg.off,
								EndAddr: seg.baseAddr+uint64(offset)+f.UncompressedSize64-seg.off,
								Offset: 0,
								Permission: seg.permission,
							})


                            return &Address{
                                LibInfo: lib,
                                Offset: off - uint64(offset),
                                Permission: seg.permission,
                            }, nil
                        }

                    }
                }

            } else {
                res := &Address{
                    LibInfo: &LibraryInfo{
                        LibName: seg.libName,
                        Process: process,
                    },
                    Offset: address - seg.baseAddr + seg.off,
                    Permission: seg.permission,
                }
                err := res.LibInfo.ParseLibrary()
				if err != nil {
					return &Address{}, err
				}
				DoneLib = append(DoneLib, &CachedLibInfo{
					LibInfo: res.LibInfo,
					BaseAddr: seg.baseAddr,
					EndAddr: seg.endAddr,
					Offset: seg.off,
					Permission: seg.permission,
				})
                return res, nil
            }
        }
    }
    return &Address{}, fmt.Errorf("Failed to parse %x: anouymous address", address)
}

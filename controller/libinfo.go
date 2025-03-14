package controller

import (
	"strings"
	"os"
	"errors"
	"path/filepath"
	"fmt"
	"archive/zip"
	"io"
	"golang.org/x/exp/slices"
)

type LibraryInfo struct {
	LibName string
	LibPath string
	RealFilePath string
	NonElfOffset uint64
	Process *Process
}

func CreateLibrary(process *Process, libName string) (*LibraryInfo, error) {
	libInfo := &LibraryInfo{}
	libInfo.LibName = libName
	libInfo.Process = process
	if err := libInfo.ParseLibrary(); err != nil {
		return &LibraryInfo{}, err
	}
	return libInfo, nil
}

func (this *LibraryInfo) ParseLibrary() error {
	if this.LibName == "" {
		return fmt.Errorf("ParseLibrary: No Library Name Specified.")
	}
	if strings.HasPrefix(this.LibName, "/") {
        if _, err := os.Stat(this.LibName); err != nil {
            return err
        }
        this.LibPath = this.LibName
		this.RealFilePath = this.LibName
	} else {
		err := this.LocateLibrary()
		if err != nil {
			return err
		}
	}
	return nil
	// fmt.Printf("Found library at %s\n", libInfo.RealFilePath)
}

func (this *LibraryInfo) LocateLibrary() error {
	if this.Process == nil {
		return errors.New("LocateLibrary: process info not found")
	}
	SearchPath := this.Process.GetLibSearchPaths()
	
	var full_paths []string
    for _, paths := range SearchPath {
        check_path := strings.TrimRight(paths, "/") + "/" + this.LibName
        _, err := os.Stat(check_path)
        if err != nil {
            continue
        }
        path_info, err := os.Lstat(check_path)
        if err != nil {
            continue
        }
        if path_info.Mode()&os.ModeSymlink != 0 {
            real_path, err := filepath.EvalSymlinks(check_path)
            if err != nil {
                continue
            }
            check_path = real_path
        }
        if !slices.Contains(full_paths, check_path) {
            full_paths = append(full_paths, check_path)
        }
    }

    if len(full_paths) == 0 {
		// 在 apk 里找
		lib_search_paths := []string{"lib/arm64-v8a"}
    	for _, apk_path := range SearchPath {
        	if !strings.HasSuffix(apk_path, ".apk") {
           		continue
        	}
			zf, err := zip.OpenReader(apk_path)
			if err != nil {
				return err
			}

			for _, f := range zf.File {
				for _, search_path := range lib_search_paths {
					check_path := search_path + "/" + this.LibName
					if f.Name == check_path {
						srcFile, err := f.Open()
						if err != nil {
							return err
						}

						this.RealFilePath = apk_path
						this.LibPath = filepath.Join(this.Process.ExecPath, this.LibName)
						dstFile, err := os.OpenFile(this.LibPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
						if err != nil {
							panic(err)
						}
						if _, err := io.Copy(dstFile, srcFile); err != nil {
							panic(err)
						}
						dstFile.Close()
						srcFile.Close()
						offset, err := f.DataOffset()
						if err != nil {
							return err
						}
						this.NonElfOffset = uint64(offset)
						return nil
					}
                }
            }
        }
		return errors.New(fmt.Sprintf("LocateLibrary: can not find %s in any apk", this.LibName))
    }
    // 在已有的搜索路径下可能存在多个同名的库 提示用户指定全路径
    if len(full_paths) > 1 {
        return fmt.Errorf("LocateLibrary: find %d libs with the same name\n%s", len(full_paths), strings.Join(full_paths[:], "\n\t"))
    }
    // 修正为完整路径
    this.LibPath = full_paths[0]
	this.RealFilePath = this.LibPath
	return nil
}
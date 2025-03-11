package utils

import (
	"path"
	"syscall"
	"errors"
	"fmt"
	"os"
	"strings"
	"archive/zip"
)

type Process struct {
	pidList []uint32
	packageName string
	libName string
	execPath string
	libPath string
	realFilePath string
	nonElfOffset uint64
	procMaps map[uint32]*ProcMaps
}

func NewProcess(packageName string, libName string) (*Process, error) {
	process := &Process{}
	process.packageName = packageName
	process.libName = libName
	err := process.GetExecPath()
	if err != nil {
		return Process, err
	}
	err := process.checkPackageName()
	if err != nil {
		return Process{}, err
	}
	err := process.ParseLibInfo()
	if err != nil {
		return Process{}, err
	}
	return process, nil
}

func (this *Process) GetExecPath() error {
	exec_path, err := os.Executable()
    if err != nil {
        return fmt.Errorf("please build as executable binary, %v", err)
    }
	this.execPath = path.Dir(exec_path)
	return nil
}

func (this *Process) UpdatePidList() {
	this.pidList = []uint32{}
    content, err := RunCommand("sh", "-c", "ps -ef -o name,pid,ppid | grep ^"+name)
    if err != nil {
        return
    }
    lines := strings.Split(content, "\n")
    for _, line := range lines {
        parts := strings.Fields(line)
        value, err := strconv.ParseUint(parts[1], 10, 32)
        if err != nil {
            panic(err)
        }
        this.pidList = append(this.pidList, uint32(value))
    }
	this.procMaps := make(map[uint32]*ProcMaps)
	for _, pid := range pidList {
		procMaps, err := GetProcMaps(pid)
		if err == nil {
			this.procMaps[pid] := procMaps
		}
	}
    return
}

func (this *Process) checkPackageName() error {
	packageinfos := GetPackageInfos()
	_, err := packageinfos.FindPackageByName(this.packageName)
	return err
}


func (this *Process) SearchLib() error {
	if process.libName == "" {
		return errors.New("Empty library name")
	}
	if process.packageName == "" {
		return errors.New("Empty package name")
	}

	if strings.HasPrefix(process.libName, "/") {
        if _, err := os.Stat(process.libName); err != nil {
            return err
        }
        process.libPath = process.libName
		process.realFilePath = process.libName
		return nil
	}
	
	SearchPath := []string{
        "/system/lib64",
        "/apex/com.android.art/lib64",
        "/apex/com.android.conscrypt/lib64",
        "/apex/com.android.runtime/bin",
        "/apex/com.android.runtime/lib64/bionic",
    }
	this.UpdatePidList()
	for _, mapsInfo := range this.maps {
		mapsPaths := mapsInfo.GetLibSearchPaths()
		for _, paths := range mapsPaths {
			if !slices.Contains(SearchPath, paths) {
				SearchPath = append(SearchPath, paths)
			}
		}
	}

	pkgPaths := FindLibPathFromPackage(this.packageName)
	for _, path := range pkgPaths {
		if !slices.Contains(SearchPath, path) {
			SearchPath = append(SearchPath, path)
		}
	}

	var full_paths []string
    for _, paths := range SearchPath {
        // 去掉末尾可能存在的 /
        check_path := strings.TrimRight(paths, "/") + "/" + library
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
        	// 确保只检查 .apk
        	if !strings.HasSuffix(apk_path, ".apk") {
           		continue
        	}
			// 读取 .apk
			zf, err := zip.OpenReader(apk_path)
			if err != nil {
				return err
			}
			for _, f := range zf.File {
				for _, search_path := range lib_search_paths {
					// 这里是存在重复的可能的 不过不考虑这种情况
					check_path := search_path + "/" + library
					if f.Name == check_path {
						srcFile, err := f.Open()
						if err != nil {
							return err
						}
						// apk 路径作为最终的 uprobe 注册文件
						this.realFilePath = apk_path
						this.libPath = filepath.Join(this.execPath, library)
						dstFile, err := os.OpenFile(this.libPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
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
						sconfig.nonElfOffset = uint64(offset)
						return nil
                }
            }
        }
    }

    return errors.New(fmt.Sprintf("can not find %s in any apk", library))
    }
    // 在已有的搜索路径下可能存在多个同名的库 提示用户指定全路径
    if len(full_paths) > 1 {
        return fmt.Errorf("find %d libs with the same name\n%s", len(full_paths), strings.Join(full_paths[:], "\n\t"))
    }
    // 修正为完整路径
    this.libPath = full_paths[0]
	this.realFilePath = this.libPath
}

func FindLibPathFromPackage(name string) []string {
	SearchPath := []string{}
    content, err := RunCommand("pm", "path", name)
    if err != nil {
        panic(err)
    }
    for _, line := range strings.Split(content, "\n") {
        parts := strings.Split(line, ":")
        if len(parts) == 2 {
            // 将 apk 文件也作为搜索路径
            apk_path := parts[1]
            _, err := os.Stat(apk_path)
            if err == nil {
                SearchPath = append(SearchPath, apk_path)
            }
            // 将 apk + /lib/arm64 作为搜索路径
            items := strings.Split(parts[1], "/")
            lib_search_path := strings.Join(items[:len(items)-1], "/") + "/lib/arm64"
            _, err = os.Stat(lib_search_path)
            if err == nil {
                SearchPath = append(SearchPath, lib_search_path)
            }
        }
    }
	return SearchPath
}

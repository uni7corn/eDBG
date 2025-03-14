package controller

import (
	"io/ioutil"
	"strings"
    "errors"
	"fmt"
    "golang.org/x/exp/slices"
)



type Segment struct {
    baseAddr uint64
    off      uint64
    endAddr  uint64
    libPath  string
    libName  string
    permission string
}

type ProcMaps struct {
	pid uint32
	segments []*Segment
}

func (this *Segment) ParseLib() {
    parts := strings.Split(this.libPath, "/")
    this.libName = parts[len(parts)-1]
}

func GetProcMaps(pid uint32) (*ProcMaps, error) {
	procMaps := &ProcMaps{}
	procMaps.pid = pid
	err := procMaps.ReadMaps()
	if err != nil {
		return &ProcMaps{}, err
	}
    return procMaps, nil
}



func (this *ProcMaps) ReadMaps() error {
	if this.pid == 0 {
		return errors.New("Unexpected: pid is not ready")
	}
	filename := fmt.Sprintf("/proc/%d/maps", this.pid)
    content, err := ioutil.ReadFile(filename)
    if err != nil {
        return fmt.Errorf("Error when opening file:%v", err)
    }
    return this.ParseMapsContent(content)
}

func (this *ProcMaps) ParseMapsContent(content []byte) error {
	var (
        seg_start  uint64
        seg_end    uint64
        permission string
        seg_offset uint64
        device     string
        inode      uint64
        seg_path   string
    )
	this.segments = []*Segment{}

    for _, line := range strings.Split(string(content), "\n") {
        reader := strings.NewReader(line)
        n, err := fmt.Fscanf(reader, "%x-%x %s %x %s %d %s", &seg_start, &seg_end, &permission, &seg_offset, &device, &inode, &seg_path)
        if err == nil && n == 7 {
            if seg_path == "" {
                seg_path = fmt.Sprintf("UNNAMED_0x%x", seg_start)
            }
            new_info := &Segment{
                baseAddr: seg_start,
                off:      seg_offset,
                endAddr:  seg_end,
                libPath:  seg_path,
                permission: permission,
            }
            new_info.ParseLib()
			this.segments = append(this.segments, new_info)
        }
    }
    return nil
}

func (this *ProcMaps) GetLibSearchPaths() []string {
	search_paths := []string{}
	for _, libInfo := range this.segments {
		if strings.HasPrefix(libInfo.libPath, "/") && strings.HasSuffix(libInfo.libPath, ".so") {
            items := strings.Split(libInfo.libPath, "/")
            lib_search_path := strings.Join(items[:len(items)-1], "/")
            if !slices.Contains(search_paths, lib_search_path) {
                search_paths = append(search_paths, lib_search_path)
            }
        }
	}
	return search_paths
}
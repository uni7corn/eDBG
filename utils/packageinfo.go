package utils

import (
	"io/ioutil"
	"fmt"
	"strings"
	"strconv"
)

type PackageInfo struct {
	Name string
	Uid  uint32
}

type PackageInfos struct {
	items []PackageInfo
}

func GetPackageInfos() *PackageInfos {
	// https://zhuanlan.zhihu.com/p/31124919
	// /data/system/packages.list
	content, err := ioutil.ReadFile("/data/system/packages.list")
	if err != nil {
		panic(err)
	}
	var pis PackageInfos
	lines := strings.TrimSpace(string(content))
	for _, line := range strings.Split(lines, "\n") {
		parts := strings.Split(line, " ")
		value, err := strconv.ParseUint(parts[1], 10, 32)
		if err != nil {
			panic(err)
		}
		pis.items = append(pis.items, PackageInfo{parts[0], uint32(value)})
	}
	return &pis
}

func (this *PackageInfos) FindPackageByName(name string) (PackageInfo, error) {
	for _, item := range this.items {
		if item.Name == name {
			return item, nil
		}
	}
	return PackageInfo{}, fmt.Errorf("Failed to find package: %s", name)
}
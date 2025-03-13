package module

import (
	"eDBG/controller"
	"fmt"
	// "errors"
	manager "github.com/gojue/ebpfmanager"
)

type IEventListener interface {
    OnEvent(int, []byte, *manager.PerfMap, *manager.Manager)
}

type BreakPoint struct {
	libInfo controller.LibraryInfo
	offset uint64
	enable bool
}

type BreakPointManager struct {
	breakPoints []*BreakPoint
	probeHandler *ProbeHandler
}

func CreateBreakPointManager(listener IEventListener) *BreakPointManager {
	return &BreakPointManager{probeHandler: CreateProbeHandler(listener)}
}

func (this *BreakPointManager) CreateBreakPoint(libInfo controller.LibraryInfo, offset uint64) error {
	for _, brk := range this.breakPoints {
		if brk.libInfo.LibName == libInfo.LibName && offset == brk.offset {
			if brk.enable == false {
				brk.enable = true
			} else {
				// return fmt.Errorf("BreakPoint %x exsists")
			}
			return nil
		}
	}
	brk := &BreakPoint{
		libInfo: libInfo,
		offset: offset,
		enable: true,
	}
	this.breakPoints = append(this.breakPoints, brk)
	return nil
}

func (this *BreakPointManager) SetupProbe() error {
	// err := probeHandler.Init()
	// if err != nil {
	// 	return err
	// }
	err := this.probeHandler.SetupManager(this.breakPoints)
	if err != nil {
		return err
	}
	err = this.probeHandler.Run()
	if err != nil {
		return err
	}
	return nil
}

func (this *BreakPointManager) Start(libInfo controller.LibraryInfo, brkAddrs []uint64) error {
	for _, offset := range brkAddrs {
		err := this.CreateBreakPoint(libInfo, offset)
		if err != nil {
			fmt.Printf("Create Breakpoints Failed: %v, skipped.", err)
			continue
		}
	}
	return this.SetupProbe()
}

func (this *BreakPointManager) AddBreakPoint(libInfo controller.LibraryInfo, offset uint64) error {
	err := this.probeHandler.Stop()
	if err != nil {
		return err
	}
	err = this.CreateBreakPoint(libInfo, offset)
	if err != nil {
		return fmt.Errorf("Create Breakpoints Failed: %v, skipped.", err)
	}
	return this.SetupProbe()
}

func (this *BreakPointManager) Stop()  {
	_ = this.probeHandler.Stop()
}
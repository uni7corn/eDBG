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
	libInfo *controller.LibraryInfo
	offset uint64
	enable bool
	deleted bool
}

type BreakPointManager struct {
	breakPoints []*BreakPoint
	temporaryBreakPoint *BreakPoint
	hasTempBreak bool
	probeHandler *ProbeHandler
}

func CreateBreakPointManager(listener IEventListener) *BreakPointManager {
	return &BreakPointManager{probeHandler: CreateProbeHandler(listener), hasTempBreak: false}
}

func checkOffset(offset uint64) bool {
	return offset%4 == 0
}

func (this *BreakPointManager) SetTempBreak(address *controller.Address) error {
	offset := address.Offset
	libInfo := address.LibInfo
	if checkOffset(offset) == false {
		return fmt.Errorf("Invalid address: %x", offset)
	}
	for _, brk := range this.breakPoints {
		if brk.libInfo.LibName == libInfo.LibName && offset == brk.offset && brk.enable == true {
			return nil
		}
	}

	brk := &BreakPoint{
		libInfo: libInfo,
		offset: offset,
		enable: true,
		deleted: false,
	}
	this.temporaryBreakPoint = brk
	this.hasTempBreak = true
	return nil
}

func (this *BreakPointManager) CreateBreakPoint(address *controller.Address) error {
	offset := address.Offset
	libInfo := address.LibInfo
	if checkOffset(offset) == false {
		return fmt.Errorf("Invalid address: %x", offset)
	}
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
		deleted: false,
	}
	this.breakPoints = append(this.breakPoints, brk)
	return nil
}

func (this *BreakPointManager) SetupProbe() error {
	// err := probeHandler.Init()
	// if err != nil {
	// 	return err
	// }
	if this.hasTempBreak == true {
		err := this.probeHandler.SetupManager(append(this.breakPoints, this.temporaryBreakPoint))
		if err != nil {
			return err
		}
		this.hasTempBreak = false
	} else {
		err := this.probeHandler.SetupManager(this.breakPoints)
		if err != nil {
			return err
		}
	}
	
	
	err := this.probeHandler.Run()
	// fmt.Println("probe is running..")
	if err != nil {
		return err
	}
	return nil
}

func (this *BreakPointManager) Start(libInfo *controller.LibraryInfo, brkAddrs []uint64) error {
	for _, offset := range brkAddrs {
		err := this.CreateBreakPoint(controller.NewAddress(libInfo, offset))
		if err != nil {
			fmt.Printf("Create Breakpoints Failed: %v, skipped.", err)
			continue
		}
	}
	return this.SetupProbe()
}

func (this *BreakPointManager) Stop() error {
	return this.probeHandler.Stop()
}

func (this *BreakPointManager) PrintBreakPoints() {
	for id, brk := range this.breakPoints {
		if brk.deleted {
			continue
		}
		if !brk.enable {
			fmt.Printf("[-] ")
		} else {
			fmt.Printf("[+] ")
		}
		fmt.Printf("%d: %s+%x\n", id, brk.libInfo.LibName, brk.offset)
	}
}

func (this *BreakPointManager) ChangeBreakPoint(id int, status bool) {
	if id >= len(this.breakPoints) {
		fmt.Println("Breakpoint doesn't exist.")
		return
	}
	if this.breakPoints[id].deleted {
		fmt.Println("Breakpoint doesn't exist.")
		return
	}
	this.breakPoints[id].enable = status
}

func (this *BreakPointManager) DeleteBreakPoint(id int) {
	if id > len(this.breakPoints) {
		fmt.Println("Breakpoint doesn't exist.")
		return
	}
	if this.breakPoints[id].deleted {
		fmt.Println("Breakpoint doesn't exist.")
		return
	}
	this.breakPoints[id].enable = false
	this.breakPoints[id].deleted = true
}
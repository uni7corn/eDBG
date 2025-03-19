package module

import (
	"eDBG/controller"
	"fmt"
	manager "github.com/gojue/ebpfmanager"
)

type IEventListener interface {
    OnEvent(int, []byte, *manager.PerfMap, *manager.Manager)
}

type BreakPoint struct {
	LibInfo *controller.LibraryInfo
	Offset uint64
	Enable bool
	Deleted bool
}

type BreakPointManager struct {
	BreakPoints []*BreakPoint
	temporaryBreakPoint *BreakPoint
	HasTempBreak bool
	probeHandler *ProbeHandler
	TempBreakTid uint32
}

func CreateBreakPointManager(listener IEventListener, BTF_File string) *BreakPointManager {
	return &BreakPointManager{
		probeHandler: CreateProbeHandler(listener, BTF_File), 
		HasTempBreak: false,
	}
}

func checkOffset(offset uint64) bool {
	return offset%4 == 0
}

func (this *BreakPointManager) SetTempBreak(address *controller.Address, tid uint32) error {
	offset := address.Offset
	libInfo := address.LibInfo
	if checkOffset(offset) == false {
		return fmt.Errorf("Invalid address: %x", offset)
	}
	for _, brk := range this.BreakPoints {
		if brk.LibInfo.LibName == libInfo.LibName && offset == brk.Offset && brk.Enable == true {
			return nil
		}
	}

	brk := &BreakPoint{
		LibInfo: libInfo,
		Offset: offset,
		Enable: true,
		Deleted: false,
	}
	this.TempBreakTid = tid
	this.temporaryBreakPoint = brk
	this.HasTempBreak = true
	return nil
}

func (this *BreakPointManager) CreateBreakPoint(address *controller.Address, enable bool) error {
	offset := address.Offset
	libInfo := address.LibInfo
	if checkOffset(offset) == false {
		return fmt.Errorf("Invalid address: %x", offset)
	}
	for _, brk := range this.BreakPoints {
		if brk.LibInfo.LibName == libInfo.LibName && offset == brk.Offset {
			if brk.Enable != enable {
				brk.Enable = enable
			} else {
				// return fmt.Errorf("BreakPoint %x exsists")
			}
			return nil
		}
	}
	brk := &BreakPoint{
		LibInfo: libInfo,
		Offset: offset,
		Enable: enable,
		Deleted: false,
	}
	this.BreakPoints = append(this.BreakPoints, brk)
	return nil
}

func (this *BreakPointManager) SetupProbe() error {
	// err := probeHandler.Init()
	// if err != nil {
	// 	return err
	// }
	if this.HasTempBreak == true {
		err := this.probeHandler.SetupManager(append(this.BreakPoints, this.temporaryBreakPoint))
		if err != nil {
			return err
		}
		this.HasTempBreak = false
	} else {
		this.TempBreakTid = 0
		err := this.probeHandler.SetupManager(this.BreakPoints)
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

func (this *BreakPointManager) Start(addresss []*controller.Address) error {
	for _, addr := range addresss {
		err := this.CreateBreakPoint(addr, true)
		if err != nil {
			fmt.Printf("Create Breakpoints Failed: %v, skipped.\n", err)
			continue
		}
	}
	err := this.probeHandler.SetupManagerOptions()
	if err != nil {
		return err
	}
	return this.SetupProbe()
}

func (this *BreakPointManager) Stop() error {
	return this.probeHandler.Stop()
}

func (this *BreakPointManager) PrintBreakPoints() {
	for id, brk := range this.BreakPoints {
		if brk.Deleted {
			continue
		}
		if !brk.Enable {
			fmt.Printf("[-] ")
		} else {
			fmt.Printf("[+] ")
		}
		fmt.Printf("%d: %s+%x\n", id, brk.LibInfo.LibName, brk.Offset)
	}
}

func (this *BreakPointManager) ChangeBreakPoint(id int, status bool) {
	if id >= len(this.BreakPoints) {
		fmt.Println("Breakpoint doesn't exist.")
		return
	}
	if this.BreakPoints[id].Deleted {
		fmt.Println("Breakpoint doesn't exist.")
		return
	}
	this.BreakPoints[id].Enable = status
}

func (this *BreakPointManager) DeleteBreakPoint(id int) {
	if id > len(this.BreakPoints) {
		fmt.Println("Breakpoint doesn't exist.")
		return
	}
	if this.BreakPoints[id].Deleted {
		fmt.Println("Breakpoint doesn't exist.")
		return
	}
	this.BreakPoints[id].Enable = false
	this.BreakPoints[id].Deleted = true
}
package module

import (
	"eDBG/controller"
	"eDBG/utils"
	"fmt"
	"github.com/cilium/ebpf/perf"
	manager "github.com/gojue/ebpfmanager"
)

type IEventListener interface {
	SendRecord(rec perf.Record)
    OnEvent(int, []byte, *manager.PerfMap, *manager.Manager)
}

type BreakPoint struct {
	LibInfo *controller.LibraryInfo
	Offset uint64 // Absolute address when hardware
	Enable bool
	Deleted bool
	Hardware bool
	Pid uint32
	Type int
}

type BreakPointManager struct {
	process *controller.Process
	EnableHW bool
	BreakPoints []*BreakPoint
	temporaryBreakPoint *BreakPoint
	HasTempBreak bool
	ProbeHandler *ProbeHandler
	TempBreakTid uint32
	TempAddressAbsolute uint64
}

func CreateBreakPointManager(listener IEventListener, BTF_File string, process *controller.Process, EnableHW bool) *BreakPointManager {
	return &BreakPointManager{
		process: process,
		ProbeHandler: CreateProbeHandler(listener, BTF_File), 
		HasTempBreak: false,
		EnableHW: EnableHW,
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
		Hardware: false,
		Pid: this.process.WorkPid,
	}
	this.TempAddressAbsolute = address.Absolute
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
		Hardware: false,
		Enable: enable,
		Deleted: false,
		Pid: this.process.WorkPid,
	}
	this.BreakPoints = append(this.BreakPoints, brk)
	return nil
}

func (this *BreakPointManager) CreateHWBreakPoint(address *controller.Address, enable bool, Type int) error {
	// 存在问题，先不使用
	for _, brk := range this.BreakPoints {
		if address.Absolute == brk.Offset {
			if brk.Enable != enable {
				brk.Enable = enable
			} else {
				// return fmt.Errorf("BreakPoint %x exsists")
			}
			return nil
		}
	}
	brk := &BreakPoint{
		LibInfo: &controller.LibraryInfo{LibName: "Hardware breakpoint!"},
		Offset: address.Absolute,
		Hardware: true,
		Enable: enable,
		Deleted: false,
		Pid: this.process.WorkPid,
		Type: Type,
	}
	this.BreakPoints = append(this.BreakPoints, brk)
	return nil
}


func (this *BreakPointManager) UseUprobe() error {
	return this.ProbeHandler.SetupManager(append(this.BreakPoints, this.temporaryBreakPoint))
}

func (this *BreakPointManager) UsePerf() error {
	return this.ProbeHandler.SetupManager(append(this.BreakPoints, &BreakPoint{
		Offset: this.TempAddressAbsolute,
		Hardware: true,
		Enable: true,
		Deleted: false,
		Pid: this.process.WorkPid,
		Type: 4,
	}))
}

func (this *BreakPointManager) SetupProbe() error {
	if this.HasTempBreak == true {
		if this.EnableHW {
			safe, err := utils.SafeAddress(this.process.WorkPid, this.TempAddressAbsolute)
			// 如果断点是跳转指令，则我们需要 pstate 寄存器来预测下一条指令位置，因此必须使用 uprobe。
			// 巧合的是，此时使用 uprobe 是安全的
			// 非常不巧的是正因为如此对于 uprobe 无法工作的场景，即使支持用户的硬件断点也无法单步调试...
			// 总之先把改动的空间留出来，万一之后想到怎么办了呢
			if err != nil {
				fmt.Printf("Failed parse current addr: %v\n", this.TempAddressAbsolute, err)
			}
			if !safe {
				err = this.UsePerf()
				if err != nil {
					fmt.Printf("Failed to open perf event at %x: %v\n", this.TempAddressAbsolute, err)
					err = this.UseUprobe()
					if err != nil {
						return err
					}
				}
			} else {
				err := this.UseUprobe()
				if err != nil {
					return err
				}
			}
		} else {
			err := this.UseUprobe()
			if err != nil {
				return err
			}
		}
		this.HasTempBreak = false
	} else {
		this.TempBreakTid = 0
		err := this.ProbeHandler.SetupManager(this.BreakPoints)
		if err != nil {
			return err
		}
	}
	
	
	err := this.ProbeHandler.Run()
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
	err := this.ProbeHandler.SetupManagerOptions()
	if err != nil {
		return err
	}
	return this.SetupProbe()
}

func (this *BreakPointManager) Stop() error {
	return this.ProbeHandler.Stop()
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
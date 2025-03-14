package controller

import (
	"fmt"
	"eDBG/utils"
)

type ProcessContext struct {
	Regs []uint64
	LR uint64
	PC uint64
	SP uint64
	Pstate uint64
}

func (this *ProcessContext) GetReg(key int) uint64 {
	return this.Regs[key]
}
func (this *ProcessContext) GetPC() uint64 {
	return this.PC
}
func (this *ProcessContext) GetSP() uint64 {
	return this.SP
}
func (this *ProcessContext) GetLR() uint64 {
	return this.LR
}
func (this *ProcessContext) GetPstate() uint64 {
	return this.Pstate
}


func (this *Process) PrintContext() {
	for i, reg := range(this.Context.Regs) {
		tmpreg := reg
		if (tmpreg >> 96) == 0xB400 {
			tmpreg -= 0xB400000000000000
		}
		if tmpreg > 0x7000000000 {
			addrinfo, err := this.ParseAddress(tmpreg)
			if err == nil {
				fmt.Printf("*X%d\t0x%X(%s+%x)\n", i, reg, addrinfo.LibInfo.LibName, addrinfo.Offset)
				continue
			}
		}
		if tmpreg > 0x5000000000 {
			ok, info := utils.TryRead(this.WorkPid, uintptr(tmpreg))
			if ok == true {
				fmt.Printf("*X%d\t0x%X ◂— %s\n", i, reg, info)
				continue
			}
		}
		fmt.Printf(" X%d\t0x%X\n", i, reg)
	}
	addrinfo, err := this.ParseAddress(this.Context.LR)
	if err == nil {
		fmt.Printf("*LR\t0x%X(%s+%x)\n", this.Context.LR, addrinfo.LibInfo.LibName, addrinfo.Offset)
	} else {
		fmt.Printf(" LR\t0x%X\t", this.Context.LR)
	}
	fmt.Printf("*SP\t0x%X\n", this.Context.SP)

	addrinfo, err = this.ParseAddress(this.Context.PC)
	if err == nil {
		fmt.Printf("*PC\t0x%X(%s+%x)\n", this.Context.PC, addrinfo.LibInfo.LibName, addrinfo.Offset)
	} else {
		fmt.Printf(" PC\t0x%X\n", this.Context.PC)
	}
	// fmt.Println("─────────────────────────────────────────────────────────────────────────────────────────")
}


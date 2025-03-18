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
	if key <= 29 {
		return this.Regs[key] & 0xFFFFFFFF
	}
	if key == 30 {
		return this.LR
	}
	if key == 31 {
		return this.SP
	}
	return this.Regs[key-32]
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
		if tmpreg > 0x1000000000 {
			ok, info := utils.TryRead(this.WorkPid, uintptr(tmpreg))
			if ok == true {
				fmt.Printf("*X%d\t0x%X%s ◂— %s\n", i, reg, this.GetSymbol(tmpreg), info)
				continue
			} else {
				fmt.Printf("*X%d\t0x%X%s??\n", i, reg, this.GetSymbol(tmpreg))
				continue
			}
		}

		fmt.Printf(" X%d\t0x%X\n", i, reg)
	}
	_, err := this.ParseAddress(this.Context.LR)
	if err == nil {
		fmt.Printf("*LR\t0x%X%s\n", this.Context.LR, this.GetSymbol(this.Context.LR))
	} else {
		fmt.Printf(" LR\t0x%X\t", this.Context.LR)
	}
	fmt.Printf("*SP\t0x%X\n", this.Context.SP)

	_, err = this.ParseAddress(this.Context.PC)
	if err == nil {
		fmt.Printf("*PC\t0x%X%s\n", this.Context.PC, this.GetSymbol(this.Context.PC))
	} else {
		fmt.Printf(" PC\t0x%X\n", this.Context.PC)
	}
	// fmt.Println("─────────────────────────────────────────────────────────────────────────────────────────")
}


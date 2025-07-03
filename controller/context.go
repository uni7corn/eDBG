package controller

import (
	"eDBG/config"
	"eDBG/utils"
	"fmt"
)

type ProcessContext struct {
	Regs   []uint64
	LR     uint64
	PC     uint64
	SP     uint64
	Pstate uint64
}

func AssembleRegisters(p *ProcessContext) []uint64 {
	regs := make([]uint64, 0, 33)
	regs = append(regs, p.Regs...)
	regs = append(regs, p.LR, p.SP, p.PC)
	return regs
}

func (this *ProcessContext) GetReg(key int) uint64 {
	// W系寄存器
	if key <= 29 {
		return this.Regs[key] & 0xFFFFFFFF
	}
	if key == 30 {
		return this.LR
	}
	if key == 31 {
		return this.SP
	}
	key -= 32 // X系寄存器
	if key <= 29 {
		return this.Regs[key]
	}
	if key == 30 {
		return this.LR
	}
	if key == 31 {
		return this.SP
	}
	fmt.Println("WARNING: Unsupported register type")
	return 0 //不支持
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
	for i, reg := range this.Context.Regs {
		tmpreg := reg
		if (tmpreg >> 96) == 0xB400 {
			tmpreg -= 0xB400000000000000
		}
		if tmpreg > 0x1000000000 {
			ok, info := utils.TryRead(this.WorkPid, uintptr(tmpreg))
			if ok == true {
				fmt.Printf("%s*X%d%s\t%s0x%X%s%s ◂— %s\n", config.RED, i, config.NC, config.CYAN, reg, config.NC, this.GetSymbol(tmpreg), info)
				continue
			} else {
				fmt.Printf(" X%d\t0x%X%s\n", i, reg, this.GetSymbol(tmpreg))
				continue
			}
		}

		fmt.Printf(" X%d\t0x%X\n", i, reg)
	}
	_, err := this.ParseAddress(this.Context.LR)
	if err == nil {
		fmt.Printf("%s*LR%s\t%s0x%X%s%s\n", config.RED, config.NC, config.CYAN, this.Context.LR, config.NC, this.GetSymbol(this.Context.LR))
	} else {
		fmt.Printf(" LR\t0x%X\n", this.Context.LR)
	}
	fmt.Printf("%s*SP%s\t%s0x%X%s\n", config.RED, config.NC, config.CYAN, this.Context.SP, config.NC)

	_, err = this.ParseAddress(this.Context.PC)
	if err == nil {
		fmt.Printf("%s*PC%s\t%s0x%X%s%s\n", config.RED, config.NC, config.CYAN, this.Context.PC, config.NC, this.GetSymbol(this.Context.PC))
	} else {
		fmt.Printf(" PC\t0x%X\n", this.Context.PC)
	}
	// fmt.Println("─────────────────────────────────────────────────────────────────────────────────────────")
}

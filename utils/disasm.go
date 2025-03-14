package utils

import (
    "fmt"
    // "strings"
    "golang.org/x/arch/arm64/arm64asm"
	// "encoding/binary"
)

type IContext interface {
	GetReg(int) uint64
	GetPC() uint64
	GetLR() uint64
	GetSP() uint64
	GetPstate() uint64
}


func DisASM(code []byte) (string, error) {
	inst, err := arm64asm.Decode(code)
	if err != nil {
		return "", err
	}
	return inst.String(), nil
}
// func formatToGNUSyntax(inst arm64asm.Inst) string {
//     s := arm64asm.Plan9Syntax(inst)
//     s = strings.ToLower(s)
//     s = strings.Replace(s, "·", "", -1) // 移除 Plan9 特殊字符
//     return s
// }

func getCondition(inst arm64asm.Inst) string {
    if inst.Op != arm64asm.B {
        return "AL"
    }
    for _, arg := range inst.Args {
        if cond, ok := arg.(arm64asm.Cond); ok {
            return cond.String()
        }
    }
    return "AL"
}


func PredictNextPC(pid uint32, ctx IContext, Step bool) (uintptr, error) {
	PC := ctx.GetPC()
	pstate := ctx.GetPstate()
	asm := make([]byte, 4)
	n, err := ReadProcessMemory(pid, uintptr(PC), asm)
	if n < 4 || err != nil {
		return uintptr(PC + 4), fmt.Errorf("PredictNextPC: Failed to read instruction: %v", err)
	}

    // 1. 解码ARM64指令
    inst, err := arm64asm.Decode(asm)
    if err != nil {
        return uintptr(PC + 4), fmt.Errorf("PredictNextPC: Failed to disassemble instruction: %v", err)
    }

    // 2. 处理分支指令
    switch inst.Op {
	case arm64asm.RET:
		return uintptr(ctx.GetLR()), nil
    case arm64asm.B: // 无条件跳转
		cond := getCondition(inst)
		fmt.Println("Condition: ", cond)
		if conditionMet(cond, pstate) {
			if target := getBranchTarget(inst, PC); target != 0 {
				return uintptr(target), nil
			}
		} else {
			return uintptr(PC + 4), nil
		}
	case arm64asm.BL:
		if Step == true {
			if target := getBranchTarget(inst, PC); target != 0 {
				return uintptr(target), nil
			}
		} else {
			return uintptr(PC + 4), nil
		}
	case arm64asm.CBZ, arm64asm.CBNZ:
		//to do!
	case arm64asm.TBZ, arm64asm.TBNZ:
		//to do!
		
    }
    
    // 3. 默认返回顺序执行地址
    return uintptr(PC + 4), nil
}

// func extractCondFromOp(op arm64asm.Op) arm64asm.Cond {
//     switch op {
//     case arm64asm.BEQ: return arm64asm.CondEQ
//     case arm64asm.BNE: return arm64asm.CondNE
//     case arm64asm.BHS: return arm64asm.CondHS
//     case arm64asm.BLO: return arm64asm.CondLO
//     case arm64asm.BMI: return arm64asm.CondMI
//     case arm64asm.BPL: return arm64asm.CondPL
//     case arm64asm.BVS: return arm64asm.CondVS
//     case arm64asm.BVC: return arm64asm.CondVC
//     case arm64asm.BHI: return arm64asm.CondHI
//     case arm64asm.BLS: return arm64asm.CondLS
//     case arm64asm.BGE: return arm64asm.CondGE
//     case arm64asm.BLT: return arm64asm.CondLT
//     case arm64asm.BGT: return arm64asm.CondGT
//     case arm64asm.BLE: return arm64asm.CondLE
//     default: return arm64asm.CondNV
//     }
// }

// getBranchTarget 计算分支指令的目标地址
func getBranchTarget(inst arm64asm.Inst, PC uint64) uint64 {
    for _, arg := range inst.Args {
        if pcrel, ok := arg.(arm64asm.PCRel); ok {
            // ARM64分支指令偏移量基于PC+8计算
            return PC + 8 + uint64(int64(pcrel))
        }
    }
	fmt.Println("Failed to get branch target.")
    return 0
}

// conditionMet 判断条件码是否满足
func conditionMet(cond string, pstate uint64) bool {
    // 提取NZCV标志位
    n := (pstate >> 31) & 1
    z := (pstate >> 30) & 1
    c := (pstate >> 29) & 1
    v := (pstate >> 28) & 1

    switch cond {
	case "AL": return true
    case "EQ":  return z == 1
    case "NE":  return z == 0
    case "HS": return c == 1
    case "LO": return c == 0
    case "MI":  return n == 1
    case "PL":  return n == 0
    case "VS": return v == 1
    case "VC": return v == 0
    case "HI": return c == 1 && z == 0
    case "LS": return c == 0 || z == 1
    case "GE": return n == v
    case "LT": return n != v
    case "GT": return z == 0 && n == v
    case "LE": return z == 1 || n != v
    default: return false
    }
}
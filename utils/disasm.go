package utils

import (
    // "fmt"
    // "strings"
    "golang.org/x/arch/arm64/arm64asm"
)

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
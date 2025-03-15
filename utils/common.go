package utils

import (
	"math/rand"
	"os/exec"
	"strings"
	"time"
	"io/ioutil"
	"fmt"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}


func RunCommand(executable string, args ...string) (string, error) {
	cmd := exec.Command(executable, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}
	bytes, err := ioutil.ReadAll(stdout)
	if err != nil {
		return "", err
	}
	if err := cmd.Wait(); err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytes)), nil
}


func HexDump(address uint64, data []byte, len int) string {
    var builder strings.Builder
    for i := 0; i < len; i += 16 {
        // 计算当前行偏移量
        offset := i
        builder.WriteString(fmt.Sprintf("%08x  ", address+uint64(offset)))

        // 生成十六进制部分
        hexPart := make([]string, 16)
        asciiPart := make([]rune, 16)
        for j := 0; j < 16; j++ {
            if i+j < len {
                hexPart[j] = fmt.Sprintf("%02x", data[i+j])
                if data[i+j] >= 32 && data[i+j] <= 126 { // 可打印字符范围
                    asciiPart[j] = rune(data[i+j])
                } else {
                    asciiPart[j] = '.'
                }
            } else {
                hexPart[j] = "  " // 对齐空白
                asciiPart[j] = ' '
            }
        }

        // 拼接十六进制字符串（8 字节分组）
        hexLine := strings.Join(hexPart[:8], "") + "  " + strings.Join(hexPart[8:], "")
        builder.WriteString(fmt.Sprintf("%-47s  |%s|", hexLine, string(asciiPart)))
        builder.WriteString("\n")
    }
    return builder.String()
}
package utils

import (
	"math/rand"
	"syscall"
	"os/exec"
	"strings"
    "strconv"
	"time"
	"io/ioutil"
	"fmt"
	"compress/gzip"
	"os"
	"bytes"
	"encoding/hex"
	"bufio"
    "github.com/casbin/govaluate"
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

func CheckConfig(targetStr string) bool {
    file, err := os.Open("/proc/config.gz")
    if err != nil {
        return false
    }
    defer file.Close()

    gzReader, err := gzip.NewReader(file)
    if err != nil {
        return false
    }
    defer gzReader.Close()

    scanner := bufio.NewScanner(gzReader)
    target := []byte(targetStr)
    for scanner.Scan() {
        if bytes.Contains(scanner.Bytes(), target) {
			// fmt.Println(scanner.Text())
            return true
        }
    }
    return false
}

func FindBTFAssets() string {
    var utsname syscall.Utsname
    err := syscall.Uname(&utsname)
    if err != nil {
        fmt.Println("Error:", err)
        os.Exit(1)
    }
    btf_file := "a12-5.10-arm64_min.btf"
    if strings.Contains(B2S(utsname.Release[:]), "rockchip") {
        btf_file = "rock5b-5.10-arm64_min.btf"
    }
    fmt.Printf("Load btf_file=%s\n", btf_file)
    return btf_file
}

func B2S(bs []int8) string {
	ba := make([]byte, 0, len(bs))
	for _, b := range bs {
		ba = append(ba, byte(b))
	}
	return string(bytes.TrimSpace(bytes.Trim(ba, "\x00")))
}

func HexStringToBytes(hexStr string) ([]byte, error) {
    // 去除可能的空格或前缀（如 "0x"）
    cleaned := strings.ReplaceAll(hexStr, " ", "")
    cleaned = strings.TrimPrefix(cleaned, "0x")

    // 解码为字节数组
    data, err := hex.DecodeString(cleaned)
    if err != nil {
        return nil, fmt.Errorf("invalid hex string: %v", err)
    }
    return data, nil
}

func WriteBytesToFile(filename string, data []byte) error {
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    _, err = file.Write(data)
    if err != nil {
        return err 
    }

    return file.Sync()
}

func GetExprValue(input string, ctx IContext) (uint64, error) {
    expr, err := govaluate.NewEvaluableExpression(input)
    if err != nil {
        return 0, fmt.Errorf("Syntax Error: %v", err)
    }
    parameters := make(map[string]interface{})
    for i := 0; i < 32; i += 1 {
        parameters["w"+strconv.Itoa(i)] = ctx.GetReg(i)
        parameters["W"+strconv.Itoa(i)] = ctx.GetReg(i)
    }
    for i := 32; i < 64; i += 1 {
        parameters["x"+strconv.Itoa(i-32)] = ctx.GetReg(i)
        parameters["X"+strconv.Itoa(i-32)] = ctx.GetReg(i)
    }
    parameters["PC"] = ctx.GetPC()
    parameters["pc"] = ctx.GetPC()
    parameters["LR"] = ctx.GetLR()
    parameters["lr"] = ctx.GetLR()
    parameters["SP"] = ctx.GetSP()
    parameters["sp"] = ctx.GetSP()
    result, err := expr.Evaluate(parameters)
    if err != nil {
        return 0, fmt.Errorf("Evaluate Error: %v", err)
    }
    return uint64(result.(float64)), nil
}
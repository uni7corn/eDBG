package cli
import (
	"bufio"
	"errors"
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
	"eDBG/utils"
	"eDBG/module"
)

var Logger *log.Logger

func NewLogger() *log.Logger {
    if Logger != nil {
        return Logger
    }

    Logger = log.New(os.Stdout, "", 0)
    return Logger
}

var config := utils.NewGlobalConfig()

func main() {

	flag.Var(&config.initialOffsets, "brk", "Set initial breakpoint (hex or decimal)")
	flag.StringVar(&config.packageName, "pkg", "", "Target package name")
	flag.StringVar(&config.libName, "lib", "", "Target library name")
	flag.Parse()
	Logger := NewLogger()
	config.SetLogger(Logger)
	module.Init(config)

	log.Printf("[+] Attached to package %s", pkg)
}

func REPL() {
	scanner := bufio.NewScanner(os.Stdin)
	Logger.Print("(eDBG) ")
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			Logger.Print("(eDBG) ")
			continue
		}

		parts := strings.Fields(line)
		cmd := parts[0]
		args := parts[1:]

		switch cmd {
		case "break", "b":
			handleBreak(args)
		case "step", "s":
			handleStep()
		case "next", "n":
			handleNext()
		case "x":
			handleMemory(args)
		case "quit", "q":
			os.Exit(0)
		case "continue", "c":
			handleContinue()
		default:
			Logger.Println("Unknown command:", cmd)
		}

		Logger.Print("(eDBG) ")
	}
}

func handleBreak(args []string) {
	if len(args) == 0 {
		Logger.Println("Usage: break <offset>")
		return
	}

	offset, err := parseOffset(args[0])
	if err != nil {
		Logger.Printf("Bad offset: %v", err)
		return
	}

	if err := module.addBreakPoint(off); err != nil {
		Logger.Printf("Failed to set breakpoint: %v", err)
	} else {
		Logger.Printf("Breakpoint at 0x%x", offset)
	}
}

func handleContinue() {
	Logger.Print("todo")
}

func handleStep() {
	Logger.Print("todo")
}

func handleNext() {
	Logger.Print("todo")
}

func handleMemory(args []string) {
	Logger.Print("todo")
	// if len(args) < 2 {
	// 	log.Println("Usage: x <address> <length>")
	// 	return
	// }

	// address, err := parseOffset(args[0])
	// if err != nil {
	// 	log.Printf("Bad offset: %v", err)
	// 	return
	// }

	// length, err := strconv.Atoi(args[1])
	// if err != nil || length <= 0 {
	// 	log.Println("Bad length")
	// 	return
	// }

	// data, err := module.ReadMemory(address, length)
	// if err != nil {
	// 	log.Printf("Reading Memory Error: %v", err)
	// 	return
	// }

	// buf := &strings.Builder{}
	// for i := 0; i < len(data); i++ {
	// 	if i%16 == 0 {
	// 		if i != 0 {
	// 			buf.WriteByte('\n')
	// 		}
	// 		fmt.Fprintf(buf, "%08x  ", address+uint64(i))
	// 	}
	// 	fmt.Fprintf(buf, "%02x ", data[i])
	// }
	// log.Println(buf.String())
}

func parseOffset(s string) (uint64, error) {
	s = strings.TrimPrefix(s, "0x")
	return strconv.ParseUint(s, 16, 64)
}
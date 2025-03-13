package main

import (
    "eDBG/cli"
	"errors"
    // "bufio"
	"flag"
	// "log"
	"os"
	"strconv"
	"syscall"
    "fmt"
    "os/signal"
	"strings"
    "eDBG/module"
	// "eDBG/utils"
	"eDBG/controller"
    "eDBG/event"
    _ "github.com/shuLhan/go-bindata" // add for bindata in Makefile
)

// type Imodule interface {
// 	Start(*processController.Library, uint64[]) error
// }

// type Client

func main() {
	stopper := make(chan os.Signal, 1)
    signal.Notify(stopper, os.Interrupt, syscall.SIGTERM)

	var (
		packageName string
		libName string
	)
	var brkFlag string

	flag.StringVar(&brkFlag, "brk", "", "Breakpoint addresses in hex format, e.g., [0x1234,0x5678]")
	flag.StringVar(&packageName, "pkg", "", "Target package name")
	flag.StringVar(&libName, "lib", "", "Target library name")
	flag.Parse()

    if packageName == "" {
        fmt.Println("No Package Specified. Use --pkg com.package.name")
        os.Exit(1)
    }
    if libName == "" {
        fmt.Println("No Library Specified. Use --lib libraryname.so")
        os.Exit(1)
    }
    if brkFlag == "" {
        fmt.Println("Initial Breakpoint needed. Use --brk [0x1234,0x5678]")
        os.Exit(1)
    }

	process, err := controller.CreateProcess(packageName)
	if err != nil {
		fmt.Println("Create process error: ", err)
		os.Exit(1)
	}

	library, err := controller.CreateLibrary(process, libName)
	if err != nil {
		fmt.Println("Create Library error: ", err)
		os.Exit(1)
	}

    eventListener := event.CreateEventListener(process)
    brkManager := module.CreateBreakPointManager(eventListener)
    client := cli.CreateClient(process, library, brkManager)
    eventListener.SetupClient(client)

	brkAddrs, err := ParseBreakPoints(brkFlag)
	if err != nil {
		fmt.Println("Create Breakpoints Failed: ", err)
		os.Exit(1)
	}
	
	err = brkManager.Start(*library, brkAddrs)
	if err != nil {
		fmt.Println("Module start Failed: ", err)
		os.Exit(1)
	}
	fmt.Println("Module started. Press Ctrl+C to quit")

    // go func(){
    //     scanner := bufio.NewScanner(os.Stdin)
    //     for {
    //         scanner.Scan()
    //         err := scanner.Err()
    //         if err != nil {
    //             fmt.Printf("get input from console failed, err:%v", err)
    //             syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
    //         }
    //         input_text := scanner.Text()
    //         if input_text == "q" {
    //             syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
    //         }
    //     }
    // }()
    client.Run()
    <-stopper
}


func ParseBreakPoints(brkFlag string) ([]uint64, error) {
	trimmed := strings.Trim(brkFlag, "[]")
	if trimmed == "" {
		return nil, errors.New("ParseBreakPoints: Invalid breakpoint format. Usage: --brk [0x1234,0x5678]")
	}

	addresses := strings.Split(trimmed, ",")
	var brkAddrs []uint64
	for _, addrStr := range addresses {
		addrStr = strings.TrimSpace(addrStr)
		addr, err := strconv.ParseUint(addrStr, 0, 64)
		if err != nil {
			return nil, fmt.Errorf("ParseBreakPoints: Invalid address %q: %v", addrStr, err)
		}
		brkAddrs = append(brkAddrs, addr)
	}
	return brkAddrs, nil
}
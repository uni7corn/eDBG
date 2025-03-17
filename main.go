package main

import (
    "eDBG/cli"
	// "errors"
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
        hiddis bool
        hidreg bool
		threadFilters string
	)
	var brkFlag string

	flag.StringVar(&brkFlag, "b", "", "Breakpoint addresses in hex format, e.g., [0x1234,0x5678]")
	flag.StringVar(&packageName, "p", "", "Target package name")
	flag.StringVar(&libName, "l", "", "Target library name")
    flag.BoolVar(&hidreg, "hide-register", false, "Hide Register Window")
    flag.BoolVar(&hiddis, "hide-disassemble", false, "Hide Disassemble Window")
	flag.StringVar(&threadFilters, "t", "", "Thread name filters, e.g., [name1,name2]")
	flag.Parse()

    if packageName == "" {
        fmt.Println("No Package Specified. Use -p com.package.name")
        os.Exit(1)
    }
    if libName == "" {
        fmt.Println("No Library Specified. Use -l libraryname.so")
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
    client := cli.CreateClient(process, library, brkManager, &cli.UserConfig{
        Registers: !hidreg,
        Disasm: !hiddis,
    })
    
	brkAddrs, err := ParseBreakPoints(brkFlag)
	if err != nil {
		fmt.Println("Create Breakpoints Failed: ", err)
		os.Exit(1)
	}

	tNames, err := ParseThreadNames(threadFilters)
	if err != nil {
		fmt.Println("Create Thread names Failed: ", err)
		os.Exit(1)
	}

	for _, name := range tNames {
		client.AddThreadFilterName(name)
	}

	eventListener.SetupClient(client)
    
	err = brkManager.Start(library, brkAddrs)
	if err != nil {
		fmt.Println("Module start Failed: ", err)
		os.Exit(1)
	}
	fmt.Printf("Working on %s in %s. Press Ctrl+C to quit\n", libName, packageName)

    client.Run()
	eventListener.Run()
    <-stopper
    process.Continue()
}


func ParseBreakPoints(brkFlag string) ([]uint64, error) {
	trimmed := strings.Trim(brkFlag, "[]")
	if trimmed == "" {
		return []uint64{}, nil
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

func ParseThreadNames(brkFlag string) ([]string, error) {
	trimmed := strings.Trim(brkFlag, "[]")
	if trimmed == "" {
		return []string{}, nil
	}
	addresses := strings.Split(trimmed, ",")
	var brkAddrs []string
	for _, addrStr := range addresses {
		addrStr = strings.TrimSpace(addrStr)
		brkAddrs = append(brkAddrs, addrStr)
	}
	return brkAddrs, nil
}
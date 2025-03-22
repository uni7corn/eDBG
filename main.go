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
	"eDBG/utils"
	"eDBG/controller"
	"encoding/json"
    "eDBG/event"
    _ "github.com/shuLhan/go-bindata" // add for bindata in Makefile
)
type UserBreakPoints struct {
	LibName    string `json:"brklib"`
	Offset 	   uint64 `json:"offset"`
	Enable	   bool   `json:"enable"`
} 

var EdbgConfigVersion int

type AppConfig struct {
	ConfigVersion   int       `json:"ConfigVersion"`
    PackageName     string    `json:"packagename"`
    LibName 		string    `json:"libname"`
	Hidereg			bool 	  `json:"hidereg"`
	Hidedis			bool	  `json:"hidedis"`
	BreakPoints	[]UserBreakPoints `json:"brk"`
	TNames      []string      `json:"tname"`
}

func SaveConfig(path string, cfg AppConfig) error {
    jsonData, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(path, jsonData, 0644)
}

func LoadConfig(path string) (AppConfig, error) {
    var cfg AppConfig
    jsonData, err := os.ReadFile(path)
    if err != nil {
        return cfg, err
    }
    err = json.Unmarshal(jsonData, &cfg)
    return cfg, err
}

func main() {
	EdbgConfigVersion = 1
	stopper := make(chan os.Signal, 1)
    signal.Notify(stopper, os.Interrupt, syscall.SIGTERM)
	workedlib := make(map[string]*controller.LibraryInfo)
	var (
		inputfile string
		outputfile string
		packageName string
		libName string
        hiddis bool
        hidreg bool
		save  bool
		disableHW bool
		threadFilters string
		brkAddressInfos []*controller.Address
	)
	var brkFlag string

	flag.StringVar(&brkFlag, "b", "", "Breakpoint addresses in hex format, e.g., [0x1234,0x5678]")
	flag.StringVar(&packageName, "p", "", "Target package name")
	flag.StringVar(&libName, "l", "", "Target library name")
    flag.BoolVar(&hidreg, "hide-register", false, "Hide Register Window")
    flag.BoolVar(&hiddis, "hide-disassemble", false, "Hide Disassemble Window")
	flag.StringVar(&threadFilters, "t", "", "Thread name filters, e.g., [name1,name2]")
	flag.StringVar(&inputfile, "i", "", "Input file saved from edbg. e.g. sample.edbg")
	flag.BoolVar(&save, "s", false, "Save your progress to input file")
	flag.BoolVar(&disableHW, "disable-hw", false, "Disable hardware breakpoints")
	flag.StringVar(&outputfile, "o", "&&NotSetNotSetNotSetO=O", "Save your progress to specified file")
	flag.Parse()

	if inputfile != "" {
		cfg, err := LoadConfig(inputfile)
		if err != nil {
			fmt.Printf("Failed to load input file: %v\n", err)
			os.Exit(1)
		}

		if cfg.ConfigVersion != EdbgConfigVersion {
			fmt.Printf("Config version %d is not supported now.\n", cfg.ConfigVersion)
			os.Exit(1)
		}

		packageName = cfg.PackageName
		libName = cfg.LibName
		hidreg = cfg.Hidereg
		hiddis = cfg.Hidedis

		fmt.Printf("Using Config from: %s\n", inputfile)
	}
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
	workedlib[libName] = library
	btfFile := ""
	if !utils.CheckConfig("CONFIG_DEBUG_INFO_BTF=y") {
		btfFile = utils.FindBTFAssets()
	}
	if !utils.CheckConfig("CONFIG_HAVE_HW_BREAKPOINT=y") {
		fmt.Println("Hardware breakpoints not enabled. Using uprobes.")
		disableHW = true
	}
    eventListener := event.CreateEventListener(process)
    brkManager := module.CreateBreakPointManager(eventListener, btfFile, process, !disableHW)
    client := cli.CreateClient(process, library, brkManager, &cli.UserConfig{
        Registers: !hidreg,
        Disasm: !hiddis,
    })
    if inputfile == "" {
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

		for _, offset := range brkAddrs {
			brkAddressInfos = append(brkAddressInfos, controller.NewAddress(library, offset))
		}
	}  else {
		Config, _ := LoadConfig(inputfile)
		for _, name := range Config.TNames {
			client.AddThreadFilterName(name)
		}
		for _, brk := range Config.BreakPoints {
			val, ok := workedlib[brk.LibName]
			var libInfo *controller.LibraryInfo
			if !ok {
				_libinfo, err := controller.CreateLibrary(process, brk.LibName)
				if err != nil {
					fmt.Printf("Cannot locate library %s: %v. Skipped.\n", brk.LibName, err)
					continue
				}
				libInfo = _libinfo
				workedlib[brk.LibName] = libInfo
			} else {
				libInfo = val
			}

			if brk.Enable {
				brkAddressInfos = append(brkAddressInfos, controller.NewAddress(libInfo, brk.Offset))
			} else {
				brkManager.CreateBreakPoint(controller.NewAddress(libInfo, brk.Offset), false)
			}
		}
	}
	
	eventListener.SetupClient(client)
    
	err = brkManager.Start(brkAddressInfos)
	if err != nil {
		fmt.Println("Module start Failed: ", err)
		os.Exit(1)
	}
	fmt.Printf("Working on %s in %s. Press Ctrl+C to quit\n", libName, packageName)

    client.Run()
	eventListener.Run()
    <-stopper
    process.Continue()
	// _ = brkManager.Stop()
	if save {
		if inputfile == "" {
			fmt.Println("No input file!")
		} else {
			outputfile = inputfile
		}
	}
	if outputfile != "&&NotSetNotSetNotSetO=O" {
		cfg := AppConfig{
			ConfigVersion: EdbgConfigVersion,
			PackageName: packageName,
			LibName: libName,
			Hidereg: hidreg,
			Hidedis: hiddis,
		}

		for _, brk := range brkManager.BreakPoints {
			if brk.Deleted {
				continue
			}
			cfg.BreakPoints = append(cfg.BreakPoints, UserBreakPoints{
				LibName: brk.LibInfo.LibName,
				Offset:  brk.Offset,
				Enable:  brk.Enable,
			})
		}

		for _, t := range client.Config.ThreadFilters {
			if !t.Enable {
				continue
			}
			tName := t.Thread.Name
			if tName != "" {
				cfg.TNames = append(cfg.TNames, tName)
			}
		}
		SaveConfig(outputfile, cfg)
		fmt.Println("Progress saved to file: %s", outputfile)
	}
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
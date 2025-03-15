package cli

import (
	"eDBG/utils"
	"eDBG/controller"
	"eDBG/module"
	"os"
	"fmt"
	"strings"
	"strconv"
	"syscall"
	"bufio"
)

type DisplayInfo struct {
	Name string
	Address uint64
	Enable bool
	Len int
}

type UserConfig struct {
	Registers bool
	Disasm bool
	Display []*DisplayInfo
}

type Client struct {
	Library *controller.LibraryInfo
	Process *controller.Process
	BrkManager *module.BreakPointManager
	Config *UserConfig
	Incoming chan bool
	Done chan bool
	DoClean chan bool
}

func CreateClient(process *controller.Process, library *controller.LibraryInfo, brkManager *module.BreakPointManager, config *UserConfig) *Client {
	return &Client{library, process, brkManager, config, make(chan bool, 1), make(chan bool, 1), make(chan bool, 1)}
}

func (this *Client) Run() {
	go func() {
		for {
			<- this.DoClean
			this.StopProbes()
			this.Done <- true
		}
	}()
	go func() {
		for {
			<- this.Incoming
			// fmt.Println("Incoming!")
			this.OutputInfo()
			this.REPL()
		}
	}()
}

func (this *Client) OutputInfo() {
	// fmt.Println("?")
	if this.Config.Registers {
		fmt.Println("──────────────────────────────────────[ REGISTERS ]──────────────────────────────────────")
		this.Process.PrintContext()
	}
	if this.Config.Disasm {
		fmt.Println("──────────────────────────────────────[  DISASM  ]────────────────────────────────────────")
		this.PrintDisassembleInfo(this.Process.Context.PC, 10)
	}
	cntDisplay := 0
	for _, display := range this.Config.Display {
		if display.Enable {
			cntDisplay++
			break
		}
	}
	if cntDisplay > 0 {
		fmt.Println("──────────────────────────────────────[ DISPLAY ]────────────────────────────────────────")
		this.PrintDisplay()
	}
	if this.Config.Registers || this.Config.Disasm || cntDisplay > 0 {
		fmt.Println("─────────────────────────────────────────────────────────────────────────────────────────")
	}
}

func (this *Client) PrintDisplay() {
	for id, display := range this.Config.Display {
		if !display.Enable {
			continue
		}
		fmt.Printf("[%d] 0x%x:%s\n", id, display.Address, display.Name)
		data := make([]byte, display.Len)
		n, err := utils.ReadProcessMemory(this.Process.WorkPid, uintptr(display.Address), data)

		if err != nil {
			fmt.Printf("Reading Memory Error: %v\n", err)
		}

		fmt.Println(utils.HexDump(display.Address, data, n))
	}
}

func (this *Client) StopProbes() {
	// fmt.Println("Doing Cleaning")
	err := this.BrkManager.Stop()
	if err != nil {
		fmt.Println("Failed to terminate.")
		this.CleanUp()
	}
	// fmt.Println("Done Clean")
}

func (this *Client) REPL() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("(eDBG) ")
loop:
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			fmt.Print("(eDBG) ")
			continue
		}

		parts := strings.Fields(line)
		cmd := parts[0]
		args := parts[1:]

		switch cmd {
		case "break", "b":
			this.HandleBreak(args)
		case "step", "s":
			this.HandleStep()
			break loop
		case "next", "n":
			this.HandleNext()
			break loop
		case "examine", "x":
			this.HandleMemory(args)
		case "quit", "q":
			this.CleanUp()
			return
		case "continue", "c":
			this.HandleContinue()
			break loop
		case "display", "disp":
			this.HandleDisplay(args)
		case "undisplay", "undisp":
			this.HandleUndisplay(args)
		case "list", "l", "dis", "disassemble":
			this.HandleList(args)
		case "info", "i":
			this.HandleInfo(args)
		case "finish", "fi":
			this.HandleFinish()
			return
		case "return":
			fmt.Println("Command return is not supported because eDBG cannot perform modification. Use finish or fi.")
		case "backtrace", "bt":
			fmt.Println("Command backtrace is not supported yet.")
		case "jump", "j":
			fmt.Println("Command jump is not supported because eDBG cannot perform modification.")
		case "thread":
			fmt.Println("Command thread is not supported.")
		case "disable":
			this.HandleChangeBrk(args, false)
		case "enable":
			this.HandleChangeBrk(args, true)
		case "delete":
			this.HandleDelete(args)
		case "until", "u":
			this.HandleUntil(args)
			return
		case "watch":
			fmt.Println("Command watch is not supported yet.")
		case "run", "r":
			fmt.Println("eDBG do not execute programs. Please run it manually.")
		default:
			fmt.Println("Unknown command:", cmd)
		}

		fmt.Print("(eDBG) ")
	}
}

func (this *Client) HandleInfo(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: info break/b\n       info register/reg")
		return
	}
	switch args[0] {
		case "break", "b":
			this.BrkManager.PrintBreakPoints()
		case "reg", "register", "r":
			this.Process.PrintContext()
		default:
			fmt.Println("Usage: info break/b\n       info register/reg")
	}
}


func (this *Client) HandleChangeBrk(args []string, status bool) {
	if len(args) == 0 {
		fmt.Println("Usage: enable/disable <breakpoint id>. Use <info b> to browse breakpoint info\n")
		return
	}
	id, err := strconv.ParseInt(args[0], 0, 32)
	if err != nil {
		fmt.Printf("Bad breakpoint id: %v\n", err)
		return 
	}
	this.BrkManager.ChangeBreakPoint(int(id), status)
}


func (this *Client) HandleDelete(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: delete <breakpoint id>. Use <info b> to browse breakpoint info")
		return
	}
	id, err := strconv.ParseInt(args[0], 0, 32)
	if err != nil {
		fmt.Printf("Bad breakpoint id: %v\n", err)
		return 
	}
	this.BrkManager.DeleteBreakPoint(int(id))
}

func (this *Client) HandleFinish() {
	address, err := this.Process.ParseAddress(this.Process.Context.LR)
	if err != nil {
		fmt.Printf("Failed to parse LR: %v\n", err)
		return
	}
	// fmt.Printf("Next addr: %s+%x\n", address.LibInfo.LibName, address.Offset)
	this.BrkManager.SetTempBreak(address)
	this.HandleContinue()
}

func (this *Client) CleanUp() {
	this.Process.Continue()
	_ = this.BrkManager.Stop()
	syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
}

func (this *Client) HandleList(args []string) {
	if len(args) == 0 {
		this.PrintDisassembleInfo(this.Process.Context.PC, 10)
		return
	}
	if len(args) == 1 {
		address, err := strconv.ParseUint(args[0], 0, 64)
		if err != nil {
			fmt.Printf("Bad address: %v\nUsage: list\n       list <address>\n       list <address> <len>\n", err)
			return
		}
		this.PrintDisassembleInfo(address, 10)
		return
	}
	if len(args) >= 2 {
		address, err := strconv.ParseUint(args[0], 0, 64)
		if err != nil {
			fmt.Printf("Bad address: %v\nUsage: list\n       list <address>\n       list <address> <len>\n", err)
			return
		}
	len, err := strconv.ParseUint(args[1], 0, 32)
		if err != nil {
			fmt.Printf("Bad Length: %v\nUsage: list\n       list <address>\n       list <address> <len>\n", err)
			return
		}
		this.PrintDisassembleInfo(address, int(len))
		return
	}
}

func (this *Client) HandleDisplay(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: display <address>\n       display <address> <len>\n       display <address> <len> <name>\n")
		return
	}
	info := &DisplayInfo{"", 0, true, 16}
	address, err := strconv.ParseUint(args[0], 0, 64)
	if err != nil {
		fmt.Printf("Bad address: %v\nUsage: display <address>\n       display <address> <len>\n       display <address> <len> <name>\n", err)
		return
	}
	info.Address = address
	if len(args) > 1 {
		len, err := strconv.ParseUint(args[1], 0, 32)
		if err != nil {
			fmt.Printf("Bad Length: %v\nUsage: display <address>\n       display <address> <len>\n       display <address> <len> <name>\n", err)
			return
		}
		info.Len = int(len)
	}
	if len(args) > 2 {
		info.Name = args[2]
	}
	this.Config.Display = append(this.Config.Display, info)
}

func (this *Client) HandleUndisplay(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: undisplay <id>\n")
		return
	}
	id, err := strconv.ParseUint(args[0], 0, 64)
	if err != nil {
		fmt.Printf("Bad id: %v\n", err)
		return
	}
	if int(id) >= len(this.Config.Display) {
		fmt.Printf("Invalid id.\n")
		return
	}
	if !this.Config.Display[id].Enable {
		fmt.Printf("Invalid id.\n")
		return
	}
	this.Config.Display[id].Enable = false
}


func (this *Client) ParseUserAddress(arg string) (*controller.Address, error) {
	if strings.Contains(arg, "+") {
		lastIndex := strings.LastIndex(arg, "+")
		libName := arg[:lastIndex]
		offset_str := arg[lastIndex+1:]
		offset, err := strconv.ParseUint(offset_str, 0, 64)
		if err != nil {
			return &controller.Address{}, fmt.Errorf("Bad address: %v", err)
		}
		if libName == "$" {
			address, err := this.Process.ParseAddress(this.Process.Context.PC+offset*4)
			if err != nil {
				return &controller.Address{}, fmt.Errorf("Bad address: %v", err)
			}
			return address, nil
		} else {
			libInfo, err := controller.CreateLibrary(this.Process, libName)
			if err != nil {
				return &controller.Address{}, err
			}
			return controller.NewAddress(libInfo, offset), nil
		}
	} else {
		offset, err := strconv.ParseUint(arg, 0, 64)
		// fmt.Printf("Try to set breakpoint at 0x%x\n", offset)
		if err != nil {
			return &controller.Address{}, fmt.Errorf("Bad address: %v", err)
		}
		address := controller.NewAddress(this.Library, offset)
		if offset > 0x5000000000 {
			address, err = this.Process.ParseAddress(offset)
			if err != nil {
				return &controller.Address{}, fmt.Errorf("Bad address: %v", err)
			}
		}
		return address, nil
	}
}

func (this *Client) HandleUntil(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: until <address>")
		return
	}
	address, err := this.ParseUserAddress(args[0])
	if err != nil {
		fmt.Printf("Failed to parse address: %v\n", err)
		return
	}
	
	if err = this.BrkManager.SetTempBreak(address); err != nil {
		fmt.Printf("Failed to set Temporary breakpoint: %v\n", err)
	} else {
		// fmt.Printf("Breakpoint at 0x%x\n", offset)
		this.HandleContinue()
	}
	// fmt.Printf("Breakpoint at 0x%x!!", offset)
}


func (this *Client) HandleBreak(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: break <address>")
		return
	}
	address, err := this.ParseUserAddress(args[0])
	if err != nil {
		fmt.Printf("Failed to parse address: %v\n", err)
		return
	}
	
	if err = this.BrkManager.CreateBreakPoint(address); err != nil {
		fmt.Printf("Failed to set breakpoint: %v\n", err)
	} else {
		fmt.Printf("Breakpoint at %s+%x\n", address.LibInfo.LibName, address.Offset)
	}
	// fmt.Printf("Breakpoint at 0x%x!!", offset)
}

func (this *Client) HandleContinue() {
	err := this.BrkManager.SetupProbe()
	if err != nil {
		fmt.Println("Failed to Continue.")
		this.CleanUp()
		return
	}
	this.Process.Continue()
}

func (this *Client) HandleStep() {
	NextPC, err := utils.PredictNextPC(this.Process.WorkPid, this.Process.Context, true)
	if err != nil {
		fmt.Printf("Failed to predict next addr: %v\n", err)
		return
	}
	// fmt.Printf("Next addr: %x\n", NextPC)
	address, err := this.Process.ParseAddress(uint64(NextPC))
	if err != nil {
		fmt.Printf("Failed to parse nextPC: %v\n", err)
		return
	}
	// fmt.Printf("Next addr: %s+%x\n", address.LibInfo.LibName, address.Offset)
	this.BrkManager.SetTempBreak(address)
	this.HandleContinue()
}

func (this *Client) HandleNext() {
	NextPC, err := utils.PredictNextPC(this.Process.WorkPid, this.Process.Context, false)
	if err != nil {
		fmt.Printf("Failed to predict next addr: %v\n", err)
		return
	}
	// fmt.Printf("Next addr: %x\n", NextPC)
	address, err := this.Process.ParseAddress(uint64(NextPC))
	if err != nil {
		fmt.Printf("Failed to parse nextPC: %v\n", err)
		return
	}
	// fmt.Printf("Next addr: %s+%x\n", address.LibInfo.LibName, address.Offset)
	this.BrkManager.SetTempBreak(address)
	this.HandleContinue()
}

func (this *Client) HandleMemory(args []string) {
	// fmt.Print("todo")
	if len(args) == 0 {
		fmt.Println("Usage: x <address> <length>")
		return
	}
	address, err := strconv.ParseUint(args[0], 0, 64)
	if err != nil {
		if strings.HasPrefix(args[0], "X") {
			regnum, err := strconv.ParseUint(args[0][1:], 0, 64)
			if err != nil {
				fmt.Printf("Bad Regnum: %v\n", err)
				return
			}
			if regnum > 31 {
				fmt.Printf("No such register\n")
				return
			}
			address = this.Process.Context.GetReg(int(regnum)+32)

		} else if args[0] == "SP" {
			address = this.Process.Context.SP
		} else {
			fmt.Printf("Bad address: %v\n", err)
			return
		}
	}
	
	length := 16
	if len(args) > 1 {
		len, err := strconv.Atoi(args[1])
		if err != nil || len <= 0 {
			fmt.Println("Bad length")
			return
		}
		length = len
	}

	data := make([]byte, length)
	n, err := utils.ReadProcessMemory(this.Process.WorkPid, uintptr(address), data)

	if err != nil {
		fmt.Printf("Reading Memory Error: %v\n", err)
		return
	}

	fmt.Println(utils.HexDump(address, data, n))

}

func (this *Client) PrintDisassembleInfo(address uint64, len int) {
	// len为指令条数
	codeBuf := make([]byte, len*4)
	n, err := utils.ReadProcessMemory(this.Process.WorkPid, uintptr(address), codeBuf)
	if err != nil {
		fmt.Println("Failed to read code...")
	} else {
		addInfo, err := this.Process.ParseAddress(address)
		if err == nil {
			fmt.Printf(">>  0x%x<%s+%x>\t", address, addInfo.LibInfo.LibName, addInfo.Offset)
		} else {
			fmt.Printf(">>  0x%x%v\t", address, err)
		}

		code, err := utils.DisASM(codeBuf[0:4])
		if err == nil {
			fmt.Printf("%s\n", code)
		} else {
			fmt.Println("(disassemble failed)")
		}
		for i := 4; i < n; i += 4{
			addInfo, err = this.Process.ParseAddress(address+uint64(i))
			if err == nil {
				fmt.Printf("    0x%x<%s+%x>\t", address+uint64(i), addInfo.LibInfo.LibName, addInfo.Offset)
			} else {
				fmt.Printf("    0x%x\t", address+uint64(i))
			}

			code, err = utils.DisASM(codeBuf[i:i+4])
			if err == nil {
				fmt.Printf("%s\n", code)
			} else {
				fmt.Println("(disassemble failed)\n")
			}
		}
	}
}



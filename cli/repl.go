package cli

import (
	"eDBG/utils"
	"eDBG/controller"
	"eDBG/config"
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

type ThreadFilter struct {
	Thread *controller.Thread
	Enable bool
}

type UserConfig struct {
	Registers bool
	Disasm bool
	ThreadFilters []*ThreadFilter
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
	NotifyContinue chan bool
	PreviousCMD string
	Working bool
}

func CreateClient(process *controller.Process, library *controller.LibraryInfo, brkManager *module.BreakPointManager, config *UserConfig) *Client {
	return &Client{
		Library: library, 
		Process: process, 
		BrkManager: brkManager, 
		Config: config, 
		Incoming: make(chan bool, 1), 
		Done: make(chan bool, 1), 
		DoClean: make(chan bool, 1), 
		NotifyContinue: make(chan bool, 1), 
		PreviousCMD: "",
	}
}

func (this *Client) Run() {
	go func() {
		for {
			<- this.DoClean
			this.StopProbes()
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
		fmt.Print(config.BLUE)
		fmt.Println("──────────────────────────────────────[ REGISTERS ]──────────────────────────────────────")
		fmt.Print(config.NC)
		this.Process.PrintContext()
	}
	if this.Config.Disasm {
		fmt.Print(config.BLUE)
		fmt.Println("──────────────────────────────────────[  DISASM  ]────────────────────────────────────────")
		fmt.Print(config.NC)
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
		fmt.Print(config.BLUE)
		fmt.Println("──────────────────────────────────────[ DISPLAY ]────────────────────────────────────────")
		fmt.Print(config.NC)
		this.PrintDisplay()
	}
	if this.Config.Registers || this.Config.Disasm || cntDisplay > 0 {
		fmt.Print(config.BLUE)
		fmt.Println("─────────────────────────────────────────────────────────────────────────────────────────")
		fmt.Print(config.NC)
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
	err := this.BrkManager.Stop()
	if err != nil {
		fmt.Println("WARN: Failed to terminate. A Breakpoint maybe triggered due to a race condition.", err)
		this.CleanUp()
	} else {	
		this.Done <- true
	}
}

func (this *Client) REPL() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("(eDBG) ")
loop:
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			line = this.PreviousCMD
		} else {
			this.PreviousCMD = line
		}

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
		case "hbreak", "hb":
			this.HandleHBreak(args, config.HW_BREAKPOINT_X)
		case "rwatch":
			this.HandleHBreak(args, config.HW_BREAKPOINT_R)
		case "watch":
			this.HandleHBreak(args, config.HW_BREAKPOINT_W)
		case "step", "s":
			this.HandleStep()
			break loop
		case "next", "n":
			this.HandleNext()
			break loop
		case "examine", "x":
			this.HandleMemory(args)
		case "dump":
			this.HandleDump(args)
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
			fmt.Println("Command return is not supported because eDBG cannot perform modification. Use finish or fi instead.")
		case "backtrace", "bt":
			fmt.Println("Command backtrace is coming soon...")
		case "jump", "j":
			fmt.Println("Command jump is not supported because eDBG cannot perform modification.")
		case "thread", "t":
			this.HandleThread(args)
		case "disable":
			this.HandleChangeBrk(args, false)
		case "enable":
			this.HandleChangeBrk(args, true)
		case "delete":
			this.HandleDelete(args)
		case "until", "u":
			this.HandleUntil(args)
			return
		case "run", "r":
			fmt.Println("eDBG DO NOT execute programs. Please run it manually.")
		case "set":
			this.HandleSet(args)
		case "write", "w":
			this.HandleWrite(args)
		default:
			fmt.Println("Unknown command:", cmd)
		}

		fmt.Print("(eDBG) ")
	}
}

func (this *Client) HandleSet(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: set <address> <name>")
		return
	}
	address, err := this.ParseUserAddressToAbsolute(args[0])
	if err != nil {
		fmt.Printf("Failed to parse address: %v\n", err)
		return
	}
	this.Process.Symbols[address] = args[1]
}


func (this *Client) AddThreadFilter(args string) {
	id, err := strconv.ParseInt(args, 0, 32)
	if err != nil {
		fmt.Printf("Bad id: %v\n", err)
		return 
	}
	tlist, err := this.Process.GetCurrentThreads()
	if err != nil {
		fmt.Printf("Failed to get threads: %v\n", err)
		return 
	}
	if int(id) >= len(tlist) {
		fmt.Printf("Bad id.\n")
		return 
	}
	this.Config.ThreadFilters = append(this.Config.ThreadFilters, &ThreadFilter{
		Thread: tlist[id],
		Enable: true,
	})

}
func (this *Client) AddThreadFilterName(args string) {
	this.Config.ThreadFilters = append(this.Config.ThreadFilters, &ThreadFilter{
		Thread: &controller.Thread{
			Tid: 0,
			Name: args,
		},
		Enable: true,
	})
}

func (this *Client) DeleteThreadFilter(args string) {
	id, err := strconv.ParseInt(args, 0, 32)
	if err != nil {
		fmt.Printf("Bad id: %v\n", err)
		return 
	}
	if int(id) >= len(this.Config.ThreadFilters) {
		fmt.Printf("Bad id.\n")
		return 
	}
	if this.Config.ThreadFilters[id].Enable == false {
		fmt.Printf("Bad id.\n")
		return 
	}
	this.Config.ThreadFilters[id].Enable = false
}

func (this *Client) HandleThread(args []string) {
	if len(args) == 0 {
		this.Process.PrintThreads()
		return
	}
	if len(args) >= 2 {
		switch args[0] {
		case "add", "+":
			this.AddThreadFilter(args[1])
		case "name", "+n":
			this.AddThreadFilterName(args[1])
		case "del", "-", "delete":
			this.DeleteThreadFilter(args[1])
		case "all":
			this.Config.ThreadFilters = []*ThreadFilter{}
		default:
			fmt.Println("Usage: thread add id\n      thread del id\nUse info t to browse threads.")
		}
		return
	}
	fmt.Println("Usage: thread add id\n      thread del id\nUse info t to browse threads.")
}

func (this *Client) PrintThreadFilters() {
	for id, t := range this.Config.ThreadFilters {
		if !t.Enable {
			continue
		}
		if t.Thread.Tid != 0 {
			fmt.Printf("[%d] ThreadID: %d\n", id, t.Thread.Tid)
			continue
		}
		if t.Thread.Name != "" {
			fmt.Printf("[%d] ThreadName: %s\n", id, t.Thread.Name)
			continue
		}
	}
}

func (this *Client) HandleInfo(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: info break/b\n       info register/reg/r\n       info thread/t")
		return
	}
	switch args[0] {
		case "break", "b":
			this.BrkManager.PrintBreakPoints()
		case "reg", "register", "r":
			this.Process.PrintContext()
		case "thread", "t":
			fmt.Println("Available threads:")
			this.Process.PrintThreads()
			fmt.Println("Thread filters:")
			this.PrintThreadFilters()
		default:
			fmt.Println("Usage: info break/b\n       info register/reg/r\n       info threads/t")
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
	this.BrkManager.SetTempBreak(address, this.Process.WorkTid)
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

func (this *Client) ParseUserAddressToAbsolute(arg string) (uint64, error) {
	if strings.Contains(arg, "+") {
		lastIndex := strings.LastIndex(arg, "+")
		libName := arg[:lastIndex]
		offset_str := arg[lastIndex+1:]
		offset, err := strconv.ParseUint(offset_str, 0, 64)
		if err != nil {
			return 0, fmt.Errorf("Bad address: %v", err)
		}
		if libName == "$" {
			return this.Process.Context.PC+offset*4, nil
		} else {
			libInfo, err := controller.CreateLibrary(this.Process, libName)
			if err != nil {
				return 0, err
			}
			address := controller.NewAddress(libInfo, offset)
			return this.Process.GetAbsoluteAddress(address)
		}
	} else {
		offset, err := strconv.ParseUint(arg, 0, 64)
		// fmt.Printf("Try to set breakpoint at 0x%x\n", offset)
		if err != nil {
			return 0, fmt.Errorf("Bad address: %v", err)
		}
		if offset > 0x5000000000 {
			return offset, nil
		}
		address := controller.NewAddress(this.Library, offset)
		return this.Process.GetAbsoluteAddress(address)
	}
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
	if address.Absolute == 0 {
		fmt.Printf("Relative address is not support for until\n")
		return
	}
	
	if err = this.BrkManager.SetTempBreak(address, this.Process.WorkTid); err != nil {
		fmt.Printf("Failed to set Temporary breakpoint: %v\n", err)
	} else {
		this.HandleContinue()
	}
}

func (this *Client) HandleHBreak(args []string, Type int) {
	if len(args) == 0 {
		fmt.Println("Usage: hbreak <address>")
		return
	}
	address, err := this.ParseUserAddress(args[0])
	if err != nil {
		fmt.Printf("Failed to parse address: %v\n", err)
		return
	}
	if address.Absolute == 0 {
		absolute, err := this.Process.GetAbsoluteAddress(address)
		if err != nil {
			fmt.Printf("Failed to get absolute address.")
			return
		}
		address.Absolute = absolute
	}
	if err = this.BrkManager.CreateHWBreakPoint(address, true, Type); err != nil {
		fmt.Printf("Failed to set breakpoint: %v\n", err)
	} else {
		fmt.Printf("Breakpoint at %x\n", address.Absolute)
	}
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
	
	if err = this.BrkManager.CreateBreakPoint(address, true); err != nil {
		fmt.Printf("Failed to set breakpoint: %v\n", err)
	} else {
		fmt.Printf("Breakpoint at %s+%x\n", address.LibInfo.LibName, address.Offset)
	}
}

func (this *Client) HandleContinue() {
	err := this.BrkManager.SetupProbe()
	if err != nil {
		fmt.Printf("Failed to Continue: %v\n", err)
		this.CleanUp()
		return
	}
	this.NotifyContinue <- true
	this.Working = false
}

func (this *Client) HandleStep() {
	NextPC, err := utils.PredictNextPC(this.Process.WorkPid, this.Process.Context, true)
	if NextPC == 0xDEADBEEF {
		target, err := utils.GetTarget(this.Process.WorkPid, this.Process.Context)
		if err != nil {
			fmt.Printf("Failed to get branch target: %v\n", err)
			return
		}
		address, err := this.Process.ParseAddress(uint64(this.Process.Context.GetPC()+4))
		if err != nil {
			fmt.Printf("Failed to parse nextPC: %v\n", err)
			return
		}
		this.BrkManager.SetTempBreak(address, this.Process.WorkTid)
		address2, err := this.Process.ParseAddress(uint64(target))
		if err != nil {
			fmt.Printf("Failed to parse nextPC: %v\n", err)
			return
		}
		this.BrkManager.SetTempBreak(address2, this.Process.WorkTid)
		this.HandleContinue()
		return
	}
	if err != nil {
		fmt.Printf("Failed to predict next addr: %v\n", err)
		return
	}
	address, err := this.Process.ParseAddress(uint64(NextPC))
	if err != nil {
		fmt.Printf("Failed to parse nextPC: %v\n", err)
		return
	}
	this.BrkManager.SetTempBreak(address, this.Process.WorkTid)
	this.HandleContinue()
}

func (this *Client) HandleNext() {
	NextPC, err := utils.PredictNextPC(this.Process.WorkPid, this.Process.Context, false)
	if NextPC == 0xDEADBEEF {
		target, err := utils.GetTarget(this.Process.WorkPid, this.Process.Context)
		if err != nil {
			fmt.Printf("Failed to get branch target: %v\n", err)
			return
		}
		address, err := this.Process.ParseAddress(uint64(this.Process.Context.GetPC()+4))
		if err != nil {
			fmt.Printf("Failed to parse nextPC: %v\n", err)
			return
		}
		this.BrkManager.SetTempBreak(address, this.Process.WorkTid)
		address2, err := this.Process.ParseAddress(uint64(target))
		if err != nil {
			fmt.Printf("Failed to parse nextPC: %v\n", err)
			return
		}
		this.BrkManager.SetTempBreak(address2, this.Process.WorkTid)
		this.HandleContinue()
		return
	}
	if err != nil {
		fmt.Printf("Failed to predict next addr: %v\n", err)
		return
	}
	address, err := this.Process.ParseAddress(uint64(NextPC))
	if err != nil {
		fmt.Printf("Failed to parse nextPC: %v\n", err)
		return
	}
	this.BrkManager.SetTempBreak(address, this.Process.WorkTid)
	this.HandleContinue()
}

func (this *Client) HandleWrite(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: write <address> <hexstring>")
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
	
	data, err := utils.HexStringToBytes(args[1])
	if err != nil {
		fmt.Printf("Failed to parse hexstring %s: %v\n", args[1], err)
		return
	}

	n, err := utils.WriteProcessMemory(this.Process.WorkPid, uintptr(address), data)

	if err != nil {
		fmt.Printf("Writing Memory Error: %v\n", err)
		return
	}
	fmt.Printf("%d bytes written.\n", n)
	fmt.Println(utils.HexDump(address, data, n))
}

func (this *Client) HandleDump(args []string) {
	if len(args) < 3 {
		fmt.Println("Usage: dump <address> <length> <filename>")
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

	// fmt.Println(utils.HexDump(address, data, n))
	err = utils.WriteBytesToFile(args[2], data)
	if err != nil {
		fmt.Printf("Failed to write to file: %v\n", err)
		return
	}
	fmt.Printf("Saved %d bytes to %s\n", n, args[2])
}


func (this *Client) HandleMemory(args []string) {
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
			fmt.Printf("%s>>  0x%x<%s+%x>%s\t", config.GREEN, address, addInfo.LibInfo.LibName, addInfo.Offset, config.GREEN)
		} else {
			fmt.Printf("%s>>  0x%x\t%s", config.GREEN, address, config.GREEN)
		}

		code, err := utils.DisASM(codeBuf[0:4], address, this.Process)
		if err == nil {
			index := strings.Index(code, " ")
			fmt.Printf("%s%s%s ", config.YELLOW, code[:index], config.NC)
			fmt.Printf("%s%s%s\n", config.CYAN, code[index+1:], config.NC)
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

			code, err = utils.DisASM(codeBuf[i:i+4], address+uint64(i), this.Process)
			if err == nil {
				index := strings.Index(code, " ")
				fmt.Printf("%s%s%s ", config.YELLOW, code[:index], config.NC)
				fmt.Printf("%s%s%s\n", config.CYAN, code[index+1:], config.NC)
			} else {
				fmt.Println("(disassemble failed)\n")
			}
		}
	}
}



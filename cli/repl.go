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

type Client struct {
	Pid uint32
	Library *controller.LibraryInfo
	Process *controller.Process
	BrkManager *module.BreakPointManager
	Incoming chan bool
	Done chan bool
	DoClean chan bool
}

func CreateClient(process *controller.Process, library *controller.LibraryInfo, brkManager *module.BreakPointManager) *Client {
	return &Client{0, library, process, brkManager, make(chan bool, 1), make(chan bool, 1), make(chan bool, 1)}
}

func (this *Client) Run() {
	go func() {
		for {
			<- this.DoClean
			// fmt.Println("Doing Cleaning")
			err := this.BrkManager.Stop()
			if err != nil {
				fmt.Println("Failed to terminate.")
				this.CleanUp()
				return
			}
			this.Done <- true
			<- this.Incoming
			// fmt.Println("Cli Ready")
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
				case "next", "n":
					this.HandleNext()
				case "x":
					this.HandleMemory(args)
				case "quit", "q":
					this.CleanUp()
					return
				case "continue", "c":
					this.HandleContinue()
					break loop
				default:
					fmt.Println("Unknown command:", cmd)
				}

				fmt.Print("(eDBG) ")
			}
		}
	}()
}

func (this *Client) CleanUp() {
	this.Process.Continue()
	_ = this.BrkManager.Stop()
	syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
}


func (this *Client) HandleBreak(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: break <offset>")
		return
	}

	offset, err := strconv.ParseUint(args[0], 0, 64)
	// fmt.Printf("Try to set breakpoint at 0x%x\n", offset)
	if err != nil {
		fmt.Printf("Bad offset: %v", err)
		return
	}

	if err := this.BrkManager.CreateBreakPoint(*this.Library, offset); err != nil {
		fmt.Printf("Failed to set breakpoint: %v", err)
	} else {
		fmt.Printf("Breakpoint at 0x%x\n", offset)
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
	fmt.Print("todo")
}

func (this *Client) HandleNext() {
	fmt.Print("todo")
}

func (	this *Client) HandleMemory(args []string) {
	// fmt.Print("todo")
	if len(args) < 2 {
		fmt.Println("Usage: x <address> <length>")
		return
	}

	address, err := strconv.ParseUint(args[0], 0, 64)
	if err != nil {
		fmt.Printf("Bad offset: %v", err)
		return
	}

	length, err := strconv.Atoi(args[1])
	if err != nil || length <= 0 {
		fmt.Println("Bad length")
		return
	}

	data := make([]byte, length)
	n, err := utils.ReadProcessMemory(this.Pid, uintptr(address), data)

	if err != nil {
		fmt.Printf("Reading Memory Error: %v", err)
		return
	}

	buf := &strings.Builder{}
	for i := 0; i < n; i++ {
		if i%16 == 0 {
			if i != 0 {
				buf.WriteByte('\n')
			}
			fmt.Fprintf(buf, "%08x\t", address+uint64(i))
		}
		if i%8 == 0 && i%16 != 0 {
			if i != 0 {
				buf.WriteByte(' ')
			}
		}
		fmt.Fprintf(buf, "%02x", data[i])
	}
	fmt.Println(buf.String())
}

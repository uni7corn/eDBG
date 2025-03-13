package cli

import (
	// "eDBG/utils"
	"eDBG/controller"
	"eDBG/module"
	"os"
	"fmt"
	"strings"
	"strconv"
	"bufio"
)

type Client struct {
	Library *controller.LibraryInfo
	Process *controller.Process
	BrkManager *module.BreakPointManager
}

func CreateClient(process *controller.Process, library *controller.LibraryInfo, brkManager *module.BreakPointManager) *Client {
	return &Client{library, process, brkManager}
}

func (this *Client) Run() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("(eDBG) ")
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
			os.Exit(0)
		case "continue", "c":
			this.HandleContinue()
		default:
			fmt.Println("Unknown command:", cmd)
		}

		fmt.Print("(eDBG) ")
	}
}

func (this *Client) CleanUp() {
	this.BrkManager.Stop()
}


func (this *Client) HandleBreak(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: break <offset>")
		return
	}

	offset, err := strconv.ParseUint(args[0], 0, 64)
	if err != nil {
		fmt.Printf("Bad offset: %v", err)
		return
	}

	if err := this.BrkManager.AddBreakPoint(*this.Library, offset); err != nil {
		fmt.Printf("Failed to set breakpoint: %v", err)
	} else {
		fmt.Printf("Breakpoint at 0x%x", offset)
	}
}

func (this *Client) HandleContinue() {
	this.Process.Continue()
}

func (this *Client) HandleStep() {
	fmt.Print("todo")
}

func (this *Client) HandleNext() {
	fmt.Print("todo")
}

func (	this *Client) HandleMemory(args []string) {
	fmt.Print("todo")
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

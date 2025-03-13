package event

import(
	manager "github.com/gojue/ebpfmanager"
	"encoding/binary"
	"unsafe"
	"eDBG/controller"
	"eDBG/utils"
	"eDBG/cli"
	"fmt"
)

type EventContext struct {
	Regs []uint64
	LR uint64
	PC uint64
	SP uint64
}

type EventListener struct {
	pid uint32
	client *cli.Client
	process *controller.Process
	ByteOrder binary.ByteOrder
	Incomingdata chan []byte
}

// type Event struct {
// 	pid Uint32
// 	ctx *EventContext
// }

func CreateEventListener(process *controller.Process) *EventListener {
	return &EventListener{process: process, ByteOrder: getHostByteOrder(), Incomingdata: make(chan []byte, 512)}
}

func (this *EventListener) SetupClient(client *cli.Client) {
	this.client = client
}

func getHostByteOrder() binary.ByteOrder {
	var i int32 = 0x01020304
	u := unsafe.Pointer(&i)
	pb := (*byte)(u)
	b := *pb
	if b == 0x04 {
		return binary.LittleEndian
	}
	return binary.BigEndian
}

func (this *EventListener) Workdata(data []byte) {
	<- this.client.Done
	// fmt.Println("Working data")
	bo := this.ByteOrder
	context := &EventContext{}
	for i := 12; i < 12+8*30; i += 8 {
		context.Regs = append(context.Regs, bo.Uint64(data[i:i+8]))
	}
	context.LR = bo.Uint64(data[12+8*30:12+8*31])
	context.SP = bo.Uint64(data[12+8*31:12+8*32])
	context.PC = bo.Uint64(data[12+8*32:12+8*33])

	fmt.Println("──────────────────────────────────────[  DISASM  ]────────────────────────────────────────")
	codeBuf := make([]byte, 40)
	n, err := utils.ReadProcessMemory(this.pid, uintptr(context.PC), codeBuf)
	// fmt.Println(codeBuf)
	if err != nil {
		fmt.Println("Failed to read code...")
	} else {
		code, err := utils.DisASM(codeBuf[0:4])
		if err == nil {
			fmt.Printf(">>  0x%x\t%s\n", context.PC, code)
		} else {
			fmt.Printf(">>  0x%x\t(disassemble failed)\n", context.PC)
		}
		for i := 4; i < n; i += 4{
			code, err = utils.DisASM(codeBuf[i:i+4])
			if err == nil {
				fmt.Printf("    0x%x\t%s\n", context.PC+uint64(i), code)
			} else {
				fmt.Printf("    0x%x\t(disassemble failed)\n", context.PC+uint64(i))
			}
		}
	}

	context.Print()
	this.client.Incoming <- true
}

func (this *EventListener) Run() {
	go func() {
		for {
			data := <- this.Incomingdata
			// fmt.Println("Recieved data")
			this.Workdata(data)
		}
	}()
}

func (this *EventListener) OnEvent(cpu int, data []byte, perfmap *manager.PerfMap, manager *manager.Manager) {
	bo := this.ByteOrder
	this.pid = bo.Uint32(data[4:8])
	fmt.Printf("Suspended on pid: %d\n", this.pid)
	this.process.StoppedPID(this.pid)
	this.client.Pid = this.pid
	this.Incomingdata <- data
	// fmt.Println("Send data")
	this.client.DoClean <- true
	// fmt.Println("Send DoClean")
}

func (this *EventContext) Print() {
	fmt.Println("──────────────────────────────────────[ REGISTERS ]──────────────────────────────────────")
	for i, reg := range(this.Regs) {
		if (i+1)%3 != 0 {
			fmt.Printf("X%d\t%16X\t", i, reg)
		} else {
			fmt.Printf("X%d\t%16X\n", i, reg)
		}
	}
	fmt.Printf("LR\t%16X\t", this.LR)
	fmt.Printf("SP\t%16X\t", this.SP)
	fmt.Printf("PC\t%16X\n", this.PC)
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────────────")
}
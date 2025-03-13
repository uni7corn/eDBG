package event

import(
	manager "github.com/gojue/ebpfmanager"
	"encoding/binary"
	"unsafe"
	"eDBG/controller"
	// "eDBG/utils"
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
	client *cli.Client
	process *controller.Process
	ByteOrder binary.ByteOrder
}

// type Event struct {
// 	pid Uint32
// 	ctx *EventContext
// }

func CreateEventListener(process *controller.Process) *EventListener {
	return &EventListener{process: process, ByteOrder: getHostByteOrder()}
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



func (this *EventListener) OnEvent(cpu int, data []byte, perfmap *manager.PerfMap, manager *manager.Manager) {
	bo := this.ByteOrder
	pid := bo.Uint32(data[4:8])
	fmt.Printf("Suspended on pid: %d\n", pid)
	this.process.StoppedPID(pid)
	context := &EventContext{}
	for i := 12; i < 12+8*30; i += 8 {
		context.Regs = append(context.Regs, bo.Uint64(data[i:i+8]))
	}
	context.LR = bo.Uint64(data[12+8*30:12+8*31])
	context.SP = bo.Uint64(data[12+8*31:12+8*32])
	context.PC = bo.Uint64(data[12+8*32:12+8*33])
	context.Print()
	this.client.Incoming <- true
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
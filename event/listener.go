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
	regs []uint64
	pc uint64
	sp uint64
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
	pid := bo.Uint32(data[0:4])
	this.process.StoppedPID(pid)
	context := &EventContext{}
	for i := 4; i < 4+8 * 32; i += 8 {
		context.regs = append(context.regs, bo.Uint64(data[i:i+8]))
	}
	context.pc = bo.Uint64(data[4+8*32:4+8*33])
	context.sp = bo.Uint64(data[4+8*33:4+8*34])
	context.Print()
	this.client.Run()
}

func (this *EventContext) Print() {
	fmt.Println("─────────[ REGISTERS ]─────────")
	for i, reg := range(this.regs) {
		fmt.Printf("X%d\t%x\n", i, reg)
	}
	fmt.Printf("PC\t%x\n", this.pc)
	fmt.Printf("SP\t%x\n", this.sp)
	fmt.Println("───────────────────────────────")
}
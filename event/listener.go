package event

import(
	manager "github.com/gojue/ebpfmanager"
	"encoding/binary"
	"unsafe"
	"eDBG/controller"
	// "eDBG/utils"
	"syscall"
	"eDBG/cli"
	// "fmt"
)

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
	context := &controller.ProcessContext{}
	for i := 12; i < 12+8*30; i += 8 {
		context.Regs = append(context.Regs, bo.Uint64(data[i:i+8]))
	}
	context.LR = bo.Uint64(data[12+8*30:12+8*31])
	context.SP = bo.Uint64(data[12+8*31:12+8*32])
	context.PC = bo.Uint64(data[12+8*32:12+8*33])
	context.Pstate = bo.Uint64(data[12+8*33:12+8*34])
	this.process.Context = context
	// fmt.Println("Done data")
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
	this.process.UpdatePidList()
	bo := this.ByteOrder
	this.pid = bo.Uint32(data[4:8])
	for _, ablepid := range this.process.PidList {
		if this.pid == ablepid {
			// fmt.Printf("Suspended on pid: %d\n", this.pid)
			this.process.StoppedPID(this.pid)
			this.process.WorkPid = this.pid
			this.process.UpToDate()
			this.Incomingdata <- data
			this.client.DoClean <- true
			return
		}
	}
	
	syscall.Kill(int(this.pid), syscall.SIGCONT)
	// fmt.Println("Send DoClean")
}


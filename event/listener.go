package event

import(
	manager "github.com/gojue/ebpfmanager"
	"github.com/cilium/ebpf/perf"
	"encoding/binary"
	"unsafe"
	"eDBG/controller"
	// "eDBG/utils"
	"syscall"
	"strings"
	"eDBG/cli"
	"fmt"
)

type EventListener struct {
	pid uint32
	client *cli.Client
	process *controller.Process
	ByteOrder binary.ByteOrder
	Incomingdata chan []byte
	Record  chan perf.Record
}


func CreateEventListener(process *controller.Process) *EventListener {
	return &EventListener{
		process: process, 
		ByteOrder: getHostByteOrder(), 
		Incomingdata: make(chan []byte, 512),
		Record: make(chan perf.Record, 1),
	}
}

func (this *EventListener) SendRecord(rec perf.Record) {
	this.Record <- rec
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
	bo := this.ByteOrder
	context := &controller.ProcessContext{}
	for i := 12; i < 12+8*30; i += 8 {
		context.Regs = append(context.Regs, bo.Uint64(data[i:i+8]))
	}
	context.LR = bo.Uint64(data[12+8*30:12+8*31])
	context.SP = bo.Uint64(data[12+8*31:12+8*32])
	context.PC = bo.Uint64(data[12+8*32:12+8*33])
	if len(data) >= 284 {
		// 硬件断点无法采样 pstate
		context.Pstate = bo.Uint64(data[12+8*33:12+8*34])
	}
	this.process.Context = context
	this.client.Incoming <- true
}

func (this *EventListener) Run() {
	go func() {
		for {
			data := <- this.Incomingdata
			this.Workdata(data)
		}
	}()
}

func (this *EventListener) OnEvent(cpu int, data []byte, perfmap *manager.PerfMap, manager *manager.Manager) {
	this.process.UpdatePidList()
	bo := this.ByteOrder
	this.pid = bo.Uint32(data[4:8])
	nowTid := bo.Uint32(data[12+8*34:16+8*34])
	// fmt.Printf("Suspened on %d %d\n", this.pid, nowTid)
	PC := bo.Uint64(data[12+8*32:12+8*33])
	if this.client.BrkManager.TempBreakTid != 0 {
		if PC == this.client.BrkManager.TempAddressAbsolute || PC == 0xFFFFFFFF {
			if nowTid == this.client.BrkManager.TempBreakTid {
				this.process.WorkTid = nowTid
				// this.process.Stop()
				this.process.StoppedPID(this.pid)
				if PC == 0xFFFFFFFF {
					// 硬件断点
					dataRaw := <-this.Record
					this.Incomingdata <- dataRaw.RawSample[12:]
				} else {
					this.Incomingdata <- data
				}
				
				this.client.DoClean <- true
				return
			}
			// 单步调试断点被其他线程命中
			if PC == 0xFFFFFFFF {
				<- this.Record // 舍弃这个 Sample
			}
			// this.client.BrkManager.HasTempBreak = true // 之前脑子进水了引入了一个 bug
			syscall.Kill(int(this.pid), syscall.SIGCONT)
			return
		}
	}

	for _, ablepid := range this.process.PidList {
		if this.pid == ablepid {
			this.process.WorkPid = this.pid
			valid := false
			for _, t := range this.client.Config.ThreadFilters {
				if !t.Enable {
					continue
				}
				if t.Thread.Tid != 0 {
					valid = true
					if nowTid == t.Thread.Tid {
						this.process.WorkTid = nowTid
						this.process.StoppedPID(this.pid)
						// this.process.Stop()
						this.Incomingdata <- data
						this.client.DoClean <- true
						return
					}
					continue
				}
				if t.Thread.Name != "" {
					tList, err := this.process.GetCurrentThreads()
					if err != nil {
						fmt.Printf("WARNING: Failed to get threads: %v. Filters on thread name not working.\n", err)
						continue
					}
					found := false
					for _, tInfo := range tList {
						if strings.Contains(t.Thread.Name, tInfo.Name) {
							// 线程名称有长度限制会被截断，尽量支持用户指定完整的线程名称
							valid = true
							found = true
							if tInfo.Tid == nowTid {
								this.process.WorkTid = nowTid
								this.process.StoppedPID(this.pid)
								// this.process.Stop()
								this.Incomingdata <- data
								this.client.DoClean <- true
								return
							}
						}
					}
					if !found {
						fmt.Printf("WARNING: No thread Named %s\n", t.Thread.Name)
						continue
					}
				}
			}
			if !valid {
				// 没有可用的线程过滤器，按照 pid 工作
				this.process.WorkTid = nowTid
				this.process.StoppedPID(this.pid)
				// this.process.Stop()
				this.Incomingdata <- data
				this.client.DoClean <- true
				return
			}
			
		}
	}
	
	syscall.Kill(int(this.pid), syscall.SIGCONT)
	// fmt.Println("Send DoClean")
}


package module

import (
	"fmt"
    "errors"
    "os"
    "github.com/cilium/ebpf/perf"
	"github.com/cilium/ebpf"
)

type PerfBreaks struct {
	Fd            *perf.Reader
    Cleared       bool
    Pid           uint32
    Addr          uint64
	Type		  int
	listener      IEventListener
}

func (this *PerfBreaks) RunPerfBreak(em *ebpf.Map) error {
    pid := this.Pid 
    address := this.Addr
    eopt := perf.ExtraPerfOptions{
        UnwindStack:       false,
        ShowRegs:          true,
        PerfMmap:          false,
        BrkPid:            int(pid),
        BrkAddr:           address,
        BrkLen:            4,
        BrkType:           uint32(this.Type),
        Sample_regs_user:  (1 << 33) - 1,
        Sample_stack_user: 0,
    }
    buf := os.Getpagesize() * (1 * 1024 / 4)
    
    rd, err := perf.NewReaderWithOptions(em, buf, perf.ReaderOptions{}, eopt)
    if err != nil {
        return fmt.Errorf("Setup perf event failed: %v", err)
    }
    this.Fd = rd
    this.Cleared = false
    go func() {
        for {
            record, err := rd.ReadWithExtraOptions(&eopt)
            if err != nil {
                if errors.Is(err, perf.ErrClosed) {
                    return
                }
                fmt.Println("Got record Failed: ", err)
            }
            // fmt.Println("Got record: ", record.RawSample)
            this.listener.SendRecord(record)
        }
    }()
    return nil
}

func (this *PerfBreaks) Close() {
    if !this.Cleared {
        this.Fd.Close()
        this.Cleared = true
    }
}

func (this *ProbeHandler) AddHWBreak(pid uint32, address uint64, tp int) error {
    this.Perfs = append(this.Perfs, &PerfBreaks{
        Pid: pid,
        Addr: address,
        Cleared: true,
        Type:   tp,
        listener: this.listener,
    })
    return nil
}

func (this *ProbeHandler) SetHWBreakInternel() error {
	em, found, err := this.bpfManager.GetMap("brk_events")
    if !found {
        return fmt.Errorf("map not found")
    }
	if err != nil {
        return fmt.Errorf("Get map failed: %v", err)
    }
    for _, p := range(this.Perfs) {
		err := p.RunPerfBreak(em)
		if err != nil {
			fmt.Printf("Failed to start breakpoint at %x: %v\n", p.Addr, err)
		}
    }
    return nil
}

func (this *ProbeHandler) CloseHWBreak() {
    for _, p := range(this.Perfs) {
        p.Close()
    }
}
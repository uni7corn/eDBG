package module

import (
	"fmt"
    "errors"
    "os"
    "github.com/cilium/ebpf/perf"
	manager "github.com/gojue/ebpfmanager"
)
const (
	_PERF_TYPE_BREAKPOINT     = 5
	_PERF_COUNT_HW_BREAKPOINT = 6
	_HW_BREAKPOINT_X          = 4
	_HW_BREAKPOINT_LEN_4      = 0x40
)

type PerfBreaks struct {
	Fd            *perf.Reader
    Cleared       bool
    Pid           uint32
    Addr          uint64
	Type		  int
	bpfManager    *manager.Manager
	listener      IEventListener
}

func (this *PerfBreaks) RunPerfBreak() error {
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
    em, found, err := this.bpfManager.GetMap("brk_events")
    if !found {
        return fmt.Errorf("map not found")
    }
    if err != nil {
        return fmt.Errorf("Get map failed: %v", err)
    }
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
		bpfManager: this.bpfManager,
        listener: this.listener,
    })
    return nil
}

func (this *ProbeHandler) SetHWBreakInternel() error {
    for _, p := range(this.Perfs) {
		err := p.RunPerfBreak()
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
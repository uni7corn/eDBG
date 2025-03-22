package module

import (
    "bytes"
	"fmt"
    "errors"
    "eDBG/utils"
    "eDBG/assets"
    "path/filepath"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/btf"
    "golang.org/x/sys/unix"
    // "syscall"
    "os"
    "github.com/cilium/ebpf/perf"
    // "unsafe"
    "math"
    // "github.com/cilium/ebpf/internal/unix"
	manager "github.com/gojue/ebpfmanager"
)

const (
	_PERF_TYPE_BREAKPOINT     = 5
	_PERF_COUNT_HW_BREAKPOINT = 6
	_HW_BREAKPOINT_X          = 4
	_HW_BREAKPOINT_LEN_4      = 0x40
)

type ProbeHandler struct {
	bpfManager        *manager.Manager
    bpfManagerOptions manager.Options
    listener          IEventListener
    BTF_File          string
    PerfFd            *perf.Reader
    Perf_Cleared      bool
    Has_Perf          bool
    PerfPid           uint32
    PerfAddr          uint64
    Record            chan perf.Record
}

func CreateProbeHandler(listener IEventListener, BTF_File string) *ProbeHandler {
    return &ProbeHandler{
        listener: listener, 
        BTF_File: BTF_File, 
        Perf_Cleared: true, 
        Has_Perf: false,
        Record: make(chan perf.Record, 1),
    }
}

func (this *ProbeHandler) SetHWBreak(pid uint32, address uint64) error {
    this.Has_Perf = true
    this.PerfPid = pid
    this.PerfAddr = address
    return nil
}

func (this *ProbeHandler) SetHWBreakInternel() error {
    if !this.Has_Perf {
        return nil
    }
    pid := this.PerfPid 
    address := this.PerfAddr
    eopt := perf.ExtraPerfOptions{
        UnwindStack:       false,
        ShowRegs:          true,
        PerfMmap:          false,
        BrkPid:            int(pid),
        BrkAddr:           address,
        BrkLen:            4,
        BrkType:           _HW_BREAKPOINT_X,
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
    this.PerfFd = rd
    this.Perf_Cleared = false
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
            this.Record <- record
        }
    }()
    return nil
}

func (this *ProbeHandler) CloseHWBreak() {
    if !this.Perf_Cleared {
        this.PerfFd.Close()
        this.Perf_Cleared = true
    }
}

func (this *ProbeHandler) SetupManagerOptions() error {
    // 对于没有开启 CONFIG_DEBUG_INFO_BTF 的加载额外的 btf.Spec
    if this.BTF_File != "" {
        byteBuf, err := assets.Asset("assets/" + this.BTF_File)
        if err != nil {
            return fmt.Errorf("SetupManagerOptions failed, err:%v", err)
        }
        spec, err := btf.LoadSpecFromReader((bytes.NewReader(byteBuf)))
        if err != nil {
            return fmt.Errorf("SetupManagerOptions failed, err:%v", err)
        }
        this.bpfManagerOptions = manager.Options{
            DefaultKProbeMaxActive: 512,
            VerifierOptions: ebpf.CollectionOptions{
                Programs: ebpf.ProgramOptions{
                    LogSize:     2097152,
                    KernelTypes: spec,
                },
            },
            RLimit: &unix.Rlimit{
                Cur: math.MaxUint64,
                Max: math.MaxUint64,
            },
        }
    } else {
        this.bpfManagerOptions = manager.Options{
            DefaultKProbeMaxActive: 512,
            VerifierOptions: ebpf.CollectionOptions{
                Programs: ebpf.ProgramOptions{
                    LogSize:     2097152,
                },
            },
            RLimit: &unix.Rlimit{
                Cur: math.MaxUint64,
                Max: math.MaxUint64,
            },
        }
    }
    return nil
}

func (this *ProbeHandler) SetupManager(brks []*BreakPoint, perf bool) error {
    probes := []*manager.Probe{}
    usedCount := 0
    for i, brk := range brks {
        if !brk.Enable || brk.Deleted {
            continue
        }
        var probe *manager.Probe
        usedCount++
        if usedCount > 20 {
            return fmt.Errorf("setupManager: Failed to Set Breakpoint: %x. Breakpoint count exceed 20.", brk.Offset)
        }
        sym := utils.RandStringBytes(8)
        probe = &manager.Probe{
            Section:          fmt.Sprintf("uprobe/probe_%d", i),
            EbpfFuncName:     fmt.Sprintf("probe_%d", i),
            AttachToFuncName: sym,
            RealFilePath:     brk.LibInfo.RealFilePath,
            BinaryPath:       brk.LibInfo.LibPath,
            NonElfOffset:     brk.LibInfo.NonElfOffset,
            UAddress: brk.Offset,
        }
        probes = append(probes, probe)
    }
    

    if len(probes) == 0 {
        fmt.Println("WARNING: No valid breakpoints set. eDBG may be unable to stop the program.")
    }

    this.bpfManager = &manager.Manager{
        Probes: probes,
        PerfMaps: []*manager.PerfMap{
            &manager.PerfMap{
                Map: manager.Map{
                    Name: "events",
                },
                PerfMapOptions: manager.PerfMapOptions{
                    DataHandler: this.listener.OnEvent,
                },
            },
        },
    }
    if perf {
        // fmt.Println("setup probe_perf")
        this.bpfManager.Probes = append(this.bpfManager.Probes,
            &manager.Probe{
            	// Section:      "perf_event",
                Section: "kprobe/perf_output_sample",
            	EbpfFuncName: "probe_perf",
                AttachToFuncName: "perf_output_sample",
            })
        this.bpfManager.Maps = []*manager.Map{
            &manager.Map{
                Name: "brk_events",
            },
        }
        this.Has_Perf = true
    } else {
        this.Has_Perf = false
    }
    return nil
}

func (this *ProbeHandler) Run() error {
    var bpfFileName = filepath.Join("assets", "ebpf_module.o")
    byteBuf, err := assets.Asset(bpfFileName)

    if err != nil {
        return fmt.Errorf("ProbeHandler.Run(): couldn't find asset %v .", err)
    }

    if err = this.bpfManager.InitWithOptions(bytes.NewReader(byteBuf), this.bpfManagerOptions); err != nil {
        return fmt.Errorf("ProbeHandler.Run(): couldn't init manager %v", err)
    }
    // if err = this.bpfManager.Init(bytes.NewReader(byteBuf)); err != nil {
    //     return fmt.Errorf("ProbeHandler.Run(): couldn't init manager %v", err)
    // }

    if err = this.bpfManager.Start(); err != nil {
        return fmt.Errorf("ProbeHandler.Run(): couldn't start bootstrap manager %v .", err)
    }
    if err = this.SetHWBreakInternel(); err != nil {
        return fmt.Errorf("Failed to set up Hardware breakpoint: %v", err)
    }
    // fmt.Println("Module Running...")
    return nil
}

func (this *ProbeHandler) Stop() error {
    // fmt.Println("Module Stopping...")
    this.CloseHWBreak()
    return this.bpfManager.Stop(manager.CleanAll)
}


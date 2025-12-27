package module

import (
    "bytes"
	"fmt"
    "eDBG/utils"
    "eDBG/assets"
    "path/filepath"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/btf"
    "golang.org/x/sys/unix"
    "math"
	manager "github.com/gojue/ebpfmanager"
)

type ProbeHandler struct {
	bpfManager        *manager.Manager
    bpfManagerOptions manager.Options
    listener          IEventListener
    BTF_File          string
    Perfs             []*PerfBreaks
}

func CreateProbeHandler(listener IEventListener, BTF_File string) *ProbeHandler {
    return &ProbeHandler{
        listener: listener, 
        BTF_File: BTF_File, 
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

func (this *ProbeHandler) SetupManager(brks []*BreakPoint) error {
    perf := false
    this.Perfs = []*PerfBreaks{}
    probes := []*manager.Probe{}
    usedCount := 0
    for i, brk := range brks {
        if !brk.Enable || brk.Deleted {
            continue
        }
        if brk.Hardware {
            this.AddHWBreak(brk.Pid, brk.Addr.Absolute, brk.Type)
            perf = true
            continue
        }
        var probe *manager.Probe
        usedCount++
        if usedCount > 20 {
            return fmt.Errorf("setupManager: Failed to Set Breakpoint: %x. Breakpoint count exceed 20.", brk.Addr.Absolute)
        }
        sym := utils.RandStringBytes(8)
        probe = &manager.Probe{
            Section:          fmt.Sprintf("uprobe/probe_%d", i),
            EbpfFuncName:     fmt.Sprintf("probe_%d", i),
            AttachToFuncName: sym,
            RealFilePath:     brk.Addr.LibInfo.RealFilePath,
            BinaryPath:       brk.Addr.LibInfo.LibPath,
            NonElfOffset:     brk.Addr.LibInfo.NonElfOffset,
            UAddress: brk.Addr.Offset,
            UprobeOffset: 0,
        }
        // fmt.Printf("Set uprobe: %s[%x]+%x\n", brk.Addr.LibInfo.RealFilePath, brk.Addr.LibInfo.NonElfOffset, brk.Addr.Offset)
        probes = append(probes, probe)
    }
    

    if len(probes) == 0 {
        fmt.Println("WARNING: No valid uprobe breakpoints set.")
        // return fmt.Errorf("No valid reakpoints")
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
        this.bpfManager.Probes = append(this.bpfManager.Probes,
            &manager.Probe{
                Section: "kprobe/perf_output_sample",
            	EbpfFuncName: "probe_perf",
                AttachToFuncName: "perf_output_sample",
            })
        this.bpfManager.Maps = []*manager.Map{
            &manager.Map{
                Name: "brk_events",
            },
        }
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

    if err = this.bpfManager.Start(); err != nil {
        return fmt.Errorf("ProbeHandler.Run(): couldn't start bootstrap manager %v .", err)
    }
    if err = this.SetHWBreakInternel(); err != nil {
        return fmt.Errorf("Failed to set up Hardware breakpoint: %v", err)
    }
    return nil
}

func (this *ProbeHandler) Stop() error {
    this.CloseHWBreak()
    return this.bpfManager.Stop(manager.CleanAll)
}


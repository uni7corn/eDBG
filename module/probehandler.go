package module

import (
    "bytes"
	"fmt"
    "eDBG/utils"
    "eDBG/assets"
    "path/filepath"
	// "github.com/cilium/ebpf"
	// "github.com/cilium/ebpf/btf"
	manager "github.com/gojue/ebpfmanager"
)

type ProbeHandler struct {
	bpfManager        *manager.Manager
    listener          IEventListener
}

func CreateProbeHandler(listener IEventListener) *ProbeHandler {
    return &ProbeHandler{listener: listener}
}

func (this *ProbeHandler) SetupManager(brks []*BreakPoint) error {
    probes := []*manager.Probe{}
    usedCount := 0
    for i, brk := range brks {
        if !brk.enable || brk.deleted {
            continue
        }
        var probe *manager.Probe
        usedCount++
        if usedCount > 20 {
            return fmt.Errorf("setupManager: Failed to Set Breakpoint: %x. Breakpoint count exceed 20.", brk.offset)
        }
        sym := utils.RandStringBytes(8)
        probe = &manager.Probe{
            Section:          fmt.Sprintf("uprobe/probe_%d", i+1),
            EbpfFuncName:     fmt.Sprintf("probe_%d", i+1),
            AttachToFuncName: sym,
            RealFilePath:     brk.libInfo.RealFilePath,
            BinaryPath:       brk.libInfo.LibPath,
            NonElfOffset:     brk.libInfo.NonElfOffset,
            UAddress: brk.offset,
        }
        probes = append(probes, probe)
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
    return nil
}

func (this *ProbeHandler) Run() error {
    var bpfFileName = filepath.Join("assets", "ebpf_module.o")
    byteBuf, err := assets.Asset(bpfFileName)

    if err != nil {
        return fmt.Errorf("ProbeHandler.Run(): couldn't find asset %v .", err)
    }

    if err = this.bpfManager.Init(bytes.NewReader(byteBuf)); err != nil {
        return fmt.Errorf("ProbeHandler.Run(): couldn't init manager %v", err)
    }

    if err = this.bpfManager.Start(); err != nil {
        return fmt.Errorf("ProbeHandler.Run(): couldn't start bootstrap manager %v .", err)
    }
    // fmt.Println("Module Running...")
    return nil
}

func (this *ProbeHandler) Stop() error {
    // fmt.Println("Module Stopping...")
    return this.bpfManager.Stop(manager.CleanAll)
}


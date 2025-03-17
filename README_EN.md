<div align="center">

[简体中文](README.md) | English

<img src="logo.png"/> 
</div>

> eDBG is a lightweight CLI debugger based on eBPF.<br />
>
> Compared to traditional ptrace-based debuggers, eDBG doesn't directly intrude or attach to processes, offering stronger resistance to interference and anti-detection capabilities.

## ✨ Features

- eBPF implementation introduces minimal footprint, making it harder to be detected by target programs
- Supports common debugging functionalities (see "Command Details")
- Uses pwndbg-like CLI interface with GDB-style interactions for ease of use
- File+offset based breakpoint registration enables quick startup and supports multi-thread/process debugging

## 💕 Demo

![](demo.png)

## 🚀 Requirements

- Currently only supports ARM64 Android devices with ROOT access (Recommended with [KernelSU](https://github.com/tiann/KernelSU))
- Kernel version 5.10+ (Check via `uname -r`)

## ⚙️ Usage

Download prebuilt binaries from Releases:

1. Push to device and grant permissions:
   ```shell
   adb push eDBG /data/local/tmp
   adb shell
   su
   chmod +x /data/local/tmp/eDBG


2. Start debugger:

   ```shell
   ./eDBG -p com.package.name -l libname.so -b 0x123456
   ```

   |      Option       |                   Description                   |
   | :---------------: | :---------------------------------------------: |
   |        -p         |        Required: Target app package name        |
   |        -l         |      Required: Target shared library name       |
   |        -b         | Optional: Initial breakpoints (comma-separated) |
   |  -hide-register   | Optional: Disable register info on breakpoints  |
   | -hide-disassemble | Optional: Disable assembly info on breakpoints  |
   |        -t         |      Optional: Thread name filter for eBPF      |

2. Launch target app:

   > eDBG can attach to running processes but won't auto-launch apps.

## ⚠️ Notes

- Debugging system libraries (e.g., `libc.so`, `libart.so`) may cause lag due to file+offset mechanism
- Program pause isn't supported without active breakpoints
- Thread ID specification during startup isn't supported
- Maximum 20 active breakpoints

## 💡 Commands

- **Breakpoints** `break/b`

  - Offset: `b 0x1234` (relative to debugger's initial library)
  - Memory: `b 0x6e9bfe214c` (requires running process)
  - Library+Offset: `b library.so+0x1234`
  - Relative: `b $+1` (current position +1 instruction)

- **Stepping**

  - `step/s`: Step into functions
  - `next/n`: Step over functions

- **Memory Examination** `examine/x`

  - Address: `x 0x12345678` (default 16 bytes)
  - Address+Length: `x 0x12345678 128`
  - Register: `x X0` (access [X0] memory)
  - Register+Length: `x X0 128`

- **Continue** `continue/c`: Resume execution

- **Memory Display** `display/disp`

  - Address: `disp 0x123456` (auto-print on breaks/steps)
  - Address+Length: `disp 0x123456 128`
  - Named: `disp 0x123456 128 name`

  > ⚠️ Memory address changes (e.g., app restart) may invalidate displays

- **Exit** `quit/q`: Exit debugger (won't affect target process)

- **Undisplay** `undisplay/undisp <id>`: Remove auto-display

- **Code Listing** `list/l/disassemble/dis`

  - Current: `l` (10 instructions from PC)
  - Specific: `l 0x1234` (10 instructions)
  - Custom: `l 0x1234 20` (20 instructions)

- **Information** `info/i`

  - `info b/break`: List breakpoints (`[+]`=enabled, `[-]`=disabled)
  - `info register/reg/r`: Show registers
  - `info thread/t`: List threads & filters

- **Function Finish** `finish/fi`: Execute until function return

- **Breakpoint Management**

  - `enable <id>`: Enable breakpoint
  - `disable <id>`: Disable breakpoint
  - `delete <id>`: Remove breakpoint

- **Run Until** `until/u <address>`: Execute to specified address

- **Thread Control** `thread/t`

  - `t`: List threads
  - `t +0`: Add thread filter (use `info t` for IDs)
  - `t -0`: Remove filter
  - `t all`: Clear all filters
  - `t +n threadname`: Filter by thread name

- **Repeat Command**: Press Enter with empty input

## 🛫 Compilation

1. **Environment Setup** (x86 Linux cross-compilation)

   ```shell
   sudo apt-get update
   sudo apt-get install golang==1.18 clang==14
   export GOPROXY=https://goproxy.cn,direct
   export GO111MODULE=on

2. **Build**

   ```shell
   git clone --recursive https://github.com/ShinoLeah/eDBG.git
   make
   ```

## 💭 Implementation

- Basic uprobe-based debugging with SIGSTOP/SIGCONT signals (Documentation in progress)

## 🧑‍💻 Todo

- Hide uprobe traces in maps
- Save/load configurations
- Frame support
- Backtrace functionality
- Watchpoints

## 🤝 References

- [SeeFlowerX/stackplz](https://github.com/SeeFlowerX/stackplz/tree/dev)
- [pwndbg](https://github.com/pwndbg/pwndbg)

## ❤️ Support

- Star this repo 🌟 if you find it useful
- Issues and PRs are welcome!
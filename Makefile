CMD_CLANG ?= $(shell brew --prefix llvm)/bin/clang
CMD_GO ?= go
CMD_RM ?= rm
# bcc-tools 包含 bpftool
CMD_BPFTOOL ?= docker run --rm -v $(CURDIR):/src -w /src bpftool
# CMD_BPFTOOL ?= bpftool
BUILD_PATH ?= ./build

DEBUG_PRINT ?=
LINUX_ARCH = arm64
ifeq ($(DEBUG),1)
DEBUG_PRINT := -DDEBUG_PRINT
endif

BUILD_TAGS ?=
TARGET_ARCH = $(LINUX_ARCH)

ifeq ($(BUILD_TAGS),forarm)
BUILD_TAGS := -tags forarm
TARGET_ARCH = arm
endif

CC=/opt/homebrew/Caskroom/android-ndk/28b/AndroidNDK13356709.app/Contents/NDK/toolchains/llvm/prebuilt/darwin-x86_64/bin/aarch64-linux-android29-clang
CXX=/opt/homebrew/Caskroom/android-ndk/28b/AndroidNDK13356709.app/Contents/NDK/toolchains/llvm/prebuilt/darwin-x86_64/bin/aarch64-linux-android29-clang++



.PHONY: all
all: ebpf_module genbtf assets build 
	@echo $(shell date)

.PHONY: cleanß
clean:
	$(CMD_RM) -f assets/*.d
	$(CMD_RM) -f assets/*.o
	$(CMD_RM) -f assets/ebpf_probe.go
	$(CMD_RM) -f bin/eDBG_$(TARGET_ARCH)

.PHONY: ebpf_module
ebpf_module:
	$(CMD_CLANG) \
	-D__TARGET_ARCH_$(TARGET_ARCH) \
	--target=bpf \
	-c \
	-nostdlibinc \
	-no-canonical-prefixes \
	-O2 \
	-I       libbpf/src \
	-I       ebpf_module \
	-g \
	-o assets/ebpf_module.o \
	ebpf_module/ebpf_module.c

.PHONY: assets
assets:
	$(CMD_GO) run github.com/shuLhan/go-bindata/cmd/go-bindata -pkg assets -o "assets/ebpf_probe.go" $(wildcard ./config/config_syscall_*.json ./assets/*.o ./assets/*_min.btf ./preload_libs/*.so)

.PHONY: genbtf
genbtf:
	$(CMD_BPFTOOL) gen min_core_btf assets/rock5b-5.10-f9d1b1529-arm64.btf assets/rock5b-5.10-arm64_min.btf assets/ebpf_module.o
	$(CMD_BPFTOOL) gen min_core_btf assets/a12-5.10-arm64.btf assets/a12-5.10-arm64_min.btf assets/ebpf_module.o
# genbtf:
# 	cd assets && $(CMD_BPFTOOL) gen min_core_btf rock5b-5.10-f9d1b1529-arm64.btf rock5b-5.10-arm64_min.btf ebpf_module.o
# 	cd assets && $(CMD_BPFTOOL) gen min_core_btf a12-5.10-arm64.btf a12-5.10-arm64_min.btf ebpf_module.o

.PHONY: build
build:
	CGO_ENABLED=1 CC=$(CC) CXX=$(CXX) GOARCH=arm64 GOOS=android $(CMD_GO) build $(BUILD_TAGS) -ldflags "-w -s -extldflags '-Wl,--hash-style=sysv'" -o bin/eDBG_$(TARGET_ARCH) .

#CGO_ENABLED=1 
#CC=/opt/homebrew/Caskroom/android-ndk/28b/AndroidNDK13356709.app/Contents/NDK/toolchains/llvm/prebuilt/darwin-x86_64/bin/aarch64-linux-android29-clang

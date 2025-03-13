CMD_CLANG ?= clang
CMD_GO ?= go
CMD_RM ?= rm
# CMD_BPFTOOL ?= bpftool
# ASSETS_PATH ?= user/assets
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

.PHONY: all
all: ebpf_module assets build
	@echo $(shell date)

.PHONY: clean
clean:
	$(CMD_RM) -f assets/*.d
	$(CMD_RM) -f assets/*.o
	$(CMD_RM) -f assets/ebpf_probe.go
	$(CMD_RM) -f bin/eDBG_$(TARGET_ARCH)

.PHONY: ebpf_module
ebpf_module:
	clang \
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
	$(CMD_GO) run github.com/shuLhan/go-bindata/cmd/go-bindata -pkg assets -o "assets/ebpf_probe.go" $(wildcard ./config/config_syscall_*.json ./assets/*.o)

.PHONY: build
build:
	GOARCH=arm64 GOOS=android $(CMD_GO) build $(BUILD_TAGS) -ldflags "-w -s -extldflags '-Wl,--hash-style=sysv'" -o bin/eDBG_$(TARGET_ARCH) .

# CGO_ENABLED=1 CC=aarch64-linux-android29-clang

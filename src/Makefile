CMD_CLANG ?= clang
CMD_GO ?= go
CMD_RM ?= rm
# CMD_BPFTOOL ?= bpftool
# ASSETS_PATH ?= user/assets
BUILD_PATH ?= build

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
all: ebpf_module
	@echo $(shell date)

.PHONY: clean
clean:
	$(CMD_RM) -f $(BUILD_PATH)/assets/*.d
	$(CMD_RM) -f $(BUILD_PATH)/assets/*.o

.PHONY: ebpf_module
	clang \
	-D__TARGET_ARCH_$(TARGET_ARCH) \
	-D__MODULE_STACK \
	--target=bpf \
	-c \
	-nostdlibinc \
	-no-canonical-prefixes \
	-O2 \
	$(DEBUG_PRINT)	\
	-I       ../libbpf/src \
	-I       src \
	-g \
	-o $(BUILD_PATH)/assets/ebpf_module.o \
	src/ebpf/ebpf_module.c

.PHONY: assets
assets:
	$(CMD_GO) run github.com/shuLhan/go-bindata/cmd/go-bindata -pkg assets -o "assets/ebpf_probe.go" $(wildcard ./user/config/config_syscall_*.json ./user/assets/*.o ./user/assets/*_min.btf ./preload_libs/*.so)

.PHONY: build
build:
	GOARCH=arm64 GOOS=android CGO_ENABLED=1 CC=aarch64-linux-android29-clang $(CMD_GO) build $(BUILD_TAGS) -ldflags "-w -s -extldflags '-Wl,--hash-style=sysv'" -o bin/stackplz_$(TARGET_ARCH) .
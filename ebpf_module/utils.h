#include "include/vmlinux_510.h"

#include "include/bpf/bpf_helpers.h"
// #include "include/bpf/bpf_core_read.h"
#include "include/bpf/bpf_tracing.h"

char __license[] SEC("license") = "GPL";
__u32 _version SEC("version") = 0xFFFFFFFE;


#define READ_KERN(ptr)                                                                         \
    ({                                                                                         \
        typeof(ptr) _val;                                                                      \
        __builtin_memset((void *) &_val, 0, sizeof(_val));                                     \
        bpf_probe_read((void *) &_val, sizeof(_val), &ptr);                                    \
        _val;                                                                                  \
    })
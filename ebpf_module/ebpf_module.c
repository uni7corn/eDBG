#include "utils.h"

struct data_t {
    __u64 pid;
	__u64 regs[31];
	__u64 sp;
	__u64 pc;
    __u64 pstate;
    __u64 tid;
};

struct {                                                                                     
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);                                                                     
    __uint(max_entries, 1);                                                         
    __type(key, u32);                                                                    
    __type(value, struct data_t);                                                                
} event_map SEC(".maps");

struct {                                                                                       
    __uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);                                                                       
    __uint(max_entries, 1024);                                                         
    __type(key, int);                                                                    
    __type(value, __u32);                                                                
} events SEC(".maps");


static __always_inline u32 do_probe(struct pt_regs* ctx, u32 point_key) {
    __u32 zero = 0;
    struct data_t *data = bpf_map_lookup_elem(&event_map, &zero);
    if (!data) return 0; 
    // struct data_t data;
    // data->pid = (u32)(bpf_get_ns_current_pid_tgid() >> 32);
    // struct task_struct *task = (struct task_struct *) bpf_get_current_task();
    // struct task_struct *group_leader = READ_KERN(task->group_leader);
    // data->pid = (u32) get_task_pid(group_leader);
    // data->tid = (__u64) get_task_pid(task);

    data->pid = bpf_get_current_pid_tgid() >> 32;
    data->tid = (__u32)bpf_get_current_pid_tgid();

    for(int i = 0; i < 31; ++i) {
        bpf_probe_read_kernel(&data->regs[i], sizeof(data->regs[i]), &ctx->regs[i]);
    }
    bpf_probe_read_kernel(&data->sp, sizeof(data->sp), &ctx->sp);
    bpf_probe_read_kernel(&data->pc, sizeof(data->pc), &ctx->pc);
    bpf_probe_read_kernel(&data->pstate, sizeof(data->pstate), &ctx->pstate);
    bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, data, sizeof(struct data_t));
    bpf_send_signal(19);
    // bpf_send_signal_task(group_leader, 19, PIDTYPE_TGID, 8);
    return 0;
}


#define PROBE(name)                          \
    SEC("uprobe/probe_##name")                     \
    int probe_##name(struct pt_regs* ctx)    \
    {                                              \
        u32 point_key = name;                       \
        return do_probe(ctx, point_key);    \
    }


PROBE(0) // Temporary breakpoint for singlestep
PROBE(1)
PROBE(2)
PROBE(3)
PROBE(4)
PROBE(5)
PROBE(6)
PROBE(7)
PROBE(8)
PROBE(9)
PROBE(10)
PROBE(11)
PROBE(12)
PROBE(13)
PROBE(14)
PROBE(15)
PROBE(16)
PROBE(17)
PROBE(18)
PROBE(19)
PROBE(20)
PROBE(21)
PROBE(22)
PROBE(23)

struct {
    __uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
} brk_events SEC(".maps");

SEC("kprobe/perf_output_sample")
int probe_perf(struct pt_regs *ctx ) {
    __u32 zero = 0;
    struct data_t *data = bpf_map_lookup_elem(&event_map, &zero);
    if (!data) return 0; 
    data->pid = bpf_get_current_pid_tgid() >> 32;
    data->tid = (__u32)bpf_get_current_pid_tgid();
    for(int i = 0; i < 31; ++i) {
        bpf_probe_read_kernel(&data->regs[i], sizeof(data->regs[i]), &ctx->regs[i]);
    }
    data->pc = 0xffffffff; // 从 fd 读寄存器数据
    bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, data, sizeof(struct data_t));
    bpf_send_signal(19);
    return 0;
}

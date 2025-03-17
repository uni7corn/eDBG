package controller

import (
	"fmt"
    "io/ioutil"
    // "os"
    "path/filepath"
    "strings"
)

type Thread struct {
	Tid uint32
	Name string
}

func (this *Process) UpdateThreadsPid(pid uint32) error {
    taskDir := fmt.Sprintf("/proc/%d/task", pid)
    files, err := ioutil.ReadDir(taskDir)
    if err != nil {
        return fmt.Errorf("Could not read task proc dir: %v", err)
    }

    threads := []*Thread{}
    for _, f := range files {
        tid := f.Name()
        commPath := filepath.Join(taskDir, tid, "comm")
        commContent, err := ioutil.ReadFile(commPath)
        if err != nil {
            continue // 忽略无法读取的线程
        }
		t := &Thread{}
        t.Name = strings.TrimSpace(string(commContent))
		fmt.Sscanf(tid, "%d", &t.Tid)
		threads = append(threads, t)
    }

	this.Threads[pid] = threads
	this.ThreadsUpToDate[pid] = true
    return nil
}

func (this *Process) GetCurrentThreads() ([]*Thread, error) {
	if val, ok := this.ThreadsUpToDate[this.WorkPid]; !ok || !val {
		err := this.UpdateThreadsPid(this.WorkPid)
		if err != nil {
			return []*Thread{}, fmt.Errorf("Failed to get thread infos on pid %d : %v", this.WorkPid, err)
		}
	}
	tList, ok := this.Threads[this.WorkPid]
	if !ok {
		return []*Thread{}, fmt.Errorf("Failed to get thread infos on pid %d unexpectedly.", this.WorkPid)
	}
	return tList, nil
}

func (this *Process) PrintThreads() {
	tList, err := this.GetCurrentThreads()
	if err != nil {
		fmt.Println(err)
		return
	}
	for id, t := range tList {
		fmt.Printf("[%d] %d: %s\n", id, t.Tid, t.Name)
	}
}
package main

import (
	"fmt"
	"os"
	"os/exec"

	phpstart "github.com/paketo-buildpacks/php-start"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "USAGE:")
		fmt.Fprintln(os.Stderr, "    procmgr <path-to-proc-file>")
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}

	procs, err := phpstart.ReadProcs(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading/parsing procs file:", err)
		os.Exit(2)
	}

	if err := runProcs(procs); err != nil {
		fmt.Fprintln(os.Stderr, "error running procs:", err)
		os.Exit(2)
	}
}

type procMsg struct {
	ProcName string
	Cmd      *exec.Cmd
	Err      error
}

func runProcs(procs phpstart.Procs) error {
	msgs := make(chan procMsg)

	for procName, proc := range procs.Processes {
		go runProc(procName, proc, msgs)
	}

	msg := <-msgs
	fmt.Fprintln(os.Stderr, "process", msg.ProcName, "exited, status:", msg.Cmd.ProcessState)
	return msg.Err
}

func runProc(procName string, proc phpstart.Proc, msgs chan procMsg) {
	cmd := exec.Command(proc.Command, proc.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		msgs <- procMsg{procName, cmd, err}
	}

	err = cmd.Wait()
	if err != nil {
		msgs <- procMsg{procName, cmd, err}
	}

	msgs <- procMsg{procName, cmd, nil}
}

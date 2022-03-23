package phpstart

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v2"
)

// Procs is the existing list of process names and commands to run
type Procs struct {
	Processes map[string]Proc
}

// Proc is a single process to run
type Proc struct {
	Command string
	Args    []string
}

func NewProc(command string, args []string) Proc {
	return Proc{Command: command, Args: args}
}

func NewProcs() Procs {
	return Procs{
		Processes: map[string]Proc{},
	}
}

// Add takes a process and a name, and adds it to the process list.
func (procs Procs) Add(procName string, newProc Proc) {
	procs.Processes[procName] = newProc
}

// WriteFile writes a Procs process list into YAML onto the given path
func (procs Procs) WriteFile(path string) error {
	bytes, err := yaml.Marshal(procs)
	if err != nil {
		//untested
		return err
	}
	return os.WriteFile(path, bytes, 0644)
}

// ReadProcs is a utility function that given a path to `procs.yml`, will
// unmarshall it into a Procs process list.
func ReadProcs(path string) (Procs, error) {
	procs := Procs{}

	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return Procs{
			Processes: map[string]Proc{},
		}, nil
	} else if err != nil {
		return Procs{}, fmt.Errorf("failed to open proc.yml: %w", err)
	}
	defer file.Close()

	contents, err := io.ReadAll(file)
	if err != nil {
		return Procs{}, err
	}

	err = yaml.UnmarshalStrict(contents, &procs)
	if err != nil {
		return Procs{}, fmt.Errorf("invalid proc.yml contents:\n %q: %w", contents, err)
	}

	return procs, nil
}

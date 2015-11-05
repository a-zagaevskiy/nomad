package executor

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/hashicorp/nomad/client/allocdir"
	"github.com/hashicorp/nomad/client/driver/args"
	"github.com/hashicorp/nomad/client/driver/environment"
	"github.com/hashicorp/nomad/nomad/structs"
)

// BasicExecutor should work everywhere, and as a result does not include
// any resource restrictions or runas capabilities.
type BasicExecutor struct {
	cmd exec.Cmd
}

// TODO: Have raw_exec use this as well.
func NewBasicExecutor() Executor {
	return &BasicExecutor{}
}

func (e *BasicExecutor) Limit(resources *structs.Resources) error {
	if resources == nil {
		return errNoResources
	}
	return nil
}

func (e *BasicExecutor) ConfigureTaskDir(taskName string, alloc *allocdir.AllocDir) error {
	taskDir, ok := alloc.TaskDirs[taskName]
	if !ok {
		return fmt.Errorf("Error finding task dir for (%s)", taskName)
	}
	e.cmd.Dir = taskDir
	return nil
}

func (e *BasicExecutor) Start() error {
	// Parse the commands arguments and replace instances of Nomad environment
	// variables.
	envVars, err := environment.ParseFromList(e.cmd.Env)
	if err != nil {
		return err
	}

	parsedPath, err := args.ParseAndReplace(e.cmd.Path, envVars.Map())
	if err != nil {
		return err
	} else if len(parsedPath) != 1 {
		return fmt.Errorf("couldn't properly parse command path: %v", e.cmd.Path)
	}

	e.cmd.Path = parsedPath[0]
	combined := strings.Join(e.cmd.Args, " ")
	parsed, err := args.ParseAndReplace(combined, envVars.Map())
	if err != nil {
		return err
	}
	e.cmd.Args = parsed

	// We don't want to call ourself. We want to call Start on our embedded Cmd
	return e.cmd.Start()
}

func (e *BasicExecutor) Open(pid string) error {
	pidNum, err := strconv.Atoi(pid)
	if err != nil {
		return fmt.Errorf("Failed to parse pid %v: %v", pid, err)
	}

	process, err := os.FindProcess(pidNum)
	if err != nil {
		return fmt.Errorf("Failed to reopen pid %d: %v", pidNum, err)
	}
	e.cmd.Process = process
	return nil
}

func (e *BasicExecutor) Wait() error {
	// We don't want to call ourself. We want to call Start on our embedded Cmd
	return e.cmd.Wait()
}

func (e *BasicExecutor) ID() (string, error) {
	if e.cmd.Process != nil {
		return strconv.Itoa(e.cmd.Process.Pid), nil
	} else {
		return "", fmt.Errorf("Process has finished or was never started")
	}
}

func (e *BasicExecutor) Shutdown() error {
	return e.ForceStop()
}

func (e *BasicExecutor) ForceStop() error {
	return e.cmd.Process.Kill()
}

func (e *BasicExecutor) Command() *exec.Cmd {
	return &e.cmd
}

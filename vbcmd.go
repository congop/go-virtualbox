package virtualbox

import (
	"bytes"
	"errors"
	"os/exec"
	"runtime"
)

type option func(Command)

// Command is the mock-able interface to run VirtualBox commands
// such as VBoxManage (host side) or VBoxControl (guest side)
type Command interface {
	setOpts(opts ...option) Command
	isGuest() bool
	path() string
	run(args ...string) error
	runOut(args ...string) (string, error)
	runOutErr(args ...string) (string, string, error)
}

var (
	// Verbose toggles the library in verbose execution mode.
	Verbose bool
	// ErrMachineExist holds the error message when the machine already exists.
	ErrMachineExist = errors.New("machine already exists")
	// ErrMachineNotExist holds the error message when the machine does not exist.
	ErrMachineNotExist = errors.New("machine does not exist")
	// ErrCommandNotFound holds the error message when the VBoxManage commands was not found.
	ErrCommandNotFound = errors.New("command not found")
)

type command struct {
	program string
	sudoer  bool // Is current user a sudoer?
	sudo    bool // Is current command expected to be run under sudo?
	guest   bool
}

func (vbcmd command) setOpts(opts ...option) Command {
	var cmd Command = &vbcmd
	for _, opt := range opts {
		opt(cmd)
	}
	return cmd
}

func sudo(sudo bool) option {
	return func(cmd Command) {
		vbcmd := cmd.(*command)
		vbcmd.sudo = sudo
		Trace("Next sudo: %v", vbcmd.sudo)
	}
}

func (vbcmd command) isGuest() bool {
	return vbcmd.guest
}

func (vbcmd command) path() string {
	return vbcmd.program
}

func (vbcmd command) prepare(args []string) *exec.Cmd {
	program := vbcmd.program
	argv := []string{}
	Trace("Command: '%+v', runtime.GOOS: '%s'", vbcmd, runtime.GOOS)
	if vbcmd.sudoer && vbcmd.sudo && runtime.GOOS != osWindows {
		program = "sudo"
		argv = append(argv, vbcmd.program)
	}
	argv = append(argv, args...)
	Trace("executing: %v %v", program, argv)
	return exec.Command(program, argv...) // #nosec
}

func (vbcmd command) run(args ...string) error {
	defer vbcmd.setOpts(sudo(false))
	cmd := vbcmd.prepare(args)
	if Verbose {
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		defer func() {
			stdoutStr := stdout.String()
			if stdoutStr != "" {
				Debug("Stdout@%v =>[[ %s ]]\n", cmd.String(), stdoutStr)
			}

			stderrStr := stderr.String()
			if stderrStr != "" {
				Debug("Stderr@%v =>[[ %s ]] \n", cmd.String(), stderrStr)
			}
		}()
	}
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.Error); ok && ee == exec.ErrNotFound {
			return ErrCommandNotFound
		}
		return err
	}
	return nil
}

func (vbcmd command) runOut(args ...string) (string, error) {
	defer vbcmd.setOpts(sudo(false))
	cmd := vbcmd.prepare(args)
	if Verbose {
		var stderr bytes.Buffer
		// Users of this module may not have a say on stdout/stderr
		// But they usually are able to configure logging and Debug.
		// We are therefore giving them the opportunity to receive the
		// command run output
		cmd.Stderr = &stderr
		defer func() {
			stderrStr := stderr.String()
			if stderrStr != "" {
				Debug("Stderr@%v =>[[ %s ]] \n", cmd.String(), stderrStr)
			}
		}()
	}

	b, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.Error); ok && ee == exec.ErrNotFound {
			err = ErrCommandNotFound
		}
	}
	return string(b), err
}

func (vbcmd command) runOutErr(args ...string) (string, string, error) {
	defer vbcmd.setOpts(sudo(false))
	cmd := vbcmd.prepare(args)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if ee, ok := err.(*exec.Error); ok && ee == exec.ErrNotFound {
			err = ErrCommandNotFound
		}
	}
	return stdout.String(), stderr.String(), err
}

// RunVBoxManageCmd run VBoxManage with the given arguments.
func RunVBoxManageCmd(args ...string) (sdterr string, stdout string, err error) {
	return Manage().runOutErr(args...)
}

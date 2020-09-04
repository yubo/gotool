package util

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"k8s.io/klog/v2"
)

type Cmd struct {
	name string
	args []string
	env  []string

	stdin  *os.File
	stdout *os.File
	stderr *os.File
}

func New(name string) *Cmd {
	return &Cmd{
		name:   name,
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

func (p *Cmd) Args(args ...string) *Cmd {
	p.args = append(p.args, args...)
	return p
}

func (p *Cmd) Env(key, value string) *Cmd {
	p.env = append(p.env, fmt.Sprintf("%s=%s", key, value))
	return p
}

func (p *Cmd) String() string {
	return fmt.Sprintf("%s %s %s",
		strings.Join(p.env, " "),
		p.name,
		strings.Join(p.args, " "))
}

func (p *Cmd) Run() error {
	cmd := exec.Command(p.name, p.args...)

	if len(p.env) > 0 {
		cmd.Env = append(os.Environ(), p.env...)
	}

	cmd.Stdin = p.stdin
	cmd.Stdout = p.stdout
	cmd.Stderr = p.stderr

	klog.V(3).Infof("cmd.Run %s", p.String())

	return cmd.Run()
}

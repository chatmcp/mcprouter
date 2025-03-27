package mcpclient

import (
	"io"
	"os/exec"
)

type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func NewStdioTransport(command string) (*StdioTransport, error) {

	cmd := exec.Command(
		"sh",
		"-c",
		command,
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	return &StdioTransport{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}, nil
}

func (t *StdioTransport) Write(p []byte) (n int, err error) {
	return t.stdin.Write(p)
}

func (t *StdioTransport) Close() error {
	return t.stdin.Close()
}

func (t *StdioTransport) Read(p []byte) (n int, err error) {
	return t.stdout.Read(p)
}

func (t *StdioTransport) GetStdErrOutput() io.ReadCloser {
	return t.stderr
}

func (t *StdioTransport) Start() {
	t.cmd.Start()
}

func (t *StdioTransport) Stop() error {
	return t.cmd.Wait()
}

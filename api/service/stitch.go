package service

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
)

type Stitcher interface {
	Stitch(out io.Writer, in io.Reader) (string, error)
}

type ExecStitch struct {
	cmd     []string
	profile bool
}

func NewExecStitch(cmdArgs []string, profile bool) (*ExecStitch, error) {
	execName := cmdArgs[0]

	inf, err := os.Stat(execName)
	if err != nil {
		return nil, err
	}

	if inf.IsDir() {
		return nil, fmt.Errorf("Cannot be a directory")
	}
	return &ExecStitch{cmdArgs, profile}, nil
}

func (es *ExecStitch) Stitch(out io.Writer, in io.Reader) (string, error) {

	var e *exec.Cmd
	var pBuf *bytes.Buffer

	if es.profile {

		pr, pw, err := NewNamedPipe()
		if err != nil {
			return "", fmt.Errorf("Error opening profiling pipe: %v", err)
		}
		defer pw.Close()

		b := append(es.cmd[1:], "--profile", pr.Name())
		e = exec.Command(es.cmd[0], b...)
		pBuf = bytes.NewBuffer(make([]byte, 0))
		go func() {
			defer pr.Close()
			defer os.Remove(pr.Name())

			if _, err := io.Copy(pBuf, pr); err != nil {
				fmt.Printf("Profiling stitch failed: %v\n", err)
			}
		}()
	} else {
		e = exec.Command(es.cmd[0], es.cmd[1:]...)
	}
	errBuf := bytes.NewBuffer(make([]byte, 0))
	e.Stdout = out
	e.Stdin = in
	e.Stderr = errBuf

	err := e.Run()
	if err != nil {
		return "", fmt.Errorf("Execute stitch failed: %v - %v ", err, string(errBuf.Bytes()))
	}
	e.Wait()

	if es.profile {
		return pBuf.String(), nil
	}

	return "", nil

}

type TCPStitch struct {
	stitchAddr *net.TCPAddr
}

func NewTCPStitch(addr string) (*TCPStitch, error) {
	tAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &TCPStitch{tAddr}, nil
}

func (ts *TCPStitch) Stitch(out io.Writer, in io.Reader) (string, error) {

	conn, err := net.DialTCP("tcp", nil, ts.stitchAddr)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if _, err := io.Copy(conn, in); err != nil {
		return "", fmt.Errorf("Sending to tcp stitch failed: %v", err)
	}
	errChan := make(chan error, 0)
	go func() {
		if _, err := io.Copy(out, conn); err != nil {
			errChan <- fmt.Errorf("Receiving from tcp stitch failed: %v", err)
		}
		errChan <- nil
	}()
	err = <-errChan
	return "", err
}

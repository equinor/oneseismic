package service

import (
	"bytes"
	"fmt"
	"io"
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

func NewExecStitch(cmdArgs []string, profile bool) *ExecStitch {

	return &ExecStitch{cmdArgs, profile}
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

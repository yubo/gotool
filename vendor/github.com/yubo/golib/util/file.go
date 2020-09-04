package util

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func ReadFile(url string) ([]byte, error) {
	if strings.HasPrefix(url, "http") {
		resp, err := http.Get(url)
		if err != nil {
			return []byte{}, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return []byte{}, errors.New(resp.Status)
		}
		return ioutil.ReadAll(resp.Body)
	}

	file, err := Filepath(url)
	if err != nil {
		return []byte{}, err
	}

	return ioutil.ReadFile(file)
}

func Filepath(name string) (string, error) {
	if strings.HasPrefix(name, "~/") {
		name = os.Getenv("HOME") + name[1:]
	}
	return filepath.Abs(name)
}

func ReadFileTo64(url string) (*string, error) {
	b, err := ReadFile(url)
	if err != nil {
		return nil, err
	}

	v := Base64Encode(b)
	return &v, nil
}

func WriteTempFile(dir, pattern string, content []byte) (string, error) {

	tmpfile, err := ioutil.TempFile(dir, pattern)
	if err != nil {
		return "", err
	}

	if _, err := tmpfile.Write(content); err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		return "", err
	}

	if err := tmpfile.Close(); err != nil {
		os.Remove(tmpfile.Name())
		return "", err
	}

	return tmpfile.Name(), nil
}

func CheckDir(dir string, mk bool) error {
	fi, err := os.Stat(dir)
	if err != nil {
		if mk {
			if err = os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}
		return nil
	}

	if fi.IsDir() {
		return nil
	}

	return fmt.Errorf("%s exist, but is not dir", dir)
}

func CheckCmds(cmds ...string) error {
	for _, cmd := range cmds {
		if _, err := exec.LookPath(cmd); err != nil {
			return fmt.Errorf("%s not found.", cmd)
		}
	}
	return nil
}

func CheckDirs(dirs ...string) error {
	for _, dir := range dirs {
		if err := CheckDir(dir, false); err != nil {
			return err
		}
	}
	return nil
}

func Popen(timeout time.Duration, dryRun bool, cmd string, args ...string) (string, error) {
	var err error

	if cmd, err = exec.LookPath(cmd); err != nil {
		return "", fmt.Errorf("unable to do popen: %s not found.", cmd)
	}

	cmdString := fmt.Sprintf("%s %s", cmd, strings.Join(args, " "))
	if dryRun {
		return cmdString, ErrDryRun
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, cmd, args...).CombinedOutput()
	if err != nil {
		err = fmt.Errorf("%s %s", err.Error(), out)
	}
	return string(out), err
}

func Bash(script []byte) ([]byte, error) {
	in := bytes.NewBuffer(script)
	bash := exec.Command("bash")

	stdin, _ := bash.StdinPipe()
	go func() {
		io.Copy(stdin, in)
		stdin.Close()
	}()

	return bash.CombinedOutput()
}

func Exec(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Write(w io.Writer, b []byte) error {
	n, err := w.Write(b)
	if err != nil {
		return nil
	}
	if n != len(b) {
		return io.ErrShortWrite
	}
	return nil
}

func IsDir(file string) bool {
	f, e := os.Stat(file)
	if e != nil {
		return false
	}
	return f.IsDir()
}

func IsFile(file string) bool {
	f, e := os.Stat(file)
	if e != nil {
		return false
	}
	return !f.IsDir()
}

func CleanSockFile(net, addr string) (string, string) {
	if net == "unix" && IsFile(addr) {
		os.Remove(addr)
	}
	return net, addr
}

func ReadFileInt(filename string) (int, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return 0, err
	}

	if i, err := strconv.Atoi(strings.TrimSpace(string(data))); err != nil {
		return 0, err
	} else {
		return i, nil
	}
}

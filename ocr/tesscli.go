// +build !gosseract

package ocr

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/piggy-moe/subtitle/config"
	"github.com/piggy-moe/subtitle/util"
)

const VersionTag = ""

var lock sync.Mutex
var args []string

func SetupTesseract() {
	args = []string{
		"stdin", "stdout",
		"-l", strings.Join(config.Value.Tesseract.Langs, "+"),
		"--psm", strconv.Itoa(config.Value.Tesseract.Psm),
	}
}

func StopTesseract() {
	// no-op
}

func RunTesseract(image []byte) (string, error) {
	lock.Lock()
	defer lock.Unlock()
	cmd := exec.Command("tesseract", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("unable to get stdin pipe: %w", err)
	}
	if _, err := stdin.Write(image); err != nil {
		return "", fmt.Errorf("unable to write image to stdin pipe: %w", err)
	}
	if err := stdin.Close(); err != nil {
		return "", fmt.Errorf("unable to close stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("unable to get stderr pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("unable to get stderr pipe: %w", err)
	}
	stdoutBuf := util.BufferPool.Get().(*bytes.Buffer)
	stderrBuf := util.BufferPool.Get().(*bytes.Buffer)
	stdoutBuf.Reset()
	stderrBuf.Reset()
	go io.Copy(stdoutBuf, stdout)
	go io.Copy(stderrBuf, stderr)
	if err := cmd.Run(); err != nil {
		log.Printf("error occurs while running tesseract: %s", err.Error())
		log.Print("stderr of tesseract is shown below:")
		fmt.Println(string(stderrBuf.Bytes()))
		return "", fmt.Errorf("error occurs while running tesseract: %w", err)
	}
	if stderrBuf.Len() > 0 {
		log.Print("stderr from tesseract is not empty")
		log.Print("stderr of tesseract is shown below:")
		fmt.Println(string(stderrBuf.Bytes()))
	}
	return string(stdoutBuf.Bytes()), nil
}

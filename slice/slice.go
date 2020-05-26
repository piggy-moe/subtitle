package slice

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/piggy-moe/subtitle/config"
	"github.com/piggy-moe/subtitle/util"
)

func makeArgs(ctx *cli.Context) []string {
	vf := []string{fmt.Sprintf("fps=%d/%d", config.Value.Slice.Fps, config.Value.Slice.FrameInterval)}
	vf = append(vf, config.Value.Ffmpeg.Filters...)
	args := []string{
		"-hide_banner",
		"-i", ctx.String("input"),
		"-ss", ctx.String("begin"),
		"-to", ctx.String("end"),
		"-vf", strings.Join(vf, ","),
	}
	args = append(args, config.Value.Ffmpeg.AppendArgs...)
	args = append(args, path.Join(ctx.String("dir"), "%06d."+config.Value.Slice.Format))
	return args
}

func Slice(ctx *cli.Context) error {
	dir := ctx.String("dir")
	if err := os.MkdirAll(dir, os.ModeDir|os.FileMode(0755)); err != nil {
		log.Fatal(err)
	}
	begin := ctx.String("begin")
	beginTime, err := util.ParseDuration(begin)
	if err != nil {
		log.Fatalf("unable to parse beginning time: %s", err.Error())
	}
	end := ctx.String("end")
	endTime, err := util.ParseDuration(end)
	if err != nil {
		log.Fatalf("unable to parse endding time: %s", err.Error())
	}

	progress := make(chan time.Duration, 10)
	go runFfmpeg(makeArgs(ctx), progress, beginTime, endTime)
	counter := 0
	log.Print("performing stream frame renameing")

	t := beginTime
	for curTime := range progress {
		for ; t < curTime; t += time.Second * time.Duration(config.Value.Slice.FrameInterval) {
			pathname := path.Join(dir, fmt.Sprintf("h%02dm%02d", int(t.Hours()), int(t.Minutes())%60))
			if err := os.MkdirAll(pathname, os.ModeDir|os.FileMode(0755)); err != nil {
				log.Fatal(err)
			}
			for fidB := 0; fidB < config.Value.Slice.Fps; fidB++ {
				fid := fidB * config.Value.Slice.FpsFactor
				counter++
				oldname := path.Join(dir, fmt.Sprintf("%06d.%s", counter, config.Value.Slice.Format))
				newname := path.Join(pathname, fmt.Sprintf("s%02df%02d.%s", int(t.Seconds())%60, fid, config.Value.Slice.Format))
				if err := os.Rename(oldname, newname); err != nil {
					if os.IsNotExist(err) {
						log.Printf("frame %s/%02d not exist, exitting", util.FormatDuration(t), fid)
						return nil
					}
					log.Fatal(err)
				}
			}
		}
	}
	return nil
}

func runFfmpeg(args []string, progress chan<- time.Duration, beginTime, endTime time.Duration) {
	cmd := exec.Command("ffmpeg", args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	stderrBuf := &bytes.Buffer{}
	tee := io.TeeReader(stderr, stderrBuf)
	go func(r io.Reader) {
		s := bufio.NewScanner(r)
		s.Split(bufio.ScanWords)
		for s.Scan() {
			w := s.Text()
			if strings.HasPrefix(w, "time=") {
				log.Printf("ffmpeg progress report: %s", w)
				t, err := util.ParseDuration(w[len("time="):])
				if err != nil {
					log.Printf("unrecognized progress timestamp: %q", w)
				}
				progress <- t + beginTime
			}
		}
	}(tee)
	log.Printf("starting ffmpeg with %q", args)
	if err := cmd.Run(); err != nil {
		log.Printf("error occurs while running ffmpeg: %s", err.Error())
		log.Print("stderr of ffmpeg is shown below:")
		fmt.Println(string(stderrBuf.Bytes()))
		os.Exit(1)
	}
	progress <- endTime
}

package ocr

import (
	"bytes"
	"fmt"
	"image"
	"log"
	"os"
	"path"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/piggy-moe/subtitle/binarize"
	"github.com/piggy-moe/subtitle/config"
	"github.com/piggy-moe/subtitle/util"
)

type pipelineTask struct {
	name   string
	time   time.Duration
	frame  int
	img    *image.Gray
	status string
	text   string

	idle     int64
	prevChan <-chan pipelineTask
	nextChan chan<- pipelineTask
	tokens   chan<- struct{}
	stop     chan<- struct{}
	result   chan<- pipelineTask
}

var replacer util.Replacer

func Init() {
	binarize.Init()
	replacer = util.MustNewReplacer(config.Value.Ocr.Replace)
	SetupTesseract()
}

func GetText(buf []byte) (string, error) {
	text, err := RunTesseract(buf)
	if err != nil {
		return "", err
	}
	return replacer.Replace(text), nil
}

func pipeline(task pipelineTask) {
	source, err := binarize.Load(task.name)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("frame %s/%02d not exist, exitting", util.FormatDuration(task.time), task.frame)
			task.tokens <- struct{}{}
			task.stop <- struct{}{}
			return
		}
		log.Fatal(err)
	}
	cropped := binarize.Crop(source.(binarize.SubImager))
	binaried, index1 := binarize.Binarize(cropped)
	optimized, index2 := binarize.Optimize(cropped, binaried, index1)
	task.img = binarize.Trim(optimized, index2)
	binarize.CoordPool.Put(index1)
	binarize.CoordPool.Put(index2)
	if task.img == nil {
		task.status = "EMPTY"
		task.nextChan <- task
		task.result <- task
		task.tokens <- struct{}{}
		log.Printf(fmt.Sprintf("%s/%02d (EMPTY) idle=%3dms %q\n",
			util.FormatDuration(task.time), task.frame, task.idle, task.text))
		return
	}
	task.status = "RESUL"
	idleStart := time.Now()
	prev := <-task.prevChan
	task.idle += time.Since(idleStart).Milliseconds()
	bound := cropped.Bounds()
	cacheLimit := config.Value.Ocr.Cache.Calculate(bound.Dx() * bound.Dy())
	if !config.Value.Ocr.Cache.Equal(0, 0) {
		if binarize.Difference(prev.img, task.img) <= cacheLimit {
			task.text = prev.text
			task.nextChan <- task
			task.result <- task
			task.tokens <- struct{}{}
			log.Printf(fmt.Sprintf("%s/%02d (CACHE) idle=%3dms %q\n",
				util.FormatDuration(task.time), task.frame, task.idle, task.text))
			return
		}
	}
	buf := util.BufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	if binarize.Encode(buf, task.img, config.Value.Ocr.Format, config.Value.Ocr.JpgQuality); err != nil {
		log.Print(err)
	}
	task.text, err = GetText(buf.Bytes())
	util.BufferPool.Put(buf)
	if err != nil {
		log.Printf("failed to get text from %s: %s", task.name, err.Error())
	}
	task.nextChan <- task
	task.result <- task
	task.tokens <- struct{}{}
	log.Printf(fmt.Sprintf("%s/%02d (RESUL) idle=%3dms %q\n",
		util.FormatDuration(task.time), task.frame, task.idle, task.text))
}

func Ocr(ctx *cli.Context) error {
	Init()
	dir := ctx.String("dir")
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
	outputFilename := ctx.String("output")
	outFile, err := os.Create(outputFilename)
	if err != nil {
		log.Fatalf(err.Error())
	}

	result := make(chan pipelineTask)
	done := make(chan struct{})
	go writeResult(beginTime, outFile, result, done)

	concurrency := ctx.Int("concurrency")
	token := make(chan struct{}, concurrency)
	stop := make(chan struct{})
	for i := 0; i < concurrency; i++ {
		token <- struct{}{}
	}

	var prevChan, nextChan chan pipelineTask
	nextChan = make(chan pipelineTask, 1)
	nextChan <- pipelineTask{}
	for t := beginTime; t < endTime; t += time.Second * time.Duration(config.Value.Slice.FrameInterval) {
		pathname := path.Join(dir, fmt.Sprintf("h%02dm%02d", int(t.Hours()), int(t.Minutes())%60))
		for fidB := 0; fidB < config.Value.Slice.Fps; fidB++ {
			fid := fidB * config.Value.Slice.FpsFactor

			prevChan = nextChan
			nextChan = make(chan pipelineTask, 1)

			idleStart := time.Now()
			select {
			case <-stop:
				goto finish
			case <-token:
			}

			filename := fmt.Sprintf("s%02df%02d.%s", int(t.Seconds())%60, fid, config.Value.Slice.Format)
			go pipeline(pipelineTask{
				name:     path.Join(pathname, filename),
				time:     t,
				frame:    fid,
				idle:     time.Since(idleStart).Milliseconds(),
				prevChan: prevChan,
				nextChan: nextChan,
				tokens:   token,
				stop:     stop,
				result:   result,
			})

		}
	}
finish:
	for i := 0; i < concurrency; i++ {
		<-token
	}
	close(result)
	<-done
	StopTesseract()
	return nil
}

func writeResult(begin time.Duration, file *os.File, ch <-chan pipelineTask, done chan<- struct{}) {
	initted := false
	start, last := pipelineTask{}, pipelineTask{}

	buf := make(map[int]pipelineTask)
	counter := 0

	expectTime := begin
	expectFrame := 0

	for recevied := range ch {
		buf[counter] = recevied
		counter++

		found := true
		for found {
			found = false
			for index, item := range buf {
				if item.time == expectTime && item.frame == expectFrame {
					found = true
					delete(buf, index)
					expectFrame += config.Value.Slice.FpsFactor
					if expectFrame == config.Value.Slice.Fps*config.Value.Slice.FpsFactor {
						expectTime += time.Second * time.Duration(config.Value.Slice.FrameInterval)
						expectFrame = 0
					}

					if start.text != item.text || start.status != item.status {
						if !initted {
							initted = true
						} else if len(start.text) > 0 {
							fmt.Fprintf(file, "%s/%02d->%s/%02d %q\n",
								util.FormatDuration(start.time), start.frame,
								util.FormatDuration(last.time), last.frame,
								start.text,
							)
						}
						start = item
					}
					last = item
					last.frame += config.Value.Slice.FpsFactor
					if last.frame == config.Value.Slice.Fps*config.Value.Slice.FpsFactor {
						last.time += time.Second * time.Duration(config.Value.Slice.FrameInterval)
						last.frame = 0
					}
					break
				}
			}
		}

	}

	if len(buf) > 0 {
		panic("still item in buf")
	}

	if initted && len(start.text) > 0 {
		fmt.Fprintf(file, "%s/%02d->%s/%02d %q\n",
			util.FormatDuration(start.time), start.frame,
			util.FormatDuration(last.time), last.frame,
			start.text,
		)
	}
	done <- struct{}{}
}

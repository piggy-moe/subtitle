package conv

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/piggy-moe/subtitle/config"
	"github.com/piggy-moe/subtitle/util"
)

type subtitleItem struct {
	t1, t2 time.Duration
	f1, f2 int
	text   string
}

func Convert(ctx *cli.Context) error {
	input, err := os.Open(ctx.String("input"))
	if err != nil {
		log.Fatal(err)
	}
	defer input.Close()
	output, err := os.Create(ctx.String("output"))
	if err != nil {
		log.Fatal(err)
	}
	defer output.Close()
	ch := make(chan subtitleItem)
	format, ok := formatter[config.Value.Convert.Format]
	if !ok {
		log.Fatalf("unsupported format %q", config.Value.Convert.Format)
	}
	go func(input *os.File) {
		var err error
		replacer := util.MustNewReplacer(config.Value.Convert.Replace)
		scanner := bufio.NewScanner(input)
		lineNum := 0
		p := subtitleItem{}
		for scanner.Scan() {
			lineNum++
			l := scanner.Text()
			x := subtitleItem{}
			var s1, s2 string
			if _, err = fmt.Sscanf(l, "%8s/%02d->%8s/%02d %q", &s1, &x.f1, &s2, &x.f2, &x.text); err != nil {
				log.Fatalf("failed to parse line %d: %s", lineNum, err.Error())
			}
			if x.t1, err = util.ParseDuration(s1); err != nil {
				log.Fatalf("failed to parse line %d: %s", lineNum, err.Error())
			}
			if x.t2, err = util.ParseDuration(s2); err != nil {
				log.Fatalf("failed to parse line %d: %s", lineNum, err.Error())
			}
			x.text = replacer.Replace(x.text)
			if util.Silimar(p.text, x.text, config.Value.Convert.Merge) && p.t2 == x.t1 && p.f2 == x.f1 {
				p.t2 = x.t2
				p.f2 = x.f2
			} else {
				if len(p.text) > 0 {
					ch <- p
				}
				p = x
			}
		}
		if len(p.text) > 0 {
			ch <- p
		}
		close(ch)
	}(input)
	format(output, ch)
	return nil
}

var formatter = map[string]func(*os.File, <-chan subtitleItem){
	"raw": func(w *os.File, ch <-chan subtitleItem) {
		for x := range ch {
			fmt.Fprintf(w, "%s/%02d->%s/%02d %q\n",
				util.FormatDuration(x.t1), x.f1,
				util.FormatDuration(x.t2), x.f2,
				x.text,
			)
		}
	},
	"srt": func(w *os.File, ch <-chan subtitleItem) {
		id := 0
		r := 1000.0 / float64(config.Value.Slice.Fps*config.Value.Slice.FpsFactor)
		for x := range ch {
			id++
			fmt.Fprintf(w, "%d\n", id)
			fmt.Fprintf(w, "%s,%03d --> %s,%03d\n",
				util.FormatDuration(x.t1), int(r*float64(x.f1)),
				util.FormatDuration(x.t2), int(r*float64(x.f2)),
			)
			fmt.Fprintf(w, "%s\n\n", x.text)
		}
	},
	"lrc": func(w *os.File, ch <-chan subtitleItem) {
		r := 1000.0 / float64(config.Value.Slice.Fps*config.Value.Slice.FpsFactor)
		for x := range ch {
			fmt.Fprintf(w, "[%s.%02d]%s\n", util.FormatDuration(x.t1), int(r*float64(x.f1)), x.text)
		}
	},
	"plain": func(w *os.File, ch <-chan subtitleItem) {
		for x := range ch {
			fmt.Fprintln(w, x.text)
		}
	},
}

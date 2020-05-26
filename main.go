package main

import (
	"log"
	"os"
	"reflect"

	"github.com/urfave/cli/v2"

	"github.com/piggy-moe/subtitle/check"
	"github.com/piggy-moe/subtitle/config"
	"github.com/piggy-moe/subtitle/conv"
	"github.com/piggy-moe/subtitle/ocr"
	"github.com/piggy-moe/subtitle/slice"
	"github.com/piggy-moe/subtitle/util"
)

const Version = "v1.0.0"

var sharedFlags = map[string]cli.Flag{
	"cpuprof": &cli.StringFlag{
		Name:    "cpuprof",
		Aliases: []string{"P"},
		Usage:   "save CPU profile to `FILE`",
	},
	"memprof": &cli.StringFlag{
		Name:    "memprof",
		Aliases: []string{"M"},
		Usage:   "save mempry profile to `FILE`",
	},
	"config": &cli.StringFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Value:   "subtitle.json",
		Usage:   "read configuration from `CONFIG`",
	},
	"input": &cli.StringFlag{
		Name:     "input",
		Aliases:  []string{"i"},
		Required: true,
		Usage:    "specify `VIDEO` as input video (required)",
	},
	"dir": &cli.StringFlag{
		Name:     "dir",
		Aliases:  []string{"d"},
		Required: true,
		Usage:    "use `DIR` to store frames (required)",
	},
	"begin": &cli.StringFlag{
		Name:    "begin",
		Aliases: []string{"s"},
		Value:   "00:00:00",
		Usage:   "specify `TIME` of beginning, formatted in hh:mm:ss",
	},
	"end": &cli.StringFlag{
		Name:        "end",
		Aliases:     []string{"t"},
		Value:       "99:59:59",
		DefaultText: "auto detect",
		Usage:       "specify `TIME` of ending, formatted in hh:mm:ss",
	},
	"output": &cli.StringFlag{
		Name:     "output",
		Aliases:  []string{"o"},
		Required: true,
		Usage:    "save OCR results to `FILE` (required)",
	},
	"concurrency": &cli.IntFlag{
		Name:    "concurrency",
		Aliases: []string{"j"},
		Value:   1,
		Usage:   "use `X` workers in OCR",
	},
}

func overwrite(f cli.Flag, fields map[string]interface{}) cli.Flag {
	flag := reflect.ValueOf(f).Elem()
	newFlag := reflect.New(flag.Type()).Elem()
	newFlag.Set(flag)
	for k, v := range fields {
		newFlag.FieldByName(k).Set(reflect.ValueOf(v))
	}
	return newFlag.Addr().Interface().(cli.Flag)
}

func main() {
	app := &cli.App{
		Name:                   "subtitle",
		Usage:                  "A tool for extracting subtitles from videos with hard-coded subtitles",
		Version:                Version + ocr.VersionTag,
		HideHelp:               true,
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			sharedFlags["cpuprof"],
			sharedFlags["memprof"],
		},
		Before: util.StartProfile,
		Action: cli.ShowAppHelp,
		After:  util.StopProfile,
		Commands: []*cli.Command{
			&cli.Command{
				Name:  "new",
				Usage: "generate the default configuration file",
				Flags: []cli.Flag{
					overwrite(sharedFlags["config"], map[string]interface{}{
						"Usage": "save default configuration to `CONFIG`",
					}),
				},
				Before: config.Reset,
				Action: config.Save,
			},
			&cli.Command{
				Name:  "slice",
				Usage: "slice video into frames",
				Flags: []cli.Flag{
					sharedFlags["config"],
					sharedFlags["input"],
					sharedFlags["dir"],
					sharedFlags["begin"],
					sharedFlags["end"],
				},
				Before: config.Load,
				Action: slice.Slice,
			},
			&cli.Command{
				Name:  "check",
				Usage: "check configuration for best extraction quality",
				Flags: []cli.Flag{
					sharedFlags["config"],
					overwrite(sharedFlags["input"], map[string]interface{}{
						"Usage": "use `IMAGE` as sample frame for testing (required)",
					}),
					overwrite(sharedFlags["output"], map[string]interface{}{
						"Required": false,
						"Usage":    "save debugging image to `FILE`",
					}),
				},
				Before: config.Load,
				Action: check.Check,
			},
			&cli.Command{
				Name:  "ocr",
				Usage: "run OCR to extract subtitles from frames",
				Flags: []cli.Flag{
					sharedFlags["config"],
					sharedFlags["dir"],
					sharedFlags["begin"],
					sharedFlags["end"],
					sharedFlags["output"],
					sharedFlags["concurrency"],
				},
				Before: config.Load,
				Action: ocr.Ocr,
			},
			&cli.Command{
				Name:  "conv",
				Usage: "convert OCR results to subtitle file",
				Flags: []cli.Flag{
					sharedFlags["config"],
					overwrite(sharedFlags["input"], map[string]interface{}{
						"Usage": "read OCR results from `FILE` (required)",
					}),
					overwrite(sharedFlags["output"], map[string]interface{}{
						"Usage": "save formatted subtitles to `FILE` (required)",
					}),
				},
				Before: config.Load,
				Action: conv.Convert,
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

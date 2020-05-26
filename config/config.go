package config

import (
	"encoding/json"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

type Config struct {
	Ffmpeg    FfmpegConfig    `json:"ffmpeg"`
	Tesseract TesseractConfig `json:"tesseract"`
	Slice     SliceConfig     `json:"slice"`
	Binarize  BinarizeConfig  `json:"binarize"`
	Check     CheckConfig     `json:"check"`
	Ocr       OcrConfig       `json:"ocr"`
	Convert   ConvertConfig   `json:"convert"`
}

type FfmpegConfig struct {
	Filters    []string `json:"filters"`
	AppendArgs []string `json:"appendArgs"`
}

type TesseractConfig struct {
	Langs []string `json:"langs"`
	Psm   int      `json:"psm"`
}

type SliceConfig struct {
	Fps           int    `json:"fps"`
	FpsFactor     int    `json:"fpsFactor"`
	FrameInterval int    `json:"frameInterval"`
	Format        string `json:"format"`
}

type BinarizeConfig struct {
	Crop       Area            `json:"crop"`
	TextColors []ColorGroup    `json:"textColors"`
	Optitmizer OptimizerConfig `json:"optimizer"`
}

type OptimizerConfig struct {
	Connectivity int            `json:"connectivity"`
	Size         Range          `json:"size"`
	Width        Range          `json:"width"`
	Height       Range          `json:"height"`
	Border       BorderConfig   `json:"border"`
	NoOnEdge     NoOnEdgeConfig `json:"noOnEdge"`
}

type BorderConfig struct {
	Color []ColorGroup  `json:"colors"`
	Level RelativeValue `json:"level"`
}

type NoOnEdgeConfig struct {
	Left   bool `json:"left"`
	Right  bool `json:"right"`
	Top    bool `json:"top"`
	Bottom bool `json:"bottom"`
}

type CheckConfig struct {
	MaskLevel  float64    `json:"maskLevel"`
	Cropped    ColorGroup `json:"cropped"`
	Background ColorGroup `json:"background"`
	Text       ColorGroup `json:"text"`
	Discarded  ColorGroup `json:"discarded"`
}

type OcrConfig struct {
	Cache      RelativeValue `json:"cache"`
	Margin     MarginConfig  `json:"margin"`
	Format     string        `json:"format"`
	JpgQuality int           `json:"jpgQuality"`
	Replace    []Replace     `json:"replace"`
}

type MarginConfig struct {
	X RelativeValue `json:"x"`
	Y RelativeValue `json:"y"`
}

type Replace struct {
	Regexp bool   `json:"regexp"`
	From   string `json:"from"`
	To     string `json:"to"`
}

type ConvertConfig struct {
	Replace []Replace     `json:"replace"`
	Format  string        `json:"format"`
	Merge   RelativeValue `json:"merge"`
}

var Value Config

func Load(ctx *cli.Context) error {
	file, err := os.Open(ctx.String("config"))
	if err != nil {
		log.Fatal(err)
	}
	if err := json.NewDecoder(file).Decode(&Value); err != nil {
		log.Fatal(err)
	}
	file.Close()
	return nil
}

func Save(ctx *cli.Context) error {
	file, err := os.Create(ctx.String("config"))
	if err != nil {
		log.Fatal(err)
	}
	enc := json.NewEncoder(file)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(Value); err != nil {
		log.Fatal(err)
	}
	file.Close()
	return nil
}

func Reset(*cli.Context) error {
	Value = Config{
		Ffmpeg: FfmpegConfig{
			Filters:    []string{},
			AppendArgs: []string{},
		},
		Tesseract: TesseractConfig{
			Langs: []string{"eng"},
			Psm:   3,
		},
		Slice: SliceConfig{
			Fps:           1,
			FpsFactor:     1,
			FrameInterval: 1,
			Format:        "jpg",
		},
		Binarize: BinarizeConfig{
			Crop: Area{
				Left:   MustNewRelativeValue("0%+0"),
				Right:  MustNewRelativeValue("100%+0"),
				Top:    MustNewRelativeValue("0%+0"),
				Bottom: MustNewRelativeValue("100%+0"),
			},
			TextColors: []ColorGroup{},
			Optitmizer: OptimizerConfig{
				Connectivity: 8,
				Size:         MustNewRange("0%+0", "100%+0"),
				Width:        MustNewRange("0%+0", "100%+0"),
				Height:       MustNewRange("0%+0", "100%+0"),
				Border: BorderConfig{
					Color: []ColorGroup{},
					Level: MustNewRelativeValue("0%+0"),
				},
				NoOnEdge: NoOnEdgeConfig{
					Left:   false,
					Right:  false,
					Top:    false,
					Bottom: false,
				},
			},
		},
		Check: CheckConfig{
			MaskLevel:  0.8,
			Cropped:    MustNewColorGroup("#7f7f7f"),
			Background: MustNewColorGroup("#000000"),
			Discarded:  MustNewColorGroup("#0000ff"),
			Text:       MustNewColorGroup("#ff0000"),
		},
		Ocr: OcrConfig{
			Cache: MustNewRelativeValue("0%+0"),
			Margin: MarginConfig{
				X: MustNewRelativeValue("0%+20"),
				Y: MustNewRelativeValue("0%+20"),
			},
			Format:     "jpg",
			JpgQuality: 100,
			Replace: []Replace{
				Replace{
					Regexp: false,
					From:   "\n",
					To:     " ",
				},
			},
		},
		Convert: ConvertConfig{
			Replace: []Replace{},
			Merge:   MustNewRelativeValue("0%+0"),
			Format:  "srt",
		},
	}
	return nil
}

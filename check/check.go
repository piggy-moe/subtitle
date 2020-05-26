package check

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"log"

	"github.com/urfave/cli/v2"

	"github.com/piggy-moe/subtitle/binarize"
	"github.com/piggy-moe/subtitle/config"
	"github.com/piggy-moe/subtitle/ocr"
	"github.com/piggy-moe/subtitle/util"
)

func Check(ctx *cli.Context) error {
	filename := ctx.String("input")
	binarize.Init()
	source, err := binarize.Load(filename)
	if err != nil {
		log.Fatal(err)
	}
	bounds := source.Bounds()
	cropped := binarize.Crop(source.(binarize.SubImager))
	binaried, index1 := binarize.Binarize(cropped)
	optimized, index2 := binarize.Optimize(cropped, binaried, index1)
	trimed := binarize.Trim(optimized, index2)
	if ctx.IsSet("output") {
		mask := image.NewRGBA(bounds)
		paintMask(mask, binaried, optimized)
		output := renderOutput(source, mask)
		binarize.Save(ctx.String("output"), output, config.Value.Ocr.Format, config.Value.Ocr.JpgQuality)
	}
	if trimed == nil {
		log.Print("no text detected")
	} else {
		buf := util.BufferPool.Get().(*bytes.Buffer)
		buf.Reset()
		if err := binarize.Encode(buf, trimed, config.Value.Ocr.Format, config.Value.Ocr.JpgQuality); err != nil {
			log.Print(err)
		}
		ocr.Init()
		result, err := ocr.GetText(buf.Bytes())
		if err != nil {
			log.Fatal(err)
		}
		util.BufferPool.Put(buf)
		fmt.Println(result)
		ocr.StopTesseract()
	}
	return nil
}

func paintMask(mask *image.RGBA, bin, opt *image.Gray) {
	b := mask.Bounds()
	minX := b.Min.X + config.Value.Binarize.Crop.Left.Calculate(b.Dx())
	maxX := b.Min.X + config.Value.Binarize.Crop.Right.Calculate(b.Dx())
	minY := b.Min.Y + config.Value.Binarize.Crop.Top.Calculate(b.Dy())
	maxY := b.Min.Y + config.Value.Binarize.Crop.Bottom.Calculate(b.Dy())
	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			if x <= minX || x >= maxX || y <= minY || y >= maxY {
				mask.Set(x, y, config.Value.Check.Cropped.Color)
			} else if bin.GrayAt(x, y).Y == 0 && opt.GrayAt(x, y).Y == 255 {
				mask.Set(x, y, config.Value.Check.Discarded.Color)
			} else if opt.GrayAt(x, y).Y == 0 {
				mask.Set(x, y, config.Value.Check.Text.Color)
			} else {
				mask.Set(x, y, config.Value.Check.Background.Color)
			}
		}
	}
}

func renderOutput(source, mask image.Image) image.Image {
	b := source.Bounds()
	output := image.NewRGBA(b)
	f := config.Value.Check.MaskLevel
	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			r1, g1, b1, _ := mask.At(x, y).RGBA()
			r2, g2, b2, _ := source.At(x, y).RGBA()
			output.Set(x, y, color.RGBA{
				R: uint8((float64(r1)*f + float64(r2)*(1-f)) / 0x100),
				G: uint8((float64(g1)*f + float64(g2)*(1-f)) / 0x100),
				B: uint8((float64(b1)*f + float64(b2)*(1-f)) / 0x100),
			})
		}
	}
	return output
}

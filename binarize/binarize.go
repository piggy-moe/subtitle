package binarize

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"sync"

	"github.com/piggy-moe/subtitle/config"
	"github.com/piggy-moe/subtitle/util"
)

type Coordinate struct {
	X, Y int
}

type SubImager interface {
	image.Image
	SubImage(image.Rectangle) image.Image
}

const (
	blackValue = 0
	whiteValue = 255
)

var (
	black = color.Gray{blackValue}
	white = color.Gray{whiteValue}
)

var CoordPool = sync.Pool{
	New: func() interface{} {
		return []Coordinate{}
	},
}

func min(x, y int) int {
	if x <= y {
		return x
	}
	return y
}

func max(x, y int) int {
	if x >= y {
		return x
	}
	return y
}

func Init() {
	switch config.Value.Binarize.Optitmizer.Connectivity {
	case 8:
		// no-op
	case 4:
		directions = directions[0:4]
	default:
		log.Fatalf("unsupported pixel connectivity: %d", config.Value.Binarize.Optitmizer.Connectivity)
	}
}

func Load(name string) (image.Image, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(file)
	if err != nil {
		log.Print(err)
	}
	if err != nil {
		return nil, err
	}
	file.Close()
	return img, nil
}

func Encode(w io.Writer, img image.Image, format string, jpgQuality int) error {
	switch format {
	case "jpg":
		return jpeg.Encode(w, img, &jpeg.Options{Quality: jpgQuality})
	case "png":
		return png.Encode(w, img)
	default:
		log.Fatalf("unsupported image format %q", config.Value.Slice.Format)
		panic(nil) // unreachable
	}
}

func Save(name string, img image.Image, format string, jpgQuality int) error {
	file, err := os.Create(name)
	if err != nil {
		return fmt.Errorf("unable to create file: %s", err.Error())
	}
	if err := Encode(file, img, format, jpgQuality); err != nil {
		return fmt.Errorf("failed to encode image: %s", err.Error())
	}
	file.Close()
	return nil
}

func Crop(img SubImager) image.Image {
	b := img.Bounds()
	return img.SubImage(image.Rectangle{
		Min: image.Point{
			X: b.Min.X + config.Value.Binarize.Crop.Left.Calculate(b.Dx()),
			Y: b.Min.Y + config.Value.Binarize.Crop.Top.Calculate(b.Dy()),
		},
		Max: image.Point{
			X: b.Min.X + config.Value.Binarize.Crop.Right.Calculate(b.Dx()),
			Y: b.Min.Y + config.Value.Binarize.Crop.Bottom.Calculate(b.Dy()),
		},
	})
}

func Binarize(img image.Image) (*image.Gray, []Coordinate) {
	b := img.Bounds()
	imgNew := image.NewGray(b)
	index := CoordPool.Get().([]Coordinate)[:0]
	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			matched := false
			tcol := rgb(img.At(x, y))
			for _, cg := range config.Value.Binarize.TextColors {
				if cg.Contains(tcol) {
					matched = true
					break
				}
			}
			if matched {
				imgNew.SetGray(x, y, black)
				index = append(index, Coordinate{x, y})
			} else {
				imgNew.SetGray(x, y, white)
			}
		}
	}
	return imgNew, index
}

func Optimize(source image.Image, img *image.Gray, index []Coordinate) (*image.Gray, []Coordinate) {
	bound := img.Bounds()
	minS := config.Value.Binarize.Optitmizer.Size.Min.Calculate(bound.Dx() * bound.Dy())
	maxS := config.Value.Binarize.Optitmizer.Size.Max.Calculate(bound.Dx() * bound.Dy())
	minW := config.Value.Binarize.Optitmizer.Width.Min.Calculate(bound.Dx())
	maxW := config.Value.Binarize.Optitmizer.Width.Max.Calculate(bound.Dx())
	minH := config.Value.Binarize.Optitmizer.Height.Min.Calculate(bound.Dy())
	maxH := config.Value.Binarize.Optitmizer.Height.Max.Calculate(bound.Dy())
	imgNew := image.NewGray(bound)
	for x := bound.Min.X; x < bound.Max.X; x++ {
		for y := bound.Min.Y; y < bound.Max.Y; y++ {
			imgNew.SetGray(x, y, white)
		}
	}
	indexNew := CoordPool.Get().([]Coordinate)[:0]
	subindex := CoordPool.Get().([]Coordinate)[:0]
	length := bound.Dx() * bound.Dy()
	visited := util.BoolSlicePool.Get().([]bool)[:0]
	if cap(visited) < length {
		visited = append(visited[:cap(visited)], make([]bool, length-cap(visited))...)
	} else {
		visited = visited[:length]
	}
	for i := range visited {
		visited[i] = false
	}
	for _, c := range index {
		if !visited[img.PixOffset(c.X, c.Y)/1] {
			size := 0
			minX, maxX, minY, maxY := math.MaxInt32, math.MinInt32, math.MaxInt32, math.MinInt32
			b, bTotal := 0, 0
			subindex = subindex[:0]
			callback := func(x, y int) {
				size++
				minX = min(minX, x)
				maxX = max(maxX, x)
				minY = min(minY, y)
				maxY = max(maxY, y)
				subindex = append(subindex, Coordinate{x, y})
			}
			onBorder := func(x, y int) {
				tcol := rgb(source.At(x, y))
				matched := false
				for _, cg := range config.Value.Binarize.Optitmizer.Border.Color {
					if cg.Contains(tcol) {
						matched = true
						break
					}
				}
				bTotal++
				if matched {
					b++
				}
			}
			bfs(img, visited, c, callback, onBorder)
			minB := config.Value.Binarize.Optitmizer.Border.Level.Calculate(bTotal)
			w := maxX - minX + 1
			h := maxY - minY + 1
			discard := size < minS || size > maxS || w < minW || w > maxW || h < minH || h > maxH || b < minB
			discard = discard || (config.Value.Binarize.Optitmizer.NoOnEdge.Left && minX == bound.Min.X)
			discard = discard || (config.Value.Binarize.Optitmizer.NoOnEdge.Right && maxX == bound.Max.X-1)
			discard = discard || (config.Value.Binarize.Optitmizer.NoOnEdge.Top && minY == bound.Min.Y)
			discard = discard || (config.Value.Binarize.Optitmizer.NoOnEdge.Bottom && maxY == bound.Max.Y-1)
			if discard {
				for _, sc := range subindex {
					imgNew.SetGray(sc.X, sc.Y, white)
				}
			} else {
				for _, sc := range subindex {
					imgNew.SetGray(sc.X, sc.Y, black)
				}
				indexNew = append(indexNew, subindex...)
			}
		}
	}
	CoordPool.Put(subindex)
	util.BoolSlicePool.Put(visited)
	return imgNew, indexNew
}

func Trim(img *image.Gray, index []Coordinate) *image.Gray {
	minX := math.MaxInt32
	maxX := math.MinInt32
	minY := math.MaxInt32
	maxY := math.MinInt32
	for _, c := range index {
		minX = min(minX, c.X)
		maxX = max(maxX, c.X)
		minY = min(minY, c.Y)
		maxY = max(maxY, c.Y)
	}
	if minX == math.MaxInt32 {
		return nil
	}
	dx := config.Value.Ocr.Margin.X.Calculate(maxX - minX + 1)
	dy := config.Value.Ocr.Margin.Y.Calculate(maxY - minY + 1)

	b := img.Bounds()
	imgNew := image.NewGray(image.Rect(minX-dx, minY-dy, maxX+dx+1, maxY+dy+1))
	for x := minX - dx; x <= maxX+dx; x++ {
		for y := minY - dy; y <= maxY+dy; y++ {
			if (image.Point{x, y}).In(b) == false {
				imgNew.SetGray(x, y, white)
			} else {
				imgNew.SetGray(x, y, img.GrayAt(x, y))
			}
		}
	}
	return imgNew
}

func Difference(img1, img2 *image.Gray) int {
	if img1 == nil || img2 == nil {
		return math.MaxInt32
	}
	b1 := img1.Bounds()
	b2 := img2.Bounds()
	bU := b1.Union(b2)
	result := 0
	for x := bU.Min.X; x < bU.Max.X; x++ {
		for y := bU.Min.Y; y < bU.Max.Y; y++ {
			p := image.Point{x, y}
			if p.In(b1) {
				if p.In(b2) {
					if img1.GrayAt(x, y).Y != img2.GrayAt(x, y).Y {
						result++
					}
				} else { // p.In(b2) == false
					if img1.GrayAt(x, y).Y == blackValue {
						result++
					}
				}
			} else { // p.In(b1) == false && p.In(b2)
				if img2.GrayAt(x, y).Y == blackValue {
					result++
				}
			}
		}
	}
	return result
}

func rgb(c color.Color) color.RGBA {
	tr, tg, tb, _ := c.RGBA()
	return color.RGBA{
		R: uint8(tr / 0x100),
		G: uint8(tg / 0x100),
		B: uint8(tb / 0x100),
	}
}

var directions = []Coordinate{
	{1, 0},
	{0, 1},
	{-1, 0},
	{0, -1},
	{1, 1},
	{-1, 1},
	{-1, -1},
	{1, -1},
}

func bfs(img *image.Gray, vis []bool, start Coordinate, cb, onBorder func(x, y int)) {
	bound := img.Bounds()
	q := []Coordinate{start}
	vis[img.PixOffset(start.X, start.Y)/1] = true
	for len(q) > 0 {
		h := q[len(q)-1]
		q = q[:len(q)-1]
		cb(h.X, h.Y)
		for _, d := range directions {
			xx := h.X + d.X
			yy := h.Y + d.Y
			coord := Coordinate{xx, yy}
			off := img.PixOffset(coord.X, coord.Y) / 1
			if (image.Point{xx, yy}).In(bound) && !vis[off] {
				vis[off] = true
				if img.GrayAt(xx, yy).Y == blackValue {
					q = append(q, coord)
				} else {
					onBorder(xx, yy)
				}
			}
		}
	}
}

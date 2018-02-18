package main

import (
	"flag"
	"image"
	"image/color/palette"
	"image/gif"
	"log"
	"math"
	"os"
	"sync"

	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"

	_ "image/jpeg"
	_ "image/png"
)

func matMul(p, q *f64.Aff3) f64.Aff3 {
	return f64.Aff3{
		p[3*0+0]*q[3*0+0] + p[3*0+1]*q[3*1+0],
		p[3*0+0]*q[3*0+1] + p[3*0+1]*q[3*1+1],
		p[3*0+0]*q[3*0+2] + p[3*0+1]*q[3*1+2] + p[3*0+2],
		p[3*1+0]*q[3*0+0] + p[3*1+1]*q[3*1+0],
		p[3*1+0]*q[3*0+1] + p[3*1+1]*q[3*1+1],
		p[3*1+0]*q[3*0+2] + p[3*1+1]*q[3*1+2] + p[3*1+2],
	}
}

func rotate(dst draw.Image, src image.Image, angle float64, op draw.Op) {
	db := dst.Bounds()
	sb := src.Bounds()
	dx := float64(db.Min.X) + float64(db.Dx())/2
	dy := float64(db.Min.Y) + float64(db.Dy())/2
	sx := float64(sb.Min.X) + float64(sb.Dx())/2
	sy := float64(sb.Min.Y) + float64(sb.Dy())/2
	s, c := math.Sincos(angle)

	s2d := f64.Aff3{1, 0, sx, 0, 1, sy}
	s2d = matMul(&s2d, &f64.Aff3{c, -s, 0, s, c, 0})
	s2d = matMul(&s2d, &f64.Aff3{1, 0, -dx, 0, 1, -dy})
	draw.BiLinear.Transform(dst, s2d, src, sb, op, nil)
}

func openImage(name string) (image.Image, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

func createGIF(name string, img *gif.GIF) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return gif.EncodeAll(f, img)
}

func rotatePaletted(src image.Image, sb image.Rectangle, backfill image.Image, angle float64) *image.Paletted {
	rotated := image.NewRGBA(sb)
	draw.Copy(rotated, image.Point{}, backfill, rotated.Bounds(), draw.Src, nil)
	rotate(rotated, src, angle, draw.Src)
	dst := image.NewPaletted(sb, palette.WebSafe)
	draw.FloydSteinberg.Draw(dst, sb, rotated, sb.Min)
	return dst
}

func generateFrames(src image.Image, steps int) []*image.Paletted {
	bounds := src.Bounds()
	backfill := image.NewUniform(src.At(bounds.Min.X, bounds.Min.Y))
	stepAngle := 2 * math.Pi / float64(steps)

	frames := make([]*image.Paletted, steps)
	var wg sync.WaitGroup
	wg.Add(steps)
	for i := 0; i < steps; i++ {
		go func(i int) {
			defer wg.Done()
			log.Println("Frame", i+1, "starting")
			frames[i] = rotatePaletted(src, bounds, backfill, stepAngle*float64(i))
			log.Println("Frame", i+1, "done")
		}(i)
	}
	wg.Wait()
	return frames
}

func main() {
	var steps, delay int
	flag.IntVar(&steps, "steps", 25, "Number of steps to complete a full rotation")
	flag.IntVar(&delay, "delay", 1, "Amount of delay between steps in 100ths of a second")
	flag.Parse()

	original, err := openImage(flag.Arg(0))
	if err != nil {
		log.Fatal("Cannot load image: ", err)
	}
	frames := generateFrames(original, steps)
	delays := make([]int, steps)
	for i := range delays {
		delays[i] = delay
	}

	log.Println("Writing to", flag.Arg(1))
	if err := createGIF(flag.Arg(1), &gif.GIF{Image: frames, Delay: delays}); err != nil {
		log.Fatal("Cannot create image: ", err)
	}
}

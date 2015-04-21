package main

import (
	"flag"
	"image"
	"image/color/palette"
	"image/gif"
	"log"
	"math"
	"os"

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

func rotate(dst draw.Image, src image.Image, angle float64) {
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
	draw.BiLinear.Transform(dst, &s2d, src, sb, nil)
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

func main() {
	var steps, delay int
	flag.IntVar(&steps, "steps", 25, "Number of steps to complete a full rotation")
	flag.IntVar(&delay, "delay", 1, "Amount of delay between steps in 100ths of a second")
	flag.Parse()

	stepAngle := 2 * math.Pi / float64(steps)

	var frames []*image.Paletted
	var delays []int

	original, err := openImage(flag.Arg(0))
	if err != nil {
		log.Fatal("Cannot load image: ", err)
	}

	bounds := original.Bounds()
	backfill := image.NewUniform(original.At(bounds.Min.X, bounds.Min.Y))
	drawer := draw.FloydSteinberg

	for i := 0; i < steps; i++ {
		log.Println("Frame", i+1)
		rotated := image.NewRGBA(bounds)
		draw.Copy(rotated, image.Point{}, backfill, rotated.Bounds(), nil)
		rotate(rotated, original, float64(i)*stepAngle)
		frame := image.NewPaletted(bounds, palette.WebSafe)
		drawer.Draw(frame, bounds, rotated, bounds.Min)
		frames = append(frames, frame)
		delays = append(delays, delay)
	}

	if err := createGIF(flag.Arg(1), &gif.GIF{Image: frames, Delay: delays}); err != nil {
		log.Fatal("Cannot create image: ", err)
	}
}

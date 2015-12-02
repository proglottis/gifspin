package main

import (
	"container/heap"
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

type frame struct {
	i     int
	image *image.Paletted
}

type frameHeap []frame

func (h frameHeap) Len() int           { return len(h) }
func (h frameHeap) Less(i, j int) bool { return h[i].i < h[j].i }
func (h frameHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *frameHeap) Push(x interface{}) {
	*h = append(*h, x.(frame))
}

func (h *frameHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func rotatePaletted(src image.Image, sb image.Rectangle, backfill image.Image, angle float64) *image.Paletted {
	rotated := image.NewRGBA(sb)
	draw.Copy(rotated, image.Point{}, backfill, rotated.Bounds(), draw.Src, nil)
	rotate(rotated, src, angle, draw.Src)
	dst := image.NewPaletted(sb, palette.WebSafe)
	draw.FloydSteinberg.Draw(dst, sb, rotated, sb.Min)
	return dst
}

func generateFrames(src image.Image, steps int) <-chan frame {
	bounds := src.Bounds()
	backfill := image.NewUniform(src.At(bounds.Min.X, bounds.Min.Y))
	stepAngle := 2 * math.Pi / float64(steps)

	c := make(chan frame)
	var wg sync.WaitGroup
	wg.Add(steps)
	for i := 0; i < steps; i++ {
		go func(i int) {
			log.Println("Frame", i+1, "starting")
			dst := rotatePaletted(src, bounds, backfill, stepAngle*float64(i))
			log.Println("Frame", i+1, "done")
			c <- frame{
				i:     i,
				image: dst,
			}
			wg.Done()
		}(i)
	}
	go func() {
		wg.Wait()
		close(c)
	}()
	return c
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

	h := &frameHeap{}
	heap.Init(h)
	for f := range generateFrames(original, steps) {
		heap.Push(h, f)
		log.Println("Frame", f.i+1, "queued")
	}
	var frames []*image.Paletted
	var delays []int
	for h.Len() > 0 {
		f := heap.Pop(h).(frame)
		log.Println("Frame", f.i+1, "dequeued")
		frames = append(frames, f.image)
		delays = append(delays, delay)
	}

	log.Println("Writing to", flag.Arg(1))
	if err := createGIF(flag.Arg(1), &gif.GIF{Image: frames, Delay: delays}); err != nil {
		log.Fatal("Cannot create image: ", err)
	}
}

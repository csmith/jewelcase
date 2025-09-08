package jewelcase

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	xdraw "golang.org/x/image/draw"
)

//go:embed frame.jpg
var frameData []byte

var frame image.Image

// ErrAlreadyProcessed is returned when an image appears to already have the jewel case effect applied.
var ErrAlreadyProcessed = errors.New("image appears to be already processed")

func init() {
	var err error
	frame, err = jpeg.Decode(bytes.NewReader(frameData))
	if err != nil {
		panic(fmt.Sprintf("failed to decode embedded frame: %v", err))
	}
}

const (
	targetWidth  = 750
	targetHeight = 750
	frameOffsetX = 98
	frameOffsetY = 13
)

// Options controls which visual effects are applied to the album art.
type Options struct {
	// ColourCorrection applies subtle saturation and contrast reduction with a blue tint
	ColourCorrection bool

	// RoundedCorners applies randomly-sized rounded corners to the image
	RoundedCorners bool

	// EdgeSoftening applies alpha transparency to the edges for a softer look
	EdgeSoftening bool

	// RandomOffset applies a small random positional offset when placing the image in the frame
	RandomOffset bool

	// RandomRotation applies a subtle random rotation to the image
	RandomRotation bool

	// Reflection adds a diagonal white highlight to simulate light reflection
	Reflection bool

	// Force processes images even if they appear to already be processed
	Force bool
}

// Process applies the jewel case frame and effects to the provided album art image.
// The input image is scaled and cropped to fit the frame, then various effects are applied
// based on the provided Options. Returns ErrAlreadyProcessed if the image appears to already
// be processed (unless opts.Force is true). Returns the final framed image.
func Process(albumArt image.Image, opts Options) (image.Image, error) {
	// Skip images that are already the output size unless forced
	if !opts.Force {
		bounds := albumArt.Bounds()
		frameBounds := frame.Bounds()
		if bounds.Dx() == frameBounds.Dx() && bounds.Dy() == frameBounds.Dy() {
			return nil, ErrAlreadyProcessed
		}
	}

	output := scaleAndCrop(albumArt)

	if opts.ColourCorrection {
		output = applyColourCorrection(output)
	}
	if opts.EdgeSoftening {
		output = applyEdgeSoftening(output)
	}
	if opts.RoundedCorners {
		output = applyRoundedCorners(output)
	}
	if opts.Reflection {
		output = applyReflection(output)
	}
	if opts.RandomRotation {
		output = applyRotation(output)
	}

	finalX := frameOffsetX
	finalY := frameOffsetY
	if opts.RandomOffset {
		finalX += int(rand.Float64()*17) - 8 // -8 to +8
		finalY += int(rand.Float64()*11) - 5 // -5 to +5
	}

	result := image.NewRGBA(frame.Bounds())
	draw.Draw(result, result.Bounds(), frame, image.Point{}, draw.Src)
	draw.Draw(result, image.Rect(finalX, finalY, finalX+targetWidth, finalY+targetHeight), output, image.Point{}, draw.Over)
	return result, nil
}

func loadImage(inputPath string) (image.Image, error) {
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return nil, err
	}
	defer inputFile.Close()

	var img image.Image
	ext := strings.ToLower(filepath.Ext(inputPath))
	switch ext {
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(inputFile)
	case ".png":
		img, err = png.Decode(inputFile)
	default:
		return nil, fmt.Errorf("unsupported image format: %s", ext)
	}

	return img, err
}

func saveImage(img image.Image, outputPath string) error {
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	ext := strings.ToLower(filepath.Ext(outputPath))
	switch ext {
	case ".jpg", ".jpeg":
		return jpeg.Encode(outputFile, img, &jpeg.Options{Quality: 95})
	case ".png":
		return png.Encode(outputFile, img)
	default:
		return fmt.Errorf("unsupported output format: %s", ext)
	}
}

// ProcessFile applies the jewel case effect to an image file and saves the result.
// Reads from inputPath, applies effects, and writes to outputPath. The output format
// is determined by the outputPath extension. Supports JPEG and PNG output formats.
func ProcessFile(inputPath, outputPath string, opts Options) error {
	img, err := loadImage(inputPath)
	if err != nil {
		return err
	}

	result, err := Process(img, opts)
	if err != nil {
		return err
	}

	return saveImage(result, outputPath)
}

func scaleAndCrop(albumArt image.Image) *image.RGBA {
	bounds := albumArt.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	scale := max(float64(targetWidth)/float64(width), float64(targetHeight)/float64(height))
	scaledWidth := int(float64(width) * scale)
	scaledHeight := int(float64(height) * scale)

	scaled := image.NewRGBA(image.Rect(0, 0, scaledWidth, scaledHeight))
	xdraw.BiLinear.Scale(scaled, scaled.Bounds(), albumArt, albumArt.Bounds(), xdraw.Over, nil)

	cropX := (scaledWidth - targetWidth) / 2
	cropY := (scaledHeight - targetHeight) / 2
	output := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	draw.Draw(output, output.Bounds(), scaled, image.Point{X: cropX, Y: cropY}, draw.Src)

	return output
}

func applyRotation(img *image.RGBA) *image.RGBA {
	bounds := img.Bounds()

	angle := (rand.Float64() - 0.5) * math.Pi / 180
	cos := math.Abs(math.Cos(angle))
	sin := math.Abs(math.Sin(angle))
	scale := math.Min(1.0/(cos+sin), 1.0)

	scaledSize := int(float64(targetWidth) * scale)
	scaled := image.NewRGBA(image.Rect(0, 0, scaledSize, scaledSize))
	xdraw.BiLinear.Scale(scaled, scaled.Bounds(), img, img.Bounds(), xdraw.Over, nil)

	result := image.NewRGBA(bounds)
	centerX, centerY := float64(targetWidth)/2, float64(targetHeight)/2

	for y := 0; y < targetHeight; y++ {
		for x := 0; x < targetWidth; x++ {
			// Translate to center, rotate, translate back
			fx := float64(x) - centerX
			fy := float64(y) - centerY
			rx := fx*math.Cos(-angle) - fy*math.Sin(-angle)
			ry := fx*math.Sin(-angle) + fy*math.Cos(-angle)
			rx += float64(scaledSize) / 2
			ry += float64(scaledSize) / 2

			// Bilinear interpolation for smooth edges
			if rx >= 1 && ry >= 1 && rx < float64(scaledSize-1) && ry < float64(scaledSize-1) {
				x0, y0 := int(rx), int(ry)
				x1, y1 := x0+1, y0+1
				fx, fy := rx-float64(x0), ry-float64(y0)

				c00 := scaled.RGBAAt(x0, y0)
				c01 := scaled.RGBAAt(x0, y1)
				c10 := scaled.RGBAAt(x1, y0)
				c11 := scaled.RGBAAt(x1, y1)

				r := uint8(float64(c00.R)*(1-fx)*(1-fy) + float64(c10.R)*fx*(1-fy) +
					float64(c01.R)*(1-fx)*fy + float64(c11.R)*fx*fy)
				g := uint8(float64(c00.G)*(1-fx)*(1-fy) + float64(c10.G)*fx*(1-fy) +
					float64(c01.G)*(1-fx)*fy + float64(c11.G)*fx*fy)
				b := uint8(float64(c00.B)*(1-fx)*(1-fy) + float64(c10.B)*fx*(1-fy) +
					float64(c01.B)*(1-fx)*fy + float64(c11.B)*fx*fy)
				a := uint8(float64(c00.A)*(1-fx)*(1-fy) + float64(c10.A)*fx*(1-fy) +
					float64(c01.A)*(1-fx)*fy + float64(c11.A)*fx*fy)

				result.Set(x, y, color.RGBA{R: r, G: g, B: b, A: a})
			}
		}
	}

	return result
}

func applyReflection(img *image.RGBA) *image.RGBA {
	bounds := img.Bounds()
	result := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			original := img.RGBAAt(x, y)

			fx := float64(x) / float64(targetWidth)
			fy := float64(y) / float64(targetHeight)

			// Add slight white highlight based on diagonal position
			reflectionIntensity := math.Max(0, 0.3*(1-(fx+fy)/2))
			r := math.Min(255, float64(original.R)+reflectionIntensity*40)
			g := math.Min(255, float64(original.G)+reflectionIntensity*40)
			b := math.Min(255, float64(original.B)+reflectionIntensity*40)

			result.Set(x, y, color.RGBA{
				R: uint8(r),
				G: uint8(g),
				B: uint8(b),
				A: original.A,
			})
		}
	}

	return result
}

func applyColourCorrection(img *image.RGBA) *image.RGBA {
	bounds := img.Bounds()
	corrected := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			fr := float64(uint8(r >> 8))
			fg := float64(uint8(g >> 8))
			fb := float64(uint8(b >> 8))

			// Reduce saturation
			avg := (fr + fg + fb) / 3
			fr = fr*0.9 + avg*0.1
			fg = fg*0.9 + avg*0.1
			fb = fb*0.9 + avg*0.1

			// Reduce contrast
			fr = fr*0.95 + 128*0.05
			fg = fg*0.95 + 128*0.05
			fb = fb*0.95 + 128*0.05

			// Blue tint
			fb = math.Min(255, fb*1.02)

			corrected.Set(x, y, color.RGBA{
				R: uint8(math.Max(0, math.Min(255, fr))),
				G: uint8(math.Max(0, math.Min(255, fg))),
				B: uint8(math.Max(0, math.Min(255, fb))),
				A: uint8(a >> 8),
			})
		}
	}

	return corrected
}

func applyRoundedCorners(img *image.RGBA) *image.RGBA {
	bounds := img.Bounds()
	result := image.NewRGBA(bounds)

	topLeftRadius := 6 + rand.Float64()*6
	topRightRadius := 6 + rand.Float64()*6
	bottomLeftRadius := 6 + rand.Float64()*6
	bottomRightRadius := 6 + rand.Float64()*6

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			distFromLeft := float64(x - bounds.Min.X)
			distFromRight := float64(bounds.Max.X - x - 1)
			distFromTop := float64(y - bounds.Min.Y)
			distFromBottom := float64(bounds.Max.Y - y - 1)

			if shouldRound(distFromLeft, distFromTop, topLeftRadius) ||
				shouldRound(distFromRight, distFromTop, topRightRadius) ||
				shouldRound(distFromLeft, distFromBottom, bottomLeftRadius) ||
				shouldRound(distFromRight, distFromBottom, bottomRightRadius) {
				result.Set(x, y, color.NRGBA{})
			} else {
				result.Set(x, y, img.At(x, y))
			}
		}
	}

	return result
}

func shouldRound(dist1, dist2, radius float64) bool {
	if dist1 >= radius || dist2 >= radius {
		return false
	}
	cornerDist := math.Sqrt((radius-dist1)*(radius-dist1)+
		(radius-dist2)*(radius-dist2)) - radius
	return cornerDist > 0
}

func applyEdgeSoftening(img *image.RGBA) *image.RGBA {
	bounds := img.Bounds()
	result := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			distFromLeft := float64(x - bounds.Min.X)
			distFromRight := float64(bounds.Max.X - x - 1)
			distFromTop := float64(y - bounds.Min.Y)
			distFromBottom := float64(bounds.Max.Y - y - 1)

			minDist := math.Min(
				math.Min(distFromLeft, distFromRight),
				math.Min(distFromTop, distFromBottom),
			)

			if minDist < 2 {
				alpha := minDist / 2.0
				if alpha < 1.0 {
					r, g, b, _ := img.At(x, y).RGBA()
					newAlpha := uint8(255.0 * alpha)
					result.Set(x, y, color.NRGBA{
						R: uint8(r >> 8),
						G: uint8(g >> 8),
						B: uint8(b >> 8),
						A: newAlpha,
					})
					continue
				}
			}

			result.Set(x, y, img.At(x, y))
		}
	}

	return result
}

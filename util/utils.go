package util

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/nfnt/resize"
)

// ScalingAlgorithm defines the type for image scaling algorithms.
type ScalingAlgorithm int

const (
	// Bilinear uses bilinear interpolation.
	Bilinear ScalingAlgorithm = iota
	// NearestNeighbor uses nearest-neighbor interpolation.
	NearestNeighbor
)

const (
	ImageMaxWidth  = 400 // Maximum width for the images
	ImageMaxHeight = 400 // Maximum height for the images
)

// cropImageFast crops the src image horizontally based on ratio (0 to 1).
// ratio=0 shows right half only, ratio=1 shows left half only.
// The cropped area moves accordingly.
func CropImageFast(src *image.Image, ratio float64, algo ScalingAlgorithm) image.Image {
	bounds := (*src).Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// Clamp ratio [0,1]
	ratio = max(0, ratio)
	if ratio > 1 {
		ratio = 1
	}

	endX := int(float64(w) * ratio)
	destRect := image.Rect(0, 0, endX, h)

	croppedImage := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(croppedImage, destRect, *src, image.Point{}, draw.Src)
	return RescaleImageFast(croppedImage, algo)
}

// rescaleImageFast rescales the src image to fit within ImageMaxWidth and ImageMaxHeight using Bilinear interpolation.
func RescaleImageFast(src image.Image, algo ScalingAlgorithm) image.Image {
	w, h := GetScaledBounds(&src)
	var interp resize.InterpolationFunction
	switch algo {
	case NearestNeighbor:
		interp = resize.NearestNeighbor
	default:
		interp = resize.Bilinear
	}
	return resize.Resize(uint(w), uint(h), src, interp)
}

// absDiff calculates the absolute difference between two uint32 values.
func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

// computeImageDiff computes the pixel-wise difference between two images.
// It returns a new image showing the differences and the mean absolute error (MAE).
func ComputeImageDiffFast(
	img1, img2 *image.Image,
	algo ScalingAlgorithm,
	differenceAsMonochrome bool,
) (image.Image, float64, uint64) {
	bounds := (*img1).Bounds()
	// Computing difference requires both images to have the same bounds.
	img22 := *img2
	if !(*img2).Bounds().Eq(bounds) {
		var filter imaging.ResampleFilter
		switch algo {
		case NearestNeighbor:
			filter = imaging.NearestNeighbor
		default:
			filter = imaging.Linear
		}
		img22 = imaging.Resize(*img2, bounds.Dx(), bounds.Dy(), filter)
	}

	var amplificationFactor float64 = 5.0

	diff := image.NewRGBA(bounds)
	var totalDiff uint64
	var pixelCount uint64

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r1, g1, b1, _ := (*img1).At(x, y).RGBA()
			r2, g2, b2, _ := img22.At(x, y).RGBA()

			dr := absDiff(r1, r2) >> 8
			dg := absDiff(g1, g2) >> 8
			db := absDiff(b1, b2) >> 8

			ampR := math.Min(float64(dr)*amplificationFactor, 255)
			ampG := math.Min(float64(dg)*amplificationFactor, 255)
			ampB := math.Min(float64(db)*amplificationFactor, 255)

			// If monochrome, show the same red color for all differences
			if differenceAsMonochrome {
				// If there is no difference, set pixel to black
				if dr == 0 && dg == 0 && db == 0 {
					ampR, ampG, ampB = 0, 0, 0
				} else {
					// Otherwise, set everything to red channel for visibility
					ampG = 0
					ampB = 0
					ampR = 240
				}
			}

			diff.Set(x, y, color.RGBA{
				R: uint8(ampR),
				G: uint8(ampG),
				B: uint8(ampB),
				A: 255,
			})

			totalDiff += uint64(dr) + uint64(dg) + uint64(db)
			pixelCount++
		}
	}

	mae := float64(totalDiff) / float64(pixelCount*3)
	return diff, mae, pixelCount
}

// loadImage attempts to load an image from the given path.
// It includes special handling for .jxl files, converting them to PNG using 'djxl' utility.
// TODO: handle webps and animated versions of those formats
func LoadImage(path string) (image.Image, error) {
	ext := filepath.Ext(path)
	if ext == ".jxl" {
		// Convert JXL to PNG using djxl. This assumes 'djxl' is available in the system's PATH.
		tmpFile := path + ".converted.png"
		cmd := exec.Command("djxl", path, tmpFile)
		err := cmd.Run()
		if err != nil {
			return nil, err
		}
		defer os.Remove(tmpFile)
		return imaging.Open(tmpFile)
	}
	return imaging.Open(path)
}

// formatIntWithSpaces takes an integer and returns a string
// with spaces as thousands separators.
func FormatIntWithSpaces(n int64) string {
	s := strconv.Itoa(int(n))
	if len(s) <= 3 {
		return s
	}

	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return strings.Join(parts, " ")
}

// getScaledBounds calculates the scaled dimensions of an image
// to fit within the maximum allowed dimensions (ImageMaxWidth and ImageMaxHeight).
func GetScaledBounds(srcImage *image.Image) (float32, float32) {
	bounds := (*srcImage).Bounds()

	sizeX := float32(bounds.Dx())
	sizeY := float32(bounds.Dy())

	if bounds.Dx() > int(ImageMaxWidth) || bounds.Dy() > int(ImageMaxHeight) {
		// Scale down to fit within maxSizeX and maxSizeY
		scaleX := float32(bounds.Dx()) / ImageMaxWidth
		scaleY := float32(bounds.Dy()) / ImageMaxHeight

		if scaleX > scaleY {
			sizeX = ImageMaxWidth
			sizeY = float32(bounds.Dy()) / scaleX
		} else {
			sizeX = float32(bounds.Dx()) / scaleY
			sizeY = ImageMaxHeight
		}
	}

	return sizeX, sizeY
}

// MoveFileToTrash moves the specified file to the system trash/recycle bin.
// It uses `trash` command underneath, which should be available on most systems.
func MoveFileToTrash(path string) error {
	cmd := exec.Command("trash", path)
	return cmd.Run()
}

package main

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

// cropImageFast crops the src image horizontally based on ratio (0 to 1).
// ratio=0 shows right half only, ratio=1 shows left half only.
// The cropped area moves accordingly.
func cropImageFast(src *image.Image, ratio float64) image.Image {
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
	return rescaleImageFast(croppedImage)
}

// rescaleImageFast rescales the src image to fit within ImageMaxWidth and ImageMaxHeight using Bilinear interpolation.
func rescaleImageFast(src image.Image) image.Image {
	w, h := getScaledBounds(&src)
	return resize.Resize(uint(w), uint(h), src, resize.Bilinear)
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
func computeImageDiffFast(img1, img2 *image.Image) (image.Image, float64, uint64) {
	bounds := (*img1).Bounds()
	// Computing difference requires both images to have the same bounds.
	img22 := *img2
	if !(*img2).Bounds().Eq(bounds) {
		img22 = imaging.Resize(*img2, bounds.Dx(), bounds.Dy(), imaging.Linear)
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

// wrapStringIntelligently wraps a string into multiple lines,
// splitting at '/' characters, so that each line does not exceed maxLength.
// It prioritizes keeping segments together on a line if possible.
func wrapStringIntelligently(s string, maxLength int) string {
	// No need to wrap if it's already within the limit
	if len(s) <= maxLength {
		return s
	}

	parts := strings.Split(s, "/")
	// If no slashes to split by, return original
	if len(parts) <= 1 {
		return s
	}

	var result strings.Builder
	currentLineLength := 0
	for i, part := range parts {
		// If it's not the first part and we're starting a new segment,
		// we need to account for the '/' character's length.
		segmentLength := len(part)
		if i > 0 {
			segmentLength += 1
		}

		// Check if adding this part (and its preceding slash, if any)
		// would exceed the maxLength for the current line.
		// Also, ensure we don't start a line with just a slash if the previous part was the end of a line.
		if currentLineLength > 0 && currentLineLength+segmentLength > maxLength {
			result.WriteString("\n") // Start a new line
			currentLineLength = 0
		}

		if i > 0 && currentLineLength > 0 { // Add '/' if it's not the first segment and not at the beginning of a new line
			result.WriteString("/")
			currentLineLength += 1
		}

		if i > 0 && currentLineLength == 0 {
			result.WriteString("/")
			currentLineLength += 1 // Start with a slash if it's the first segment on a new line
		}

		result.WriteString(part)
		currentLineLength += len(part)
	}

	return result.String()
}

// loadImage attempts to load an image from the given path.
// It includes special handling for .jxl files, converting them to PNG using 'djxl' utility.
// TODO: handle webps and animated versions of those formats
func loadImage(path string) (image.Image, error) {
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
func formatIntWithSpaces(n int64) string {
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
func getScaledBounds(srcImage *image.Image) (float32, float32) {
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

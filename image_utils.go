package lblconv

import (
	"image"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
)

// resizeImage resamples the image to match the longer and shorter sides (one may be 0) and writes
// the output to outPath.
//
// Returns the resized image along with the width and height scale factors.
func resizeImage(img image.Image, longerSide, shorterSide int,
		downsamplingFilter, upsamplingFilter imaging.ResampleFilter) (
		resized image.Image, scaleWidth, scaleHeight float64, err error) {

	imgBounds := img.Bounds()
	imgWidth := imgBounds.Dx()
	imgHeight := imgBounds.Dy()

	imgLonger := imgWidth
	imgShorter := imgHeight
	isLandscape := true
	if imgHeight > imgWidth {
		imgLonger = imgHeight
		imgShorter = imgWidth
		isLandscape = false
	}

	// Calculate the target dimensions.
	if longerSide <= 0 {
		longerSide = int(math.Round(float64(shorterSide) * (float64(imgLonger) / float64(imgShorter))))
	} else if shorterSide <= 0 {
		shorterSide = int(math.Round(float64(longerSide) * (float64(imgShorter) / float64(imgLonger))))
	}

	// Select the filter based on the direction of the rescaling operation.
	var filter imaging.ResampleFilter
	if longerSide*shorterSide < imgWidth*imgHeight {
		filter = downsamplingFilter
	} else {
		filter = upsamplingFilter
	}

	// Resize.
	if isLandscape {
		resized = imaging.Resize(img, longerSide, shorterSide, filter)
		scaleWidth = float64(longerSide) / float64(imgLonger)
		scaleHeight = float64(shorterSide) / float64(imgShorter)
	} else { // Portrait.
		resized = imaging.Resize(img, shorterSide, longerSide, filter)
		scaleWidth = float64(shorterSide) / float64(imgShorter)
		scaleHeight = float64(longerSide) / float64(imgLonger)
	}

	return resized, scaleWidth, scaleHeight, nil
}

// decodeImageConfig opens the file at path and returns the results of image.DecodeConfig.
func decodeImageConfig(path string) (config image.Config, format string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return image.Config{}, "", err
	}
	defer file.Close()

	return image.DecodeConfig(file)
}

// loadImage reads and decodes the image at path and returns the results of image.Decode.
func loadImage(path string) (img image.Image, format string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	return image.Decode(f)
}

// Saves the image to path, encoding it as PNG or JPG, depending on the file extension of path.
func saveImage(path string, img image.Image, jpegQuality int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		err = png.Encode(f, img)
	default:
		err = jpeg.Encode(f, img, &jpeg.Options{Quality: jpegQuality})
	}
	return err
}

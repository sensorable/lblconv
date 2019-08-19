package lblconv

// The intermediate annotation metadata representation.

import (
	"fmt"
	"image"
	"log"
	"math"
	"math/rand"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/disintegration/imaging"
)

// Keys for known annotation attributes.
const (
	AncestorLabels = "Ancestors"  // Ancestors in the label taxonomy. Type []string.
	Confidence     = "Confidence" // Type float64 in [0.0, 1.0].
	CropCoords     = "CropCoords" // Absolute coords (x1,y1)(x2,y2) in the source image. Type string.
	DetectedText   = "Text"       // Text that is associated with the bounding box. Type string.
)

// Annotation is the intermediate representation of an object label.
type Annotation struct {
	Attributes map[string]interface{} // Additional attributes of this annotation.
	Coords     [4]float64             // Absolute x1, y1, x2, y2 offsets from the top-left corner.
	Label      string
}

// Width is the object width from a.Coords.
func (a Annotation) Width() float64 {
	return a.Coords[2] - a.Coords[0]
}

// Height is the object height from a.Coords.
func (a Annotation) Height() float64 {
	return a.Coords[3] - a.Coords[1]
}

// AnnotatedFile is the intermediate representation of file metadata.
type AnnotatedFile struct {
	Annotations []Annotation // The annotations.
	FilePath    string       // The annotated file.
}

// scaleCoords scales all Annotations.Coords by the given scale factors.
func (f *AnnotatedFile) scaleCoords(width, height float64) {
	for i := range f.Annotations {
		for j := 0; j < 4; j++ {
			if j&1 == 0 {
				f.Annotations[i].Coords[j] *= width
			} else {
				f.Annotations[i].Coords[j] *= height
			}
		}
	}
}

type subImager interface {
	SubImage(r image.Rectangle) image.Image
}

// cropObjectsFromImage returns a crop of img for each annotation with a bounding box that is at
// least partially contained in img. The crops may share their data with the original image.
//
// In addition it returns an []AnnotatedFile, one for each cropped image. The file paths are
// derived from f.FilePath, with a "_xx" suffix appended before the file extension, where xx is the
// index in f.Annotations.
func (f *AnnotatedFile) cropObjectsFromImage(img image.Image) (
		[]image.Image, []AnnotatedFile, error) {

	img2, ok := img.(subImager)
	if !ok {
		return nil, nil,
				fmt.Errorf("the image type of %q does not provide a SubImage method", f.FilePath)
	}

	crops := make([]image.Image, 0, len(f.Annotations))
	annotatedFiles := make([]AnnotatedFile, 0, len(f.Annotations))
	bounds := img.Bounds()

	for i, a := range f.Annotations {
		// Clip the bounding box to the image bounds.
		r := image.Rect(int(math.Round(a.Coords[0])), int(math.Round(a.Coords[1])),
			int(math.Round(a.Coords[2])), int(math.Round(a.Coords[3])))
		r = r.Intersect(bounds)
		if r.Empty() {
			continue
		}

		// Make a shallow clone of the annotation's attributes and add the CropCoords.
		attrs := make(map[string]interface{}, 1+len(a.Attributes))
		for k, v := range a.Attributes {
			attrs[k] = v
		}
		attrs[CropCoords] = fmt.Sprintf("(%d,%d)(%d,%d)", r.Min.X, r.Min.Y, r.Max.X, r.Max.Y)

		// Construct the file path for the crop from the original path.
		ext := filepath.Ext(f.FilePath)
		path := fmt.Sprintf("%s_%02d%s", f.FilePath[0:len(f.FilePath)-len(ext)], i, ext)

		// Create the annotation for the crop with a bounding box covering the entire area.
		fileData := AnnotatedFile{
			Annotations: []Annotation{
				{
					Attributes: attrs,
					Coords:     [4]float64{0, 0, float64(r.Dx()), float64(r.Dy())},
					Label:      a.Label,
				},
			},
			FilePath: path,
		}

		crops = append(crops, img2.SubImage(r))
		annotatedFiles = append(annotatedFiles, fileData)
	}

	return crops, annotatedFiles, nil
}

// AnnotatedFiles is the annotation metadata for a list of files.
type AnnotatedFiles []AnnotatedFile

// MapLabels replaces label (sub-)strings with substitution values, as specified in mappings.
//
// The format of mappings is old=new.
func (data *AnnotatedFiles) MapLabels(mappings []string) error {
	if len(mappings) == 0 {
		return nil
	}

	// Extract the individual old and new strings to map between.
	replacements := make([]struct{ old, new string }, len(mappings))
	for i, v := range mappings {
		a := strings.Split(v, "=")
		if len(a) != 2 {
			return fmt.Errorf("invalid mapping: %v", v)
		}

		replacements[i].old = a[0]
		replacements[i].new = a[1]
	}

	// Apply the replacements, in order, to all labels.
	count := 0
	for _, f := range *data {
		for i, aLen := 0, len(f.Annotations); i < aLen; i++ {
			a := &f.Annotations[i]

			oldLabel := a.Label
			for _, r := range replacements {
				a.Label = strings.Replace(a.Label, r.old, r.new, -1)
			}

			if a.Label != oldLabel {
				count++
			}
		}
	}

	log.Printf("The label mappings changed %d labels", count)
	return nil
}

// TransformBboxes transforms bounding boxes.
//
// First bboxes are scaled by the horizontal and vertical scale factors scaleX and scaleY.
//
// Next, the bounding box is grown (never shrunk) to match the desired aspect ratio. An aspectRatio
// of zero disables this transformation.
func (data *AnnotatedFiles) TransformBboxes(scaleX, scaleY, aspectRatio float64) {
	for _, f := range *data {
		for i, aLen := 0, len(f.Annotations); i < aLen; i++ {
			a := &f.Annotations[i]

			// Scale.
			if scaleX != 1 || scaleY != 1 {
				w := a.Width()
				h := a.Height()
				dx := (w*scaleX - w) * 0.5
				dy := (h*scaleY - h) * 0.5

				a.Coords[0] -= dx
				a.Coords[1] -= dy
				a.Coords[2] += dx
				a.Coords[3] += dy
			}

			// Grow to match desired aspect ratio.
			if aspectRatio > 0 {
				// Calculate the ratio so that the expansion works even if one of width or height is zero.
				w := a.Width()
				h := a.Height()
				var ratio float64
				if h != 0 {
					ratio = w / h
				} else {
					ratio = math.MaxFloat64
				}

				if ratio < aspectRatio {
					// Expand horizontally.
					dx := (h*aspectRatio - w) * 0.5
					a.Coords[0] -= dx
					a.Coords[2] += dx
				} else if ratio > aspectRatio {
					// Expand vertically.
					dy := (w/aspectRatio - h) * 0.5
					a.Coords[1] -= dy
					a.Coords[3] += dy
				}
			}
		}
	}
}

// Filter filters out annotations which do not match any of the given labelNames, have a confidence
// value less than minConfidence, a bounding box with less than minBboxWidth or minBboxHeight, or
// do not match the required aspect ratio.
//
// The aspect ratio of width/height must be in [minAspectRatio, maxAspectRatio], except that a
// min/max value of zero disables the respective filter.
//
// If attributes is non empty, only the listed attributes are kept. This only filters the list
// of attributes, not the annotations themselve.
//
// Similarly, requiredAttrs specifies attributes that must be present with a value that is not the
// Go zero value of their type. If this test fails for an annotation, that annotation is deleted.
func (data *AnnotatedFiles) Filter(labelNames, attributes, requiredAttrs []string,
		minConfidence float64, requireLabel bool, minBboxWidth, minBboxHeight, minAspectRatio,
		maxAspectRatio float64) {

	// Deletes the annotation at index i.
	deleteAnnotation := func(annotations []Annotation, i int) []Annotation {
		l := len(annotations)
		annotations[i] = annotations[l-1]
		return annotations[:l-1]
	}

	// Look for string in list.
	inList := func(v string, l []string) bool {
		for _, val := range l {
			if val == v {
				return true
			}
		}
		return false
	}

	numFiles := len(*data)
	numLabelsBeforeFilter := 0
	numLabelsAfterFilter := 0

	// Apply filters.
	for dataIdx, dataLen := 0, len(*data); dataIdx < dataLen; dataIdx++ {
		d := &(*data)[dataIdx]
		numLabelsBeforeFilter += len(d.Annotations)

		// Annotation filters.
	annotationLoop:
		for i, aLen := 0, len(d.Annotations); i < aLen; i++ {
			a := &d.Annotations[i]

			// Filter by confidence. If the annotation has no confidence value then it passes the filter.
			if c, ok := a.Attributes[Confidence].(float64); ok && c < minConfidence {
				d.Annotations = deleteAnnotation(d.Annotations, i)
				aLen--
				i--
				continue
			}

			// Filter by bbox size.
			width := a.Width()
			height := a.Height()
			if minBboxWidth > width || minBboxHeight > height {
				d.Annotations = deleteAnnotation(d.Annotations, i)
				aLen--
				i--
				continue
			}

			// Filter by bbox aspect ratio.
			if minAspectRatio != 0 || maxAspectRatio != 0 {
				keep := height != 0
				if keep {
					ratio := width / height
					keep = (minAspectRatio == 0 || ratio >= minAspectRatio) &&
							(maxAspectRatio == 0 || ratio <= maxAspectRatio)
				}
				if !keep {
					d.Annotations = deleteAnnotation(d.Annotations, i)
					aLen--
					i--
					continue
				}
			}

			// Filter by labels.
			if len(labelNames) > 0 && !inList(a.Label, labelNames) {
				d.Annotations = deleteAnnotation(d.Annotations, i)
				aLen--
				i--
				continue
			}

			// Filter by required attributes with non zero value.
			if len(requiredAttrs) > 0 {
				for _, k := range requiredAttrs {
					// Test against the zero value of the underlying type.
					if v := a.Attributes[k]; v == nil || v == reflect.Zero(reflect.TypeOf(v)).Interface() {
						d.Annotations = deleteAnnotation(d.Annotations, i)
						aLen--
						i--
						continue annotationLoop
					}
				}
			}

			// Filter attributes.
			if len(attributes) > 0 {
				for k := range a.Attributes {
					if !inList(k, attributes) {
						delete(a.Attributes, k)
					}
				}
			}
		}

		numLabelsAfterFilter += len(d.Annotations)

		// Delete the file annotation if files with no labels are filtered out.
		if requireLabel && len(d.Annotations) == 0 {
			dataLen--
			(*data)[dataIdx] = (*data)[dataLen]
			*data = (*data)[0:dataLen]
			dataIdx--
		}
	}

	log.Printf("Filtered out %d labels and %d files",
		numLabelsBeforeFilter-numLabelsAfterFilter, numFiles-len(*data))
}

// ProcessImages resizes all referenced images and writes them to imageOutDir using the specified
// encoding.
//
// If doCropObjects is true, individual objects as per the labels are cropped from the images. The
// crops are resized instead of the original images in this case. The data changes accordingly, with
// 0 or more cropped images replacing the original AnnotatedFile.
func (data *AnnotatedFiles) ProcessImages(imageOutDir string, longerSide, shorterSide int,
		downsamplingFilter, upsamplingFilter, encoding string, jpegQuality int,
		doCropObjects bool) error {

	doResizeImages := longerSide > 0 || shorterSide > 0
	if !doResizeImages && !doCropObjects {
		return nil
	}
	log.Print("Processing images")

	// Select the resampling algorithms.
	downsample := imaging.Box
	upsample := imaging.Linear
	filters := []struct {
		name   string
		filter *imaging.ResampleFilter
	}{
		{downsamplingFilter, &downsample},
		{upsamplingFilter, &upsample},
	}
	for _, v := range filters {
		switch v.name {
		case "nearest":
			*v.filter = imaging.NearestNeighbor
		case "box":
			*v.filter = imaging.Box
		case "linear":
			*v.filter = imaging.Linear
		case "gaussian":
			*v.filter = imaging.Gaussian
		case "lanczos":
			*v.filter = imaging.Lanczos
		default:
			return fmt.Errorf("unknown resampling filter %q", v.name)
		}
	}

	// Select the output file extension based on the requested encoding.
	var fileExt string
	switch strings.ToLower(encoding) {
	case "jpg", "jpeg":
		fileExt = ".jpg"
	case "png":
		fileExt = ".png"
	default:
		return fmt.Errorf("unsupported output encoding %q", encoding)
	}

	// Prepare for concurrent processing. Limit the number of goroutines in flight, as they load
	// potentially large images into memory.
	numTasks := 2 * runtime.NumCPU()
	if len(*data) < numTasks {
		numTasks = len(*data)
	}
	workQueue := make(chan *AnnotatedFile, 2*numTasks)

	var croppedData []AnnotatedFile
	var croppedDataCh chan *AnnotatedFile
	if doCropObjects {
		croppedData = make([]AnnotatedFile, 0, len(*data))
		croppedDataCh = make(chan *AnnotatedFile, 2*numTasks)
	}

	errors := make(chan error, 1)
	var wg sync.WaitGroup

	// Process images concurrently from a work queue.
	wg.Add(numTasks)
	for i := 0; i < numTasks; i++ {
		go func() {
			defer wg.Done()
			for d := range workQueue {
				processImage(d, imageOutDir, fileExt, longerSide, shorterSide, downsample,
					upsample, jpegQuality, doCropObjects, doResizeImages, croppedDataCh, errors)
			}
		}()
	}

	// Append image metadata for cropped images.
	var wgAppend sync.WaitGroup
	if doCropObjects {
		wgAppend.Add(1)
		go func() {
			defer wgAppend.Done()
			for d := range croppedDataCh {
				croppedData = append(croppedData, *d)
			}
		}()
	}

	// Feed the work queue.
	for i := range *data {
		workQueue <- &(*data)[i]
	}
	close(workQueue)

	// Wait for image processing to finish.
	wg.Wait()
	if doCropObjects {
		// Wait for all new metadata to be appended and then replace the old data.
		close(croppedDataCh)
		wgAppend.Wait()
		*data = croppedData
	}

	close(errors)
	if len(errors) > 0 {
		return <-errors
	}

	return nil
}

// processImage processes the image described by data.
//
// If and only if doCropObjects is true, new metadata for the image crops is written to croppedData.
func processImage(data *AnnotatedFile, imageOutDir, fileExt string, longerSide, shorterSide int,
		downsample, upsample imaging.ResampleFilter, jpegQuality int, doCropObjects, doResizeImage bool,
		croppedData chan<- *AnnotatedFile, errors chan<- error) {

	trySendError := func(err error) {
		select {
		case errors <- err:
		default:
		}
	}

	// Read the image.
	img, _, err := loadImage(data.FilePath)
	if err != nil {
		trySendError(err)
		return
	}

	// Crop labelled objects from the image if requested.
	var images []image.Image
	var imageData []*AnnotatedFile
	if doCropObjects {
		// The original image is not further processed in this case.
		var tmpData []AnnotatedFile
		images, tmpData, err = data.cropObjectsFromImage(img)
		if err != nil {
			trySendError(err)
			return
		}

		imageData = make([]*AnnotatedFile, len(tmpData))
		for i := range tmpData {
			imageData[i] = &tmpData[i]
		}
	} else {
		images = []image.Image{img}
		imageData = []*AnnotatedFile{data}
	}

	// Process either the original image or the crops.
	for i, img := range images {
		data := imageData[i]

		// Resize.
		var scaleWidth, scaleHeight float64
		if doResizeImage {
			img, scaleWidth, scaleHeight, err =
					resizeImage(img, longerSide, shorterSide, downsample, upsample)
			if err != nil {
				trySendError(err)
				return
			}
		}

		// Save the image.
		inName := filepath.Base(data.FilePath)
		inFileExt := filepath.Ext(inName)
		outName := inName[0:len(inName)-len(inFileExt)] + fileExt
		outPath := filepath.Join(imageOutDir, outName)
		if err := saveImage(outPath, img, jpegQuality); err != nil {
			trySendError(err)
			return
		}

		// Update the image file path and rescale the coordinates.
		data.FilePath = outPath
		if doResizeImage {
			data.scaleCoords(scaleWidth, scaleHeight)
		}

		// Return the metadata for the cropped image.
		if doCropObjects {
			croppedData <- data
		}
	}
}

// Split randomly splits the data into multiple datasets.
//
// The cumulativeSplits specify the cumulative distribution according to which the data is split
// into the returned datasets. Its values must add up to 100!
func (data *AnnotatedFiles) Split(cumulativeSplits []int) ([]AnnotatedFiles, error) {
	datasets := make([]AnnotatedFiles, len(cumulativeSplits))

	// Allocate slightly more than the expected size for each dataset.
	var sum int
	for i, s := range cumulativeSplits {
		percent := s - sum
		datasets[i] = make(AnnotatedFiles, 0, int(1.05*float64(percent)/100*float64(len(*data))))
		sum = s
	}
	if sum != 100 {
		return nil, fmt.Errorf("the split percentages do not add up to 100")
	}

	// Split the data.
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

outer:
	for _, d := range *data {
		r := rng.Intn(100)
		for i, s := range cumulativeSplits {
			if r < s {
				datasets[i] = append(datasets[i], d)
				continue outer
			}
		}
	}

	return datasets, nil
}

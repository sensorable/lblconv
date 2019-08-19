// Converts between KITTI, Sloth, AWS detect-labels, AWS detect-text, TFRecord and
// VGG Image Annotator label formats.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sensorable/lblconv"
)

var (
	convertFrom format // The source format.
	convertTo   format // The target format.

	imageDirPath             string   // The input directory with the labeled images.
	imageOutDirPath          string   // The output directory for images after processing.
	labelFileOrDirPath       string   // The input label directory or file, depending on the format.
	labelOutFileOrDirPaths   []string // The output label dir or file path(s), depending on the format.
	labelOutSplits           []int    // The cumulative split percentages for the output datasets.
	tfRecordLabelMapFilePath string   // The TFRecord label map file.
	numShardFiles            int      // The number of shard files to create.

	labelMappings   string  // A comma-separated string of label mappings.
	bboxScaleWidth  float64 // A scale factor for the bounding box width.
	bboxScaleHeight float64 // A scale factor for the bounding box height.
	bboxAspectRatio float64 // The desired output aspect ratio for bounding boxes.

	filterLabels         string  // A comma-separated string of labels to keep (empty keeps all).
	filterAttributes     string  // A comma-separated string of attributes to keep (empty keeps all).
	filterRequiredAttrs  string  // A comma-sep. str of required attrs (present and not zero value).
	filterConfidence     float64 // The min. confidence value.
	filterRequireLabel   bool    // Filter out files with no labels (after other filters).
	filterMinBboxWidth   float64 // The minimum bounding box width.
	filterMinBboxHeight  float64 // The minimum bounding box height.
	filterMinAspectRatio float64 // The minimum aspect ratio of bboxes (w/h).
	filterMaxAspectRatio float64 // The maximum aspect ratio of bboxes (w/h).

	imageOutEncoding        string // The file type for image outputs.
	imageResizeLonger       int    // The target length for the longer side of the image.
	imageResizeShorter      int    // The target length for the shorter side of the image.
	imageDownsamplingFilter string // The algorithm to use when downsampling.
	imageUpsamplingFilter   string // The algorithm to use when upsampling.
	imageJPEGQuality        int    // The JPEG quality for JPEG outputs.

	imageCropObjects bool // Crop individual objects from images and output these instead.
)

type format int

// The known label formats.
const (
	Unknown format = iota // If an unknown format is specified.
	AWSDetectLabels
	AWSDetectText
	Kitti
	Sloth
	TFRecord
	VIA  // VGG Image Annotator
)

func formatFrom(s string) format {
	switch s {
	case "aws-dl":
		return AWSDetectLabels
	case "aws-dt":
		return AWSDetectText
	case "kitti":
		return Kitti
	case "sloth":
		return Sloth
	case "tfrecord":
		return TFRecord
	case "via":
		return VIA
	}
	return Unknown
}

func init() {
	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "Usage of %s:\n", filepath.Base(os.Args[0]))
		_, _ = fmt.Fprintln(os.Stderr, "  aws-dl input options:\t\t-labels <dir> -images <dir>")
		_, _ = fmt.Fprintln(os.Stderr, "  aws-dt input options:\t\t-labels <dir> -images <dir>")
		_, _ = fmt.Fprintln(os.Stderr, "  kitti input options:\t\t-labels <dir> -images <dir>")
		_, _ = fmt.Fprintln(os.Stderr, "  kitti output options:\t\t-labels-out <dir>")
		_, _ = fmt.Fprintln(os.Stderr, "  sloth input options:\t\t-labels <file>")
		_, _ = fmt.Fprintln(os.Stderr, "  sloth output options:\t\t-labels-out <file>")
		_, _ = fmt.Fprintln(os.Stderr, "  tfrecord output options:\t-labels-out <file>"+
				" -tfrecord-label-map-file [-num-shards]")
		_, _ = fmt.Fprintln(os.Stderr, "  via input options:\t\t-labels <file>")
		_, _ = fmt.Fprintln(os.Stderr, "  via output options:\t\t-labels-out <file>")
		_, _ = fmt.Fprintln(os.Stderr)
		flag.PrintDefaults()
	}

	printUsageAndExit := func(msg ...interface{}) {
		log.Print(msg...)
		flag.Usage()
		os.Exit(1)
	}

	// Format arguments.
	from := flag.String("from", "", "The source `format`")
	to := flag.String("to", "", "The target `format`")

	// Path arguments.
	flag.StringVar(&imageDirPath, "images", imageDirPath,
		"The `path` to the image input directory")
	flag.StringVar(&imageOutDirPath, "images-out", imageOutDirPath,
		"The `path` to the image output directory (only required when image processing"+
				" functionality is used")
	flag.StringVar(&labelFileOrDirPath, "labels", labelFileOrDirPath,
		"The `path` to the label input file (sloth, via) or directory (kitti, aws-dl, aws-dt)")
	outPaths := flag.String("labels-out", "",
		"The comma-separated paths (`path[,...]`) to the label output files (sloth, tfrecord, via)"+
				" or directories (kitti); must be one path per value in flag -split")
	outSplits := flag.String("split", "100",
		"The comma-separated output split percentages (`percent[,...]`) to divide labels into"+
				" (only sloth, tfrecord, and via output formats); must add up to 100%")
	flag.StringVar(&tfRecordLabelMapFilePath, "tfrecord-label-map-file", tfRecordLabelMapFilePath,
		"The TFRecord label map file `path`")

	flag.IntVar(&numShardFiles, "num-shards", 1,
		"The number of shard files to create (tfrecord only)")

	// Conversion and transformation arguments.
	flag.StringVar(&labelMappings, "map-labels", labelMappings,
		"Comma-separated list of old=new label (sub-)string replacements")
	flag.Float64Var(&bboxScaleWidth, "bbox-scale-x", 1,
		"A scale factor for the width of all bounding boxes")
	flag.Float64Var(&bboxScaleHeight, "bbox-scale-y", 1,
		"A scale factor for the height of all bounding boxes")
	flag.Float64Var(&bboxAspectRatio, "bbox-aspect-ratio", 0,
		"The output aspect `ratio` for object bounding boxes; bounding boxes are grown (not shrunk)"+
				" to match this ratio when it is > 0")

	// Filter arguments.
	flag.StringVar(&filterLabels, "filter-labels", filterLabels,
		"Comma-separated list of labels to keep (after map-labels; empty string keeps all)")
	flag.StringVar(&filterAttributes, "filter-attributes", filterAttributes,
		"Comma-separated list of attributes to keep (if the target format supports attributes;"+
				" empty string keeps all)")
	flag.StringVar(&filterRequiredAttrs, "filter-required-attrs", filterRequiredAttrs,
		"Comma-separated list of required attributes whose values must not be the Go zero value for"+
				" their type to keep the annotation")
	flag.Float64Var(&filterConfidence, "min-confidence", filterConfidence,
		"The minimum confidence value to keep a label; range [0.0, 1.0)")
	flag.BoolVar(&filterRequireLabel, "require-label", filterRequireLabel,
		"Require at least one label (after filters) to keep the file")
	flag.Float64Var(&filterMinBboxWidth, "min-bbox-width", filterMinBboxWidth,
		"The min. required width in `pixels` for object bounding boxes (before resizing)")
	flag.Float64Var(&filterMinBboxHeight, "min-bbox-height", filterMinBboxHeight,
		"The min. required height in `pixels` for object bounding boxes (before resizing)")
	flag.Float64Var(&filterMinAspectRatio, "min-bbox-aspect-ratio", filterMinAspectRatio,
		"The min. required aspect `ratio` (width/height) for object bounding boxes (before resizing;"+
				" zero disables the filter)")
	flag.Float64Var(&filterMaxAspectRatio, "max-bbox-aspect-ratio", filterMaxAspectRatio,
		"The max. required aspect `ratio` (width/height) for object bounding boxes (before resizing;"+
				" zero disables the filter)")

	// Image processing arguments.
	flag.StringVar(&imageOutEncoding, "image-enc", "jpg",
		"The `encoding` for output images {jpg, png}")
	flag.IntVar(&imageResizeLonger, "resize-longer", imageResizeLonger,
		"The target `length` for the longer side of the image (zero to keep aspect ratio)")
	flag.IntVar(&imageResizeShorter, "resize-shorter", imageResizeShorter,
		"The target `length` for the shorter side of the image (zero to keep aspect ratio)")
	flag.StringVar(&imageDownsamplingFilter, "downsample-filter", "box",
		"The filter to use when downsampling an image {nearest, box, linear, gaussian, lanczos}")
	flag.StringVar(&imageUpsamplingFilter, "upsample-filter", "linear",
		"The filter to use when upsampling an image {nearest, box, linear, gaussian, lanczos}")
	flag.IntVar(&imageJPEGQuality, "jpeg-quality", 90,
		"The quality to use when encoding JPEGs [1, 100]")
	flag.BoolVar(&imageCropObjects, "crop-objects", imageCropObjects,
		"Crop and output objects from images (image processing flags apply to the individual crops)")

	// Parse and validate flags.
	flag.Parse()

	convertFrom = formatFrom(*from)
	convertTo = formatFrom(*to)

	// Validate the conversion direction.
	validInFormat := false
	for _, f := range []format{AWSDetectLabels, AWSDetectText, Kitti, Sloth, VIA} {
		if f == convertFrom {
			validInFormat = true
			break
		}
	}
	validOutFormat := false
	for _, f := range []format{Kitti, Sloth, TFRecord, VIA} {
		if f == convertTo {
			validOutFormat = true
			break
		}
	}
	if !validInFormat {
		printUsageAndExit("Unsupported input format")
	} else if !validOutFormat {
		printUsageAndExit("Unsupported output format")
	}

	// Validate input arguments.
	if labelFileOrDirPath == "" ||
			(convertFrom == Kitti && imageDirPath == "") ||
			(convertFrom == AWSDetectLabels && imageDirPath == "") ||
			(convertFrom == AWSDetectText && imageDirPath == "") {
		printUsageAndExit("Missing label or image input path argument")
	}

	// Validate output split arguments.
	labelOutFileOrDirPaths = strings.Split(*outPaths, ",")
	splits := strings.Split(*outSplits, ",")
	if len(splits) != len(labelOutFileOrDirPaths) {
		printUsageAndExit("The number of output datasets defined by -split and the number of" +
				" paths in -labels-out must match")
	}
	if convertTo == Kitti && len(splits) > 1 {
		printUsageAndExit("Argument -split is not supported with output format \"kitti\"")
	}

	// Parse splits as cumulative int percentages.
	var splitSum int
	for _, v := range splits {
		if i, err := strconv.Atoi(v); err != nil || i < 0 || i > 100 {
			printUsageAndExit("Invalid value in -split: ", v)
		} else {
			splitSum += i
			labelOutSplits = append(labelOutSplits, splitSum)
		}
	}
	if splitSum != 100 {
		printUsageAndExit("The values in -split must add up to 100%")
	}

	// Validate other output arguments.
	if convertTo == TFRecord && tfRecordLabelMapFilePath == "" {
		printUsageAndExit("Missing label output path argument")
	}

	// Transformation arguments.
	if bboxScaleWidth <= 0 || bboxScaleHeight <= 0 {
		printUsageAndExit("Invalid bounding box scale factor")
	} else if bboxAspectRatio < 0 {
		printUsageAndExit("Invalid value for -bbox-aspect-ratio")
	}

	// Image processing arguments.
	if (imageResizeLonger > 0 || imageResizeShorter > 0 || imageCropObjects) &&
			imageOutDirPath == "" {
		printUsageAndExit("Missing image output directory path")
	}
	if imageJPEGQuality < 1 || imageJPEGQuality > 100 {
		imageJPEGQuality = 92
		log.Print("Invalid JPEG quality, setting it to ", imageJPEGQuality)
	}

	// Validate filter arguments.
	if filterConfidence < 0 || filterConfidence >= 1 {
		printUsageAndExit("Invalid -min-confidence, must be in [0.0, 1.0): ", filterConfidence)
	}

	// Clean path arguments.
	if imageDirPath != "" {
		imageDirPath = filepath.Clean(imageDirPath)
	}
	if imageOutDirPath != "" {
		imageOutDirPath = filepath.Clean(imageOutDirPath)
	}
	if imageDirPath != "" && imageDirPath == imageOutDirPath {
		printUsageAndExit("The image input and output paths cannot be identical")
	}

	labelFileOrDirPath = filepath.Clean(labelFileOrDirPath)
	for i, v := range labelOutFileOrDirPaths {
		labelOutFileOrDirPaths[i] = filepath.Clean(v)
		if labelFileOrDirPath == labelOutFileOrDirPaths[i] {
			printUsageAndExit("The label input and output paths cannot be identical")
		}
	}

	tfRecordLabelMapFilePath = filepath.Clean(tfRecordLabelMapFilePath)
}

func main() {
	// Parse input.
	var data []lblconv.AnnotatedFile
	var err error
	switch convertFrom {
	case AWSDetectLabels:
		data, err = lblconv.FromAWSDetectLabels(labelFileOrDirPath, imageDirPath)
	case AWSDetectText:
		data, err = lblconv.FromAWSDetectText(labelFileOrDirPath, imageDirPath)
	case Kitti:
		data, err = lblconv.FromKitti(labelFileOrDirPath, imageDirPath)
	case Sloth:
		data, err = lblconv.FromSloth(labelFileOrDirPath)
	case VIA:
		data, err = lblconv.FromVIA(labelFileOrDirPath)
	default:
		err = fmt.Errorf("unsupported input format")
	}
	if err != nil {
		log.Fatal("Failed to parse the input: ", err)
	}

	af := lblconv.AnnotatedFiles(data)

	// Map labels.
	if len(labelMappings) > 0 {
		if err := af.MapLabels(strings.Split(labelMappings, ",")); err != nil {
			log.Fatal("Failed to map labels: ", err)
		}
	}

	// Perform transformations.
	if bboxScaleWidth != 1 || bboxScaleHeight != 1 || bboxAspectRatio > 0 {
		af.TransformBboxes(bboxScaleWidth, bboxScaleHeight, bboxAspectRatio)
	}

	// Apply filters.
	var labelNames, attrNames, requiredAttrNames []string
	if filterLabels != "" {
		labelNames = strings.Split(filterLabels, ",")
	}
	if filterAttributes != "" {
		attrNames = strings.Split(filterAttributes, ",")
	}
	if filterRequiredAttrs != "" {
		requiredAttrNames = strings.Split(filterRequiredAttrs, ",")
	}
	af.Filter(labelNames, attrNames, requiredAttrNames, filterConfidence, filterRequireLabel,
		filterMinBboxWidth, filterMinBboxHeight, filterMinAspectRatio, filterMaxAspectRatio)

	// Process images.
	err = af.ProcessImages(imageOutDirPath, imageResizeLonger, imageResizeShorter,
		imageDownsamplingFilter, imageUpsamplingFilter, imageOutEncoding, imageJPEGQuality,
		imageCropObjects)
	if err != nil {
		log.Fatal("Image processing failed: ", err)
	}

	// Split data into output datasets.
	var datasets []lblconv.AnnotatedFiles
	if len(labelOutSplits) == 1 {
		datasets = []lblconv.AnnotatedFiles{af}
	} else {
		if datasets, err = af.Split(labelOutSplits); err != nil {
			log.Fatal("Failed to split the dataset: ", err)
		}
	}

	// Write output datasets.
	for i, data := range datasets {
		outPath := labelOutFileOrDirPaths[i]
		switch convertTo {
		case Kitti:
			kittiData := lblconv.ToKitti(data)
			err = lblconv.WriteKitti(outPath, kittiData)
		case Sloth:
			slothData := lblconv.ToSloth(data)
			err = lblconv.WriteSloth(outPath, slothData)
		case TFRecord:
			err = lblconv.WriteTFRecord(outPath, tfRecordLabelMapFilePath, data, numShardFiles)
		case VIA:
			viaData := lblconv.ToVIA(data)
			err = lblconv.WriteVIA(outPath, viaData)
		default:
			err = fmt.Errorf("unsupported output format")
		}
		if err != nil {
			log.Fatal("Conversion failed: ", err)
		}

		log.Printf("Successfully wrote labels for %d files to %s", len(data), outPath)
	}

	log.Print("Total number of labelled files: ", len(af))
}

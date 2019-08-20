# lblconv

lblconv can convert labels between different label formats, with a focus on rectangular bounding
boxes for object detection tasks in computer vision and related image based machine learning
contexts. It supports various filtering and transformation functions for the annotations themselves
and the images they refer to.

Supported formats:
* AWS Rekognition detect-labels (read only)
* AWS Rekognition detect-text (read only)
* KITTI 2D object detection (read/write)
* Sloth (read/write)
* TensorFlow TFRecord (write only)
* VGG Image Annotator (VIA) (read/write)

Note that not all attributes supported by these formats are retained during the conversion.

## Getting Started

### Installing

With [Go (Golang)](https://golang.org) installed on your machine, run the following command to
download, build and install the tool:

```
go get -u github.com/sensorable/lblconv/cmd/lblconv
```

### Usage

The same format can be set as the source and target if both reading and writing are supported for
the given format and the intention is to merely filter or transform the data rather than convert
between formats.

``` 
Usage: lblconv -from <format> -to <format> [<arg> ...]

The supported input (-from) and output (-to) formats and their required arguments:
  AWS Rekognition detect-labels:
    -from aws-dl -labels <dir> -images <dir>
  AWS Rekognition detect-text:
    -from aws-dt -labels <dir> -images <dir>
  KITTI 2D object detection:
    -from kitti -labels <dir> -images <dir>
    -to kitti -labels-out <dir>
  Sloth:
    -from sloth -labels <file>
    -to sloth -labels-out <file>
  TensorFlow TFRecord:
    -to tfrecord -labels-out <file> -tfrecord-label-map-file <file> [-num-shards <int>]
  VGG Image Annotator (VIA):
    -from via -labels <file>
    -to via -labels-out <file>

Arguments:
  -bbox-aspect-ratio ratio
        The output aspect ratio for object bounding boxes; bounding boxes are grown (not shrunk) to match this ratio when it is > 0
  -bbox-scale-x float
        A scale factor for the width of all bounding boxes (default 1)
  -bbox-scale-y float
        A scale factor for the height of all bounding boxes (default 1)
  -crop-objects
        Crop and output objects from images (image processing flags apply to the individual crops)
  -downsample-filter string
        The filter to use when downsampling an image {nearest, box, linear, gaussian, lanczos} (default "box")
  -filter-attributes string
        Comma-separated list of attributes to keep (if the target format supports attributes; empty string keeps all)
  -filter-labels string
        Comma-separated list of labels to keep (after map-labels; empty string keeps all)
  -filter-required-attrs string
        Comma-separated list of required attributes whose values must not be the Go zero value for their type to keep the annotation
  -from format
        The source format
  -image-enc encoding
        The encoding for output images {jpg, png} (default "jpg")
  -images path
        The path to the image input directory
  -images-out path
        The path to the image output directory (only required when image processing functionality is used
  -jpeg-quality int
        The quality to use when encoding JPEGs [1, 100] (default 90)
  -labels path
        The path to the label input file (sloth, via) or directory (kitti, aws-dl, aws-dt)
  -labels-out path[,...]
        The comma-separated paths (path[,...]) to the label output files (sloth, tfrecord, via) or directories (kitti); must be one path per value in flag -split
  -map-labels string
        Comma-separated list of old=new label (sub-)string replacements
  -max-bbox-aspect-ratio ratio
        The max. required aspect ratio (width/height) for object bounding boxes (before resizing; zero disables the filter)
  -min-bbox-aspect-ratio ratio
        The min. required aspect ratio (width/height) for object bounding boxes (before resizing; zero disables the filter)
  -min-bbox-height pixels
        The min. required height in pixels for object bounding boxes (before resizing)
  -min-bbox-width pixels
        The min. required width in pixels for object bounding boxes (before resizing)
  -min-confidence float
        The minimum confidence value to keep a label; range [0.0, 1.0)
  -num-shards int
        The number of shard files to create (tfrecord only) (default 1)
  -require-label
        Require at least one label (after filters) to keep the file
  -resize-longer length
        The target length for the longer side of the image (zero to keep aspect ratio)
  -resize-shorter length
        The target length for the shorter side of the image (zero to keep aspect ratio)
  -split percent[,...]
        The comma-separated output split percentages (percent[,...]) to divide labels into (only sloth, tfrecord, and via output formats); must add up to 100% (default "100")
  -tfrecord-label-map-file path
        The TFRecord label map file path
  -to format
        The target format
  -upsample-filter string
        The filter to use when upsampling an image {nearest, box, linear, gaussian, lanczos} (default "linear")
```

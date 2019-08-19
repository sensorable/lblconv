package lblconv

// AWS Rekognition detect-text specific functionality.

import (
	"encoding/json"
	"io/ioutil"
)

// AWSGeometry is the geometry of a text object annotation.
type AWSGeometry struct {
	BoundingBox AWSBoundingBox
	Polygon     []AWSPoint
}

// AWSTextDetection is a single text annotation within an AWS detect-text label file.
type AWSTextDetection struct {
	Confidence   float64 // Range [0, 100].
	DetectedText string
	Geometry     AWSGeometry
	ID           int
	ParentID     *int   // Nil when Type=="LINE".
	Type         string // LINE or WORD.
}

// AWSDTAnnotatedFile defines the AWS text detection annotation structure for a single file.
type AWSDTAnnotatedFile struct {
	Annotations []AWSTextDetection `json:"TextDetections"`
	FilePath    string             `json:"-"`
}

// FromAWSDetectText reads and parses AWS detect-text annotations from labelDir and matches them
// to the images in imageDir.
func FromAWSDetectText(labelDir, imageDir string) ([]AnnotatedFile, error) {
	return parseLabelsWithOneToOneImages(labelDir, ".json", imageDir, parseAWSDetectTextFile)
}

// parseAWSDetectTextFile parses the label file at labelPath and reads metadata from the
// corresponding image at imagePath to construct an AnnotatedFile struct and return it.
//
// The extracted annotations have label "Text_Line" or "Text_Word" (and fallback "Text"), according
// to the AWSTextDetection.Type.
func parseAWSDetectTextFile(labelPath, imagePath string) (AnnotatedFile, error) {
	// Unmarshal JSON.
	enc, err := ioutil.ReadFile(labelPath)
	if err != nil {
		return AnnotatedFile{}, err
	}

	var awsFileData AWSDTAnnotatedFile
	err = json.Unmarshal(enc, &awsFileData)
	if err != nil {
		return AnnotatedFile{}, err
	}

	// Get the image width and height.
	img, _, err := decodeImageConfig(imagePath)
	if err != nil {
		return AnnotatedFile{}, err
	}

	// Convert to the intermediate representation.
	fileData := AnnotatedFile{
		Annotations: make([]Annotation, 0, len(awsFileData.Annotations)),
		FilePath:    imagePath,
	}
	for _, a := range awsFileData.Annotations {
		annotation := Annotation{
			Attributes: map[string]interface{}{
				Confidence:   a.Confidence / 100,
				DetectedText: a.DetectedText,
			},
			// Scale normalised coordinates to image coordinates.
			Coords: [4]float64{
				a.Geometry.BoundingBox.Left * float64(img.Width),
				a.Geometry.BoundingBox.Top * float64(img.Height),
				(a.Geometry.BoundingBox.Left + a.Geometry.BoundingBox.Width) * float64(img.Width),
				(a.Geometry.BoundingBox.Top + a.Geometry.BoundingBox.Height) * float64(img.Height),
			},
			Label: "Text",
		}
		if a.Type == "LINE" {
			annotation.Label = "Text_Line"
		} else if a.Type == "WORD" {
			annotation.Label = "Text_Word"
		}

		fileData.Annotations = append(fileData.Annotations, annotation)
	}

	return fileData, nil
}

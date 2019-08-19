package lblconv

// AWS Rekognition detect-labels specific functionality.

import (
	"encoding/json"
	"io/ioutil"
)

// AWSInstance is an object instance in an AWS label.
type AWSInstance struct {
	BoundingBox AWSBoundingBox
	Confidence  float64 // Range [0, 100].
}

// AWSLabel is a single annotation within an AWS labels file.
type AWSLabel struct {
	Confidence float64 // Range [0, 100].
	Instances  []AWSInstance
	Name       string
	Parents    []struct {
		Name string
	}
}

// AWSDLAnnotatedFile defines the AWS detect-labels annotation structure for a single file.
type AWSDLAnnotatedFile struct {
	Annotations  []AWSLabel `json:"Labels"`
	FilePath     string     `json:"-"`
	ModelVersion string     `json:"LabelModelVersion"`
}

// FromAWSDetectLabels reads and parses AWS detect-labels annotations from labelDir and matches them
// to the images in imageDir.
func FromAWSDetectLabels(labelDir, imageDir string) ([]AnnotatedFile, error) {
	return parseLabelsWithOneToOneImages(labelDir, ".json", imageDir, parseAWSDetectLabelsFile)
}

// parseAWSDetectLabelsFile parses the label file at labelPath and reads metadata from the
// corresponding image at imagePath to construct an AnnotatedFile struct and return it.
func parseAWSDetectLabelsFile(labelPath, imagePath string) (AnnotatedFile, error) {
	// Unmarshal JSON.
	enc, err := ioutil.ReadFile(labelPath)
	if err != nil {
		return AnnotatedFile{}, err
	}

	var awsFileData AWSDLAnnotatedFile
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
	// AWS annotation instances will be unrolled.
	fileData := AnnotatedFile{
		Annotations: make([]Annotation, 0, 2*len(awsFileData.Annotations)),
		FilePath:    imagePath,
	}
	for _, a := range awsFileData.Annotations {
		// Convert the parents attribute to a []string.
		ancestors := make([]string, len(a.Parents))
		for i, p := range a.Parents {
			ancestors[i] = p.Name
		}

		// Only keep annotations for objects, i.e. annotations with instances. These are unrolled.
		for _, i := range a.Instances {
			annotation := Annotation{
				Attributes: map[string]interface{}{
					AncestorLabels: ancestors,
					Confidence:     i.Confidence / 100,
				},
				// Scale normalised coordinates to image coordinates.
				Coords: [4]float64{
					i.BoundingBox.Left * float64(img.Width),
					i.BoundingBox.Top * float64(img.Height),
					(i.BoundingBox.Left + i.BoundingBox.Width) * float64(img.Width),
					(i.BoundingBox.Top + i.BoundingBox.Height) * float64(img.Height),
				},
				Label: a.Name,
			}

			fileData.Annotations = append(fileData.Annotations, annotation)
		}
	}

	return fileData, nil
}

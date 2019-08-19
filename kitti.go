package lblconv

// KITTI specific functionality.

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// KITTIAnnotation is a single annotation within a KITTI file.
type KITTIAnnotation struct {
	Coords [4]float64 // x1, y1, x2, y2
	Label  string
	Score  float64 // Optional, linear confidence value. No fixed range.
}

// KITTIAnnotatedFile defines the KITTI annotation structure for a single file.
type KITTIAnnotatedFile struct {
	Annotations []KITTIAnnotation
	FilePath    string
}

// FromKitti reads and parses KITTI annotations from labelDir and matches them to the images in
// imageDir.
func FromKitti(labelDir, imageDir string) ([]AnnotatedFile, error) {
	labelFiles, err := filesByExtInDir(labelDir, ".txt")
	if err != nil {
		return nil, err
	}
	log.Printf("Parsing KITTI labels for %d files", len(labelFiles))

	data, err := parseKittiAnnotations(labelFiles, imageDir)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// parseKittiAnnotations parses the KITTI annotations from labelFiles. Expects to find the
// corresponding images in imageDir, with identical base name except for the file extension.
func parseKittiAnnotations(labelFiles []string, imageDir string) ([]AnnotatedFile, error) {
	// Find the image files and create a map from base file name without ext to ext.
	imageFiles, err := filesByExtInDir(imageDir, "")
	if err != nil {
		return nil, err
	}
	imageNamesToExt := mapFileNamesToExtensions(imageFiles)

	// Read the label files and store into the in-memory struct.
	data := make([]AnnotatedFile, 0, len(labelFiles))
	for _, path := range labelFiles {
		// Parse the file.
		lines, err := readLines(path)
		if err != nil {
			log.Printf("Error while parsing, skipping %q: %v", path, err)
			continue
		}

		annotations := make([]Annotation, 0, len(lines))
		for i := 0; i < len(lines); i++ {
			a, err := parseKittiAnnotation(lines[i])
			if err != nil {
				log.Printf("Error while parsing, skipping %q: %v", path, err)
				continue
			}
			annotation := Annotation{Coords: a.Coords, Label: a.Label}
			annotations = append(annotations, annotation)
		}

		// Find the corresponding image.
		_, baseNoExt, _, err := splitPath(path)
		if err != nil {
			log.Print(err)
			continue
		}
		imageExt, found := imageNamesToExt[baseNoExt]
		if !found {
			log.Print("Could not find the corresponding image file, skipping ", path)
			continue
		}
		imagePath := filepath.Join(imageDir, baseNoExt+"."+imageExt)

		data = append(data, AnnotatedFile{Annotations: annotations, FilePath: imagePath})
	}

	return data, nil
}

// parseKittiAnnotation parses the line of values for a single annotation.
func parseKittiAnnotation(line string) (KITTIAnnotation, error) {
	a := KITTIAnnotation{}

	tokens := strings.Split(line, " ")
	if len(tokens) < 8 {
		return a, fmt.Errorf("insufficient tokens in %q", line)
	}

	a.Label = tokens[0]
	var err error
	for i := 4; i < 8 && err == nil; i++ {
		a.Coords[i-4], err = strconv.ParseFloat(tokens[i], 64)
	}
	if err != nil {
		return a, fmt.Errorf("unexpected values in %q: %v", line, err)
	}

	// Parse the optional confidence score.
	if len(tokens) >= 16 {
		a.Score, err = strconv.ParseFloat(tokens[15], 64)
	}
	if err != nil {
		return a, fmt.Errorf("unexpected score format in %q: %v", line, err)
	}

	return a, nil
}

// ToKitti converts the intermediate representation to KITTI format.
func ToKitti(data []AnnotatedFile) []KITTIAnnotatedFile {
	kittiData := make([]KITTIAnnotatedFile, 0, len(data))
	for _, fileData := range data {
		// Per file data.
		kittiFileData := KITTIAnnotatedFile{
			Annotations: make([]KITTIAnnotation, len(fileData.Annotations)),
			FilePath:    fileData.FilePath,
		}
		// Convert all annotations.
		for i, a := range fileData.Annotations {
			kittiLabel := KITTIAnnotation{Coords: a.Coords, Label: a.Label}

			// Add the optional score.
			if score, ok := a.Attributes[Confidence].(float64); ok {
				kittiLabel.Score = score
			}

			kittiFileData.Annotations[i] = kittiLabel
		}
		kittiData = append(kittiData, kittiFileData)
	}

	return kittiData
}

// WriteKitti writes data to dirPath, one file per element.
func WriteKitti(dirPath string, data []KITTIAnnotatedFile) error {
	dirInfo, err := os.Stat(dirPath)
	if err != nil || !dirInfo.IsDir() {
		return fmt.Errorf("cannot access directory %q: %v", dirPath, err)
	}

	labelDirWithSep := dirPath + string(os.PathSeparator)
	for _, fileData := range data {
		// Use the image file name with .txt extension as label file name.
		_, baseNoExt, _, err := splitPath(fileData.FilePath)
		if err != nil {
			return err
		}
		filePath := labelDirWithSep + baseNoExt + ".txt"
		file, err := os.Create(filePath)
		if err != nil {
			return err
		}

		// Write annotations to file.
		for _, a := range fileData.Annotations {
			_, err = fmt.Fprintf(file,
				"%s 0.0 0 0.0 %.2f %.2f %.2f %.2f 0.0 0.0 0.0 0.0 0.0 0.0 0.0 %f\n",
				a.Label, a.Coords[0], a.Coords[1], a.Coords[2], a.Coords[3], a.Score)
			if err != nil {
				return err
			}
		}

		if err := file.Close(); err != nil {
			return err
		}
	}

	return nil
}

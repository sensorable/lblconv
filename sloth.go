package lblconv

// Sloth specific functionality.

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// SlothAnnotation is a single annotation within a Sloth file.
type SlothAnnotation struct {
	Class  string  `json:"class,omitempty"`
	Type   string  `json:"type,omitempty"`
	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
	Width  float64 `json:"width,omitempty"`
	Height float64 `json:"height,omitempty"`
}

// SlothAnnotatedFile defines the Sloth annotation structure for a single file.
type SlothAnnotatedFile struct {
	Annotations []SlothAnnotation `json:"annotations"`
	Class       string            `json:"class,omitempty"`
	FilePath    string            `json:"filename,omitempty"`
}

// FromSloth reads and parses Sloth annotations from the file at path.
func FromSloth(path string) ([]AnnotatedFile, error) {
	enc, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var slothData []SlothAnnotatedFile
	err = json.Unmarshal(enc, &slothData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Sloth input from %q: %v", path, err)
	}

	// Convert to the intermediate representation.
	data := make([]AnnotatedFile, 0, len(slothData))
	for _, slothFileData := range slothData {
		// Per file data. Convert all annotations.
		fileData := AnnotatedFile{
			Annotations: make([]Annotation, len(slothFileData.Annotations)),
			FilePath:    slothFileData.FilePath,
		}
		for i, a := range slothFileData.Annotations {
			annotation := Annotation{Label: a.Class}
			annotation.Coords[0] = a.X
			annotation.Coords[1] = a.Y
			annotation.Coords[2] = a.X + a.Width
			annotation.Coords[3] = a.Y + a.Height
			fileData.Annotations[i] = annotation
		}
		data = append(data, fileData)
	}

	return data, nil
}

// ToSloth converts the intermediate representation to Sloth format.
func ToSloth(data []AnnotatedFile) []SlothAnnotatedFile {
	slothData := make([]SlothAnnotatedFile, 0, len(data))
	for _, fileData := range data {
		slothFileData := SlothAnnotatedFile{
			Annotations: make([]SlothAnnotation, len(fileData.Annotations)),
			Class:       "image",
			FilePath:    fileData.FilePath,
		}
		for i, a := range fileData.Annotations {
			slothLabel := SlothAnnotation{
				Class:  a.Label,
				Type:   "rect",
				X:      a.Coords[0],
				Y:      a.Coords[1],
				Width:  a.Coords[2] - a.Coords[0],
				Height: a.Coords[3] - a.Coords[1],
			}
			slothFileData.Annotations[i] = slothLabel
		}
		slothData = append(slothData, slothFileData)
	}

	return slothData
}

// WriteSloth writes the Sloth annotations to outFile.
func WriteSloth(outFile string, data []SlothAnnotatedFile) error {
	enc, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(outFile, enc, 0644); err != nil {
		return fmt.Errorf("cannot write file %q: %v", outFile, err)
	}
	return nil
}

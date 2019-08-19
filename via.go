package lblconv

// VGG Immage Annotator (VIA) specific functionality.

import (
	"encoding"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
)

// VIAShape describes the shape of an annotation.
type VIAShape struct {
	Name   string `json:"name"`
	X      int32  `json:"x"`
	Y      int32  `json:"y"`
	Width  int32  `json:"width"`
	Height int32  `json:"height"`
}

// VIARegionAnnotation is a single region annotation for a particular image in a VIA file.
type VIARegionAnnotation struct {
	Attributes map[string]string `json:"region_attributes"`
	Shape      VIAShape          `json:"shape_attributes"`
}

// VIAAnnotatedFile defines the VIA annotation structure for a single file.
type VIAAnnotatedFile struct {
	Annotations []VIARegionAnnotation `json:"regions"`
	Attributes  map[string]string     `json:"file_attributes"`
	FilePath    string                `json:"filename"`
	Size        int64                 `json:"size"`
}

// VIAOptionsAttribute defines attributes of type "radio" or "dropdown".
type VIAOptionsAttribute struct {
	Type           string            `json:"type"` // "radio" or "dropdown"
	Description    string            `json:"description"`
	Options        map[string]string `json:"options"`
	DefaultOptions map[string]bool   `json:"default_options"`
}

// VIATextAttribute defines attributes of type "text".
type VIATextAttribute struct {
	Type         string `json:"type"` // "text"
	Description  string `json:"description"`
	DefaultValue string `json:"default_value"`
}

// VIAAttributes defines the VIA attribute metadata.
type VIAAttributes struct {
	Region map[string]interface{} `json:"region"`
	File   map[string]interface{} `json:"file"`
}

// VIAProject defines the VIA project structure.
type VIAProject struct {
	Attributes    VIAAttributes               `json:"_via_attributes"`
	ImageMetadata map[string]VIAAnnotatedFile `json:"_via_img_metadata"`
	// Must exist for VIA to load the project. Default values will be used.
	Settings struct{} `json:"_via_settings"`
}

const viaLabelAttribute = "Label" // The attribute key used for labels.

// FromVIA reads and parses VIA annotations from the file at path.
func FromVIA(path string) ([]AnnotatedFile, error) {
	enc, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var viaData VIAProject
	err = json.Unmarshal(enc, &viaData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse VIA input from %q: %v", path, err)
	}

	// Convert to the intermediate representation.
	irData := make([]AnnotatedFile, 0, len(viaData.ImageMetadata))
	for _, viaFile := range viaData.ImageMetadata {
		// Per file data. Convert all annotations.
		irFile := AnnotatedFile{
			Annotations: make([]Annotation, 0, len(viaFile.Annotations)),
			FilePath:    viaFile.FilePath,
		}
		for _, a := range viaFile.Annotations {
			irObject := Annotation{}

			// Set the label and other attributes.
			if _, haveLabel := a.Attributes[viaLabelAttribute];
					haveLabel && len(a.Attributes) > 1 || len(a.Attributes) > 0 {
				irObject.Attributes = make(map[string]interface{})
			}
			for k, v := range a.Attributes {
				switch k {
				case viaLabelAttribute:
					irObject.Label = v
				case Confidence: // float64
					if v, err := strconv.ParseFloat(v, 64); err == nil {
						irObject.Attributes[k] = v
					} else {
						log.Printf("Failed to parse attribute %q as float: %v", k, err)
					}
				default:
					irObject.Attributes[k] = v
				}
			}

			// Set the bounding box.
			irObject.Coords[0] = float64(a.Shape.X)
			irObject.Coords[1] = float64(a.Shape.Y)
			irObject.Coords[2] = float64(a.Shape.X + a.Shape.Width)
			irObject.Coords[3] = float64(a.Shape.Y + a.Shape.Height)

			irFile.Annotations = append(irFile.Annotations, irObject)
		}
		irData = append(irData, irFile)
	}

	return irData, nil
}

// ToVIA converts the intermediate representation to VIA format.
func ToVIA(irData []AnnotatedFile) VIAProject {
	viaData := VIAProject{
		Attributes: VIAAttributes{
			Region: make(map[string]interface{}),
			File:   make(map[string]interface{}),
		},
		ImageMetadata: make(map[string]VIAAnnotatedFile, len(irData)),
	}

	// Adds an option to a VIAOptionsAttribute, creating the attribute if necessary.
	addAttrOption := func(attrs map[string]interface{}, attrName, attrType, option string) {
		var attr VIAOptionsAttribute
		if a, ok := attrs[attrName]; ok {
			// Copy the existing attribute.
			if v, ok := a.(VIAOptionsAttribute); ok && v.Type == attrType {
				attr = v
			} else {
				log.Printf("Invalid type %T, expected VIAOptionsAttribute", a)
				return
			}
		} else {
			// Create a new attribute.
			attr = VIAOptionsAttribute{
				Type:           attrType,
				Options:        make(map[string]string),
				DefaultOptions: make(map[string]bool, 0),
			}
		}

		// Add the option value and copy the attribute back into the map.
		attr.Options[option] = ""
		attrs[attrName] = attr
	}

	var haveTextAttr, haveConfidenceAttr bool
	for _, irFile := range irData {
		viaFile := VIAAnnotatedFile{
			Annotations: make([]VIARegionAnnotation, 0, len(irFile.Annotations)),
			Attributes:  make(map[string]string, 0), // Must not be nil as that becomes JSON null.
			FilePath:    irFile.FilePath,
		}
		for _, a := range irFile.Annotations {
			viaObject := VIARegionAnnotation{
				Attributes: map[string]string{viaLabelAttribute: a.Label},
				Shape: VIAShape{
					Name:   "rect",
					X:      int32(a.Coords[0]),
					Y:      int32(a.Coords[1]),
					Width:  int32(a.Coords[2] - a.Coords[0]),
					Height: int32(a.Coords[3] - a.Coords[1]),
				},
			}

			// Add additional attributes with string values or values that can be converted to string.
			for k, v := range a.Attributes {
				switch v := v.(type) {
				case int:
					viaObject.Attributes[k] = strconv.Itoa(v)
				case float64:
					viaObject.Attributes[k] = strconv.FormatFloat(v, 'f', -1, 64)
				case string:
					viaObject.Attributes[k] = v
				case encoding.TextMarshaler:
					if s, err := v.MarshalText(); err == nil {
						viaObject.Attributes[k] = string(s)
					} else {
						log.Printf("Failed to marshal text for %s: %v", k, v)
					}
				}
			}

			// Add the label value to the attribute metadata.
			addAttrOption(viaData.Attributes.Region, viaLabelAttribute, "radio", a.Label)

			// Add attribute metadata for DetectedText and Confidence if they are part of the annotation.
			if !haveTextAttr {
				_, haveTextAttr = a.Attributes[DetectedText]
				if haveTextAttr {
					viaData.Attributes.Region[DetectedText] = VIATextAttribute{Type: "text"}
				}
			}
			if !haveConfidenceAttr {
				_, haveConfidenceAttr = a.Attributes[Confidence]
				if haveConfidenceAttr {
					viaData.Attributes.Region[Confidence] = VIATextAttribute{Type: "text"}
				}
			}

			viaFile.Annotations = append(viaFile.Annotations, viaObject)
		}
		viaData.ImageMetadata[viaFile.FilePath] = viaFile
	}

	return viaData
}

// WriteVIA writes the VIA project data to outFile.
func WriteVIA(outFile string, data VIAProject) error {
	enc, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(outFile, enc, 0644); err != nil {
		return fmt.Errorf("cannot write file %q: %v", outFile, err)
	}
	return nil
}

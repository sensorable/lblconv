package lblconv

// TFRecord object detection specific functionality.

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"

	"github.com/golang/protobuf/proto"
	"github.com/ryszard/tfutils/go/example"
	"github.com/ryszard/tfutils/go/tfrecord"
	"github.com/ryszard/tfutils/proto/tensorflow/core/example" // package tensorflow
	protos "github.com/sensorable/lblconv/protos"
)

// TFFeatureMap maps feature names to their values. Values must be convertible to
// tensorflow.Feature.
type TFFeatureMap map[string]interface{}

// TFRecordAnnotatedFile defines the TFRecord annotation structure for a single file.
type TFRecordAnnotatedFile struct {
	Annotations TFFeatureMap
	FilePath    string
}

var (
	tfRecordLabelMap    map[string]int32 // The active label mappings.
	tfRecordNextLabelID int32 = 1        // The ID for the next label mapping.
)

// toTFRecord converts the intermediate representation for a single file to the TFRecord format.
func toTFRecord(fileData AnnotatedFile) (TFRecordAnnotatedFile, error) {
	// Get the image width and height.
	img, format, err := decodeImageConfig(fileData.FilePath)
	if err != nil {
		return TFRecordAnnotatedFile{}, fmt.Errorf("failed to decode the image metadata: %v", err)
	}

	// Read the image data.
	imgData, err := readFile(fileData.FilePath)
	if err != nil {
		return TFRecordAnnotatedFile{}, fmt.Errorf("failed to read the image: %v", err)
	}

	// Prepare the feature map for the per file data.
	f := make(map[string]interface{}, 16)
	f["image/height"] = img.Height
	f["image/width"] = img.Width
	f["image/filename"] = fileData.FilePath
	f["image/source_id"] = fileData.FilePath
	f["image/encoded"] = imgData
	f["image/format"] = format

	// Prepare the per label data.
	numLabels := len(fileData.Annotations)
	xmins := make([]float32, numLabels)
	ymins := make([]float32, numLabels)
	xmaxs := make([]float32, numLabels)
	ymaxs := make([]float32, numLabels)
	classes := make([]string, numLabels)
	classIDs := make([]int64, numLabels)
	for i, a := range fileData.Annotations {
		xmins[i] = float32(a.Coords[0]) / float32(img.Width)
		ymins[i] = float32(a.Coords[1]) / float32(img.Height)
		xmaxs[i] = float32(a.Coords[2]) / float32(img.Width)
		ymaxs[i] = float32(a.Coords[3]) / float32(img.Height)
		classes[i] = a.Label

		// Assign the ID for the string label, selecting a new one if no mapping exists.
		classIDs[i] = int64(tfRecordLabelMap[a.Label])
		if classIDs[i] == 0 {
			tfRecordLabelMap[a.Label] = tfRecordNextLabelID
			classIDs[i] = int64(tfRecordNextLabelID)
			tfRecordNextLabelID++
		}
	}
	f["image/object/bbox/xmin"] = xmins
	f["image/object/bbox/ymin"] = ymins
	f["image/object/bbox/xmax"] = xmaxs
	f["image/object/bbox/ymax"] = ymaxs
	f["image/object/class/text"] = classes
	f["image/object/class/label"] = classIDs

	// Create the example.
	return TFRecordAnnotatedFile{
		Annotations: f,
		FilePath:    fileData.FilePath,
	}, nil
}

// WriteCustomTFRecord works like WriteTFRecord, except that it allows for the TFFeatureMap to be
// customised.
//
// Before generating a tensorflow.Example from each AnnotatedFile and writing it to the TFRecord
// file, the source data and TFFeatureMap containing the default conversion for object records are
// passed to customiseFeature, which may modify the feature map to its liking, as long as all of its
// values can be converted to tensorflow.Feature.
func WriteCustomTFRecord(recordFilePath, labelMapPath string, data []AnnotatedFile,
		numShards int, customiseFeature func(f AnnotatedFile, m TFFeatureMap)) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("conversion to TensorFlow Example failed: %v", e)
		}
	}()

	if numShards <= 0 {
		numShards = 1
	}

	if tfRecordLabelMap == nil {
		// Try to load an existing label map. It is not an error if the file does not exist.
		if labelMap, maxID, err := loadTFRecordLabelMap(labelMapPath); err == nil {
			log.Print("Label map loaded successfully")
			tfRecordLabelMap = labelMap
			tfRecordNextLabelID = maxID + 1
		} else if os.IsNotExist(err) {
			log.Print("Creating a new label map")
			tfRecordLabelMap = make(map[string]int32)
			tfRecordNextLabelID = 1
		} else {
			return fmt.Errorf("failed to read the label map from %q: %v", labelMapPath, err)
		}
	}

	fmtShardSuffix := func(idx int) string {
		return fmt.Sprintf("-%05d-of-%05d", idx, numShards)
	}

	var shardFile *os.File
	shardSize := int(math.Ceil(float64(len(data)) / float64(numShards)))
	shardIdx := -1

	// Convert and serialise one data element at a time.
	for i, fileData := range data {
		// Check if a new shard file needs to be opened for writing.
		if i%shardSize == 0 {
			shardIdx++

			// Close the previous shard file.
			if shardFile != nil {
				_ = shardFile.Close()
				shardFile = nil
			}

			// Create the new shard file.
			shardPath := recordFilePath
			if numShards > 1 {
				shardPath += fmtShardSuffix(shardIdx)
			}
			f, err := os.Create(shardPath)
			if err != nil {
				return fmt.Errorf("failed to create shard at %q: %v", shardPath, err)
			}
			shardFile = f
		}

		// Convert the file data to an example.
		tfFileData, err := toTFRecord(fileData)
		if err != nil {
			log.Printf("Failed to convert %q: %v", fileData.FilePath, err)
			continue
		}
		if customiseFeature != nil {
			customiseFeature(fileData, tfFileData.Annotations)
		}
		tfExample := example.New(tfFileData.Annotations)

		// Write the example.
		if err := writeTFRecordExample(shardFile, tfExample); err != nil {
			log.Print("Failed to write example: ", err)
			break
		}
	}

	if shardFile != nil {
		shardFile.Close()
	}

	return saveTFRecordLabelMap(labelMapPath, tfRecordLabelMap)
}

// WriteTFRecord does a streaming conversion, serialisation and file write for the annotation data
// to one or more TFRecord files stored under recordFilePath (with suffixes added when numShards>1).
//
// A label map is generated and written to labelMapPath.
func WriteTFRecord(recordFilePath, labelMapPath string, data []AnnotatedFile, numShards int) error {
	return WriteCustomTFRecord(recordFilePath, labelMapPath, data, numShards, nil)
}

// writeTFRecordExample serialises the example and writes it as a TFRecord to w.
func writeTFRecordExample(w io.Writer, e *tensorflow.Example) error {
	enc, err := proto.Marshal(e)
	if err != nil {
		return err
	}

	return tfrecord.Write(w, enc)
}

// saveTFRecordLabelMap converts the labelMap to prototxt format and writes it to path.
func saveTFRecordLabelMap(path string, labelMap map[string]int32) error {
	// Copy the label map into the protobuf structure.
	siLabelMap := &protos.StringIntLabelMap{}
	siLabelMap.Item = make([]*protos.StringIntLabelMapItem, 0, len(labelMap))
	for k, v := range labelMap {
		siLabelMap.Item = append(siLabelMap.Item, &protos.StringIntLabelMapItem{
			Name: proto.String(k),
			Id:   proto.Int32(v),
		})
	}

	// Write the label map.
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create the label map file %q: %v", path, err)
	}
	defer file.Close()

	if err := proto.MarshalText(file, siLabelMap); err != nil {
		return fmt.Errorf("failed to write the label map %q: %v", path, err)
	}

	return nil
}

// loadTFRecordLabelMap loads the label map from path. It also returns the largest ID value
// encountered in the map.
//
// If an error occurs because the file does not exist, then os.IsNotExist will return true for the
// error.
func loadTFRecordLabelMap(path string) (map[string]int32, int32, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	text, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, 0, err
	}

	var siLabelMap protos.StringIntLabelMap
	if err := proto.UnmarshalText(string(text), &siLabelMap); err != nil {
		return nil, 0, err
	}

	max := func(a, b int32) int32 {
		if a > b {
			return a
		}
		return b
	}

	labelMap := make(map[string]int32, len(siLabelMap.Item))
	var maxID int32
	for _, item := range siLabelMap.Item {
		k, v := item.GetName(), item.GetId()
		if k == "" || v <= 0 {
			return nil, 0, fmt.Errorf("invalid entry: %s: %d", k, v)
		}

		labelMap[k] = v
		maxID = max(maxID, v)
	}

	return labelMap, maxID, nil
}

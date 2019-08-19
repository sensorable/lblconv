package lblconv

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// filesByExtInDir retuns all regular files with file extension ext found directly in directory
// dirPath. All files are returned if extension is empty.
func filesByExtInDir(dirPath, ext string) (files []string, err error) {
	// Open the directory.
	dirInfo, err := os.Stat(dirPath)
	if err != nil || !dirInfo.IsDir() {
		return nil, fmt.Errorf("cannot read directory %q: %v: ", dirPath, err)
	}
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access %q: %v", dirPath, err)
	}
	defer closeWithErrCheck(dir, &err)

	pathWithSep := dirPath
	if !strings.HasSuffix(dirPath, string(os.PathSeparator)) {
		pathWithSep = dirPath + string(os.PathSeparator)
	}

	// Iterate over all files in dir.
	files = make([]string, 0, 100)
	var fileList []os.FileInfo
	for fileList, err = dir.Readdir(100); len(fileList) > 0; fileList, err = dir.Readdir(100) {
		for _, file := range fileList {
			name := file.Name()
			filePath := pathWithSep + name
			// Must be a regular file or a symlink and have the requested extension/suffix.
			if (!file.Mode().IsRegular() && (file.Mode()&os.ModeSymlink == 0)) ||
					!strings.HasSuffix(name, ext) {
				continue
			}
			files = append(files, filePath)
		}
	}
	if err != nil && err != io.EOF {
		log.Printf("Failed to access some files in %q: %v", dirPath, err)
	}

	return files, nil
}

// splitPath splits the given file path into the dir name, the base name without extension and the
// extension (without the dot).
func splitPath(path string) (dir, baseNoExt, ext string, err error) {
	dir, file := filepath.Split(path)
	ext = filepath.Ext(file)
	if ext == "" {
		return "", "", "", fmt.Errorf("missing file extension in %q", path)
	}

	dir = strings.TrimSuffix(dir, string(os.PathSeparator))
	baseNoExt = file[0 : len(file)-len(ext)]
	ext = ext[1:]

	return dir, baseNoExt, ext, nil
}

// mapFileNamesToExtensions maps the base names of the given file paths, with the file type
// extensions stripped off, to the file extension (without the dot).
func mapFileNamesToExtensions(filePaths []string) map[string]string {
	mapping := make(map[string]string, len(filePaths))
	for _, path := range filePaths {
		_, baseNoExt, ext, err := splitPath(path)
		if err != nil {
			log.Print(err)
			continue
		}
		mapping[baseNoExt] = ext
	}

	return mapping
}

// labelParserFn parses a label file given the label and image file paths.
type labelParserFn func(labelPath, imagePath string) (AnnotatedFile, error)

// parseLabelsWithOneToOneImages matches label files in labelDir, with file extension labelFileExt
// (e.g. ".json") by file name to images in imageDir (with an arbitrary file extension). It then
// invokes labelParserFn on these path pairs.
//
// Returns the list of file annotations obtained by applying labelParserFn to all label files.
func parseLabelsWithOneToOneImages(labelDir, labelFileExt, imageDir string, parse labelParserFn) (
		[]AnnotatedFile, error) {

	// Get the label file paths.
	labelFiles, err := filesByExtInDir(labelDir, labelFileExt)
	if err != nil {
		return nil, err
	}
	log.Printf("Parsing labels for %d files", len(labelFiles))

	// Find the image files and create a map from base file name without ext to ext.
	imageFiles, err := filesByExtInDir(imageDir, "")
	if err != nil {
		return nil, err
	}
	imageNamesToExt := mapFileNamesToExtensions(imageFiles)

	data := make([]AnnotatedFile, 0, len(labelFiles))
	for _, labelPath := range labelFiles {
		// Find the corresponding image.
		_, baseNoExt, _, err := splitPath(labelPath)
		if err != nil {
			log.Printf("Error while parsing, skipping %q: %v", labelPath, err)
			continue
		}
		imageExt, found := imageNamesToExt[baseNoExt]
		if !found {
			log.Printf("No corresponding image file, skipping %q", labelPath)
			continue
		}
		imagePath := filepath.Join(imageDir, baseNoExt+"."+imageExt)

		// Parse the label file.
		fileData, err := parse(labelPath, imagePath)
		if err != nil {
			log.Printf("Error while parsing, skipping %q: %v", labelPath, err)
			continue
		}

		data = append(data, fileData)
	}

	return data, nil
}

// readLines returns a slice of lines read from the file at path.
func readLines(path string) (lines []string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read file %q: %v", path, err)
	}
	defer closeWithErrCheck(file, &err)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read %q as lines: %v", path, err)
	}

	return lines, nil
}

// readFile uses ioutil.ReadAll to read the file at path.
func readFile(path string) (data []byte, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer closeWithErrCheck(f, &err)

	data, err = ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// closeWithErrCheck calls c.Close(). If it returns an error, and (*e == nil), e is set to that
// error.
func closeWithErrCheck(c io.Closer, e *error) {
	err := c.Close()
	if err != nil && *e == nil {
		*e = err
	}
}

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

func unmarshalReleaseManifest(header *tar.Header, reader *tar.Reader) (ReleaseManifest, error) {
	releaseManifest := ReleaseManifest{}

	data := make([]byte, header.Size)
	_, err := io.ReadFull(reader, data)
	if err != nil {
		return releaseManifest, fmt.Errorf("Error reading 'release.MF'")
	}

	err = yaml.Unmarshal(data, &releaseManifest)
	if err != nil {
		return releaseManifest, fmt.Errorf("Error unmarshaling 'release.MF'")
	}

	return releaseManifest, nil
}

func unmarshalJobManifest(header *tar.Header, reader *tar.Reader) (ReleaseJobManifest, error) {
	jobManifest := ReleaseJobManifest{}

	data := make([]byte, header.Size)
	_, err := io.ReadFull(reader, data)
	if err != nil {
		return jobManifest, fmt.Errorf("Error reading 'job.MF'")
	}

	err = yaml.Unmarshal(data, &jobManifest)
	if err != nil {
		return jobManifest, fmt.Errorf("Error unmarshaling 'job.MF'")
	}

	return jobManifest, nil

}

func extractJobManifest(header *tar.Header, reader *tar.Reader) (ReleaseJobManifest, error) {
	jobManifest := ReleaseJobManifest{}

	data := make([]byte, header.Size)
	_, err := io.ReadFull(reader, data)
	if err != nil {
		return jobManifest, fmt.Errorf("Error reading '%s'", header.Name)
	}

	buffer := bytes.NewBuffer(data)
	gzipReader, err := gzip.NewReader(buffer)
	if err != nil {
		return jobManifest, err
	}

	tarReader := tar.NewReader(gzipReader)
	for true {
		tarHeader, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return jobManifest, err
		}

		if tarHeader.Typeflag != tar.TypeReg {
			continue
		}

		if !strings.HasPrefix(tarHeader.Name, "./job.MF") {
			continue
		}

		jobManifest, err := unmarshalJobManifest(tarHeader, tarReader)
		if err != nil {
			return jobManifest, err
		}

		return jobManifest, nil

	}

	return jobManifest, fmt.Errorf("Did not find 'job.MF' file instead '%s'", header.Name)
}

func exploreReleaseMetadata(releasePath string) (*ReleaseMetadata, error) {
	releaseFile, err := os.Open(releasePath)
	if err != nil {
		return nil, err
	}
	defer releaseFile.Close()

	gzipReader, err := gzip.NewReader(releaseFile)
	if err != nil {
		return nil, err
	}

	tarReader := tar.NewReader(gzipReader)

	releaseMetadata := &ReleaseMetadata{}

	for true {
		tarHeader, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		if tarHeader.Typeflag != tar.TypeReg {
			continue
		}

		if tarHeader.Name == "./release.MF" {
			releaseManifest, err := unmarshalReleaseManifest(tarHeader, tarReader)
			if err != nil {
				return nil, err
			}

			releaseMetadata.Manifest = releaseManifest
		}

		if strings.HasPrefix(tarHeader.Name, "./packages") {
			packageFile := ReleaseFile{
				Path: tarHeader.Name,
				Size: tarHeader.Size,
			}
			releaseMetadata.PackageFiles = append(releaseMetadata.PackageFiles, packageFile)
		}

		if strings.HasPrefix(tarHeader.Name, "./jobs") {
			jobFile := ReleaseFile{
				Path: tarHeader.Name,
				Size: tarHeader.Size,
			}
			releaseMetadata.JobFiles = append(releaseMetadata.JobFiles, jobFile)

			releaseJobManifest, err := extractJobManifest(tarHeader, tarReader)
			if err != nil {
				return nil, err
			}
			releaseMetadata.JobManifests = append(releaseMetadata.JobManifests, releaseJobManifest)
		}
	}

	return releaseMetadata, nil
}

func main() {
	filePath := flag.String("filePath", "", "Release path to explore")
	flag.Parse()

	if _, err := os.Stat(*filePath); os.IsNotExist(err) {
		panic(fmt.Errorf("File '%s' does not exist!", *filePath))
	}

	releaseMetadata, err := exploreReleaseMetadata(*filePath)
	if err != nil {
		panic(err)
	}

	releaseYAML, err := yaml.Marshal(releaseMetadata)

	fmt.Println(string(releaseYAML))

	for _, jrf := range releaseMetadata.JobFiles {
		fmt.Printf("%s\n", jrf.Name())
		fmt.Printf("%s\n", jrf.HumanReadableSize())
	}
	for _, prf := range releaseMetadata.PackageFiles {
		fmt.Printf("%s\n", prf.Name())
		fmt.Printf("%s\n", prf.HumanReadableSize())
	}
}

func (rm ReleaseManifest) printPackages() {
	fmt.Printf("digraph packages {\n")
	var allNodes []string
	for _, pkg := range rm.Packages {
		allNodes = append(allNodes, pkg.Name)
		if len(pkg.Dependencies) > 0 {
			fmt.Printf("  \"%s\" -> { \"%s\" }\n", pkg.Name, strings.Join(pkg.Dependencies, "\" \""))
		}
	}
	fmt.Printf("{ rank=same; \"%s\"}\n", strings.Join(allNodes, "\"; \""))
	fmt.Printf("\"%s\"\n", strings.Join(allNodes, "\" -> \""))
	fmt.Printf("}\n")
}

type ReleaseMetadata struct {
	Manifest     ReleaseManifest
	PackageFiles []ReleaseFile
	JobFiles     []ReleaseFile
	JobManifests []ReleaseJobManifest
}

// <release.MF>
type ReleaseManifest struct {
	Packages []ReleaseManifestPackage `yaml:"packages"`
	Jobs     []ReleaseManifestJob     `yaml:"jobs"`
}

type ReleaseManifestPackage struct {
	Name         string   `yaml:"name"`
	SHA1         string   `yaml:"sha1"`
	Fingerprint  string   `yaml:"fingerprint"`
	Version      string   `yaml:"version"`
	Dependencies []string `yaml:"dependencies"`
}

type ReleaseManifestJob struct {
	Name        string `yaml:"name"`
	SHA1        string `yaml:"sha1"`
	Fingerprint string `yaml:"fingerprint"`
	Version     string `yaml:"version"`
}

// </release.MF>

type ReleaseJobManifest struct {
	Name     string   `yaml:"name"`
	Packages []string `yaml:"packages"`
}

type ReleaseFile struct {
	Path string
	Size int64
}

func (rf ReleaseFile) Name() string {
	filename := path.Base(rf.Path)
	return strings.TrimSuffix(filename, path.Ext(filename))
}

func (rf ReleaseFile) HumanReadableSize() string {
	sizeInBytes := float64(rf.Size)
	sizeInKiloBytes := sizeInBytes / 1024
	sizeInMegaBytes := sizeInKiloBytes / 1024

	if sizeInKiloBytes < 1 {
		return fmt.Sprintf("%0.2f B", sizeInBytes)
	}

	if sizeInMegaBytes < 1 {
		return fmt.Sprintf("%0.2f KB", sizeInKiloBytes)
	}

	return fmt.Sprintf("%0.2f MB", sizeInMegaBytes)
}

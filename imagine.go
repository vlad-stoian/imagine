package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

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
}

func extractReleaseManifest(header *tar.Header, reader *tar.Reader) (ReleaseManifest, error) {
	releaseManifest := ReleaseManifest{}

	data := make([]byte, header.Size)
	_, err := reader.Read(data)
	if err != nil {
		return releaseManifest, fmt.Errorf("Error reading 'release.MF'")
	}

	err = yaml.Unmarshal(data, &releaseManifest)
	if err != nil {
		return releaseManifest, fmt.Errorf("Error unmarshaling 'release.MF'")
	}

	return releaseManifest, nil
}

func exploreReleaseMetadata(releasePath string) (*ReleaseMetadata, error) {
	releaseFile, err := os.Open(releasePath)
	if err != nil {
		return nil, err
	}
	defer releaseFile.Close()

	releaseFileGZip, err := gzip.NewReader(releaseFile)
	if err != nil {
		return nil, err
	}

	tarReader := tar.NewReader(releaseFileGZip)

	releaseMetadata := &ReleaseMetadata{}

	for true {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		if header.Name == "./release.MF" {
			releaseManifest, err := extractReleaseManifest(header, tarReader)
			if err != nil {
				return nil, err
			}

			releaseMetadata.Manifest = releaseManifest
		}

		if strings.HasPrefix(header.Name, "./packages") {
			packageFile := ReleaseFile{
				Path: header.Name,
				Size: header.Size,
			}
			releaseMetadata.PackageFiles = append(releaseMetadata.PackageFiles, packageFile)
		}

		if strings.HasPrefix(header.Name, "./jobs") {
			jobFile := ReleaseFile{
				Path: header.Name,
				Size: header.Size,
			}
			releaseMetadata.JobFiles = append(releaseMetadata.JobFiles, jobFile)
		}
	}

	return releaseMetadata, nil
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

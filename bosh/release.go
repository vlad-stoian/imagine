package bosh

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

// ### release.MF ###

type ReleaseManifest struct {
	Name     string                   `yaml:"name"`
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

// ### /release.MF ###

func UnmarshalReleaseManifest(header *tar.Header, reader *tar.Reader) (ReleaseManifest, error) {
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

// ### job.MF ###

type JobManifest struct {
	Name     string   `yaml:"name"`
	Packages []string `yaml:"packages"`
}

// ### /job.MF ###

func UnmarshalJobManifest(header *tar.Header, reader *tar.Reader) (JobManifest, error) {
	jobManifest := JobManifest{}

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

func ExtractJobManifest(header *tar.Header, reader *tar.Reader) (JobManifest, error) {
	jobManifest := JobManifest{}

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

		jobManifest, err := UnmarshalJobManifest(tarHeader, tarReader)
		if err != nil {
			return jobManifest, err
		}

		return jobManifest, nil

	}

	return jobManifest, fmt.Errorf("Did not find 'job.MF' file instead '%s'", header.Name)
}

func ExploreReleaseMetadata(releasePath string) (ReleaseMetadata, error) {
	releaseMetadata := ReleaseMetadata{}

	releaseFile, err := os.Open(releasePath)
	if err != nil {
		return releaseMetadata, err
	}
	defer releaseFile.Close()

	gzipReader, err := gzip.NewReader(releaseFile)
	if err != nil {
		return releaseMetadata, err
	}

	tarReader := tar.NewReader(gzipReader)

	for true {
		tarHeader, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return releaseMetadata, err
		}

		if tarHeader.Typeflag != tar.TypeReg {
			continue
		}

		if tarHeader.Name == "./release.MF" {
			releaseManifest, err := UnmarshalReleaseManifest(tarHeader, tarReader)
			if err != nil {
				return releaseMetadata, err
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

			releaseJobManifest, err := ExtractJobManifest(tarHeader, tarReader)
			if err != nil {
				return releaseMetadata, err
			}
			releaseMetadata.JobManifests = append(releaseMetadata.JobManifests, releaseJobManifest)
		}
	}

	return releaseMetadata, nil
}

type ReleaseMetadata struct {
	Manifest     ReleaseManifest `yaml:"manifest"`
	PackageFiles []ReleaseFile   `yaml:"package_files"`
	JobFiles     []ReleaseFile   `yaml:"job_files"`
	JobManifests []JobManifest   `yaml:"job_manifests"`
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

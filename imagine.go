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

	"github.com/awalterschulze/gographviz"
	"github.com/vlad-stoian/imagine/graphviz"

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

func exploreReleaseMetadata(releasePath string) (ReleaseMetadata, error) {
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
			releaseManifest, err := unmarshalReleaseManifest(tarHeader, tarReader)
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

			releaseJobManifest, err := extractJobManifest(tarHeader, tarReader)
			if err != nil {
				return releaseMetadata, err
			}
			releaseMetadata.JobManifests = append(releaseMetadata.JobManifests, releaseJobManifest)
		}
	}

	return releaseMetadata, nil
}

var attributes graphviz.Attributes

func createSubGraph(graph *gographviz.Escape, subGraphName string, releaseFiles []ReleaseFile) {

	clusterName := fmt.Sprintf("cluster_%s", subGraphName)
	sameRankSubGraphName := fmt.Sprintf("same_rank_%s", subGraphName)

	_ = graph.AddSubGraph(graph.Name, clusterName, attributes.GetClusterAttrsWithName(subGraphName))
	_ = graph.AddSubGraph(clusterName, sameRankSubGraphName, attributes.GetSubGraphAttrs())

	for _, releaseFile := range releaseFiles {
		_ = graph.AddNode(sameRankSubGraphName, subGraphName+"-"+releaseFile.Name(), attributes.GetNodeAttrsWithNameAndSize(releaseFile.Name(), releaseFile.HumanReadableSize()))
	}
}

func JobPrefixedName(name string) string {
	return fmt.Sprintf("jobs-%s", name)
}

func PackagePrefixedName(name string) string {
	return fmt.Sprintf("packages-%s", name)
}

func createCrazyGraph(releaseMetadata ReleaseMetadata) string {
	releaseName := releaseMetadata.Manifest.Name

	// defaultGraphAttrs := map[string]string{
	// 	"rankdir":   "LR",
	// 	"nodeshape": "record",
	// }

	graph := gographviz.NewEscape()
	_ = graph.SetDir(true)
	_ = graph.SetName(releaseName)
	_ = graph.AddAttr(releaseName, "rankdir", "LR")
	_ = graph.AddAttr(releaseName, "nodesep", "0.5")
	_ = graph.AddAttr(releaseName, "ranksep", "2")

	createSubGraph(graph, "packages", releaseMetadata.PackageFiles)
	createSubGraph(graph, "jobs", releaseMetadata.JobFiles)

	for _, jobManifest := range releaseMetadata.JobManifests {
		for _, pkg := range jobManifest.Packages {
			_ = graph.AddEdge(JobPrefixedName(jobManifest.Name), PackagePrefixedName(pkg), true, attributes.GetEdgeAttrsJobToPackage())
		}
	}

	for _, pkg := range releaseMetadata.Manifest.Packages {
		for _, pkgDep := range pkg.Dependencies {
			_ = graph.AddEdge(PackagePrefixedName(pkg.Name), PackagePrefixedName(pkgDep), true, attributes.GetEdgeAttrsPackageToPackage())
		}
	}

	return graph.String()
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

	crazyGraph := createCrazyGraph(releaseMetadata)
	fmt.Println(crazyGraph)
}

type ReleaseMetadata struct {
	Manifest     ReleaseManifest      `yaml:"manifest"`
	PackageFiles []ReleaseFile        `yaml:"package_files"`
	JobFiles     []ReleaseFile        `yaml:"job_files"`
	JobManifests []ReleaseJobManifest `yaml:"job_manifests"`
}

// <release.MF>
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

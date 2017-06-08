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
		fmt.Println("File does not exist!")
		os.Exit(1)
	}

	exploreRelease(*filePath)
}

func exploreRelease(releasePath string) {

	f, err := os.Open(releasePath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()

	gzf, err := gzip.NewReader(f)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	tarReader := tar.NewReader(gzf)

	for true {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		name := header.Name
		// typeFlag := header.Typeflag

		if name == "./release.MF" {
			data := make([]byte, header.Size)
			_, err := tarReader.Read(data)
			if err != nil {
				panic("Error reading release.MF")
			}

			releaseManifest := ReleaseManifest{}

			_ = yaml.Unmarshal(data, &releaseManifest)

			releaseManifest.printPackages()
		}

		// switch typeFlag {
		// case tar.TypeDir:
		// 	fmt.Println("Dir: ", name)
		// case tar.TypeReg:
		// 	fmt.Println("File: ", name)
		// default:
		// 	fmt.Println("Unknown File Type")
		// }
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

type ReleaseManifestPackage struct {
	Name         string   `yaml:"name"`
	SHA1         string   `yaml:"sha"`
	Version      string   `yaml:"version"`
	Dependencies []string `yaml:"dependencies"`
}

type ReleaseManifest struct {
	Packages []ReleaseManifestPackage `yaml:"packages"`
}

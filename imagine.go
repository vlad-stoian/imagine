package main

import (
	"fmt"
	"os"

	"github.com/awalterschulze/gographviz"
	"github.com/vlad-stoian/imagine/bosh"
	"github.com/vlad-stoian/imagine/graphviz"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var attributes graphviz.Attributes

func createSubGraph(graph *gographviz.Escape, subGraphName string, releaseFiles []bosh.ReleaseFile) {
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

func createCrazyGraph(releaseMetadata bosh.ReleaseMetadata) string {
	releaseName := releaseMetadata.Manifest.Name

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

var (
	verbose     = kingpin.Flag("verbose", "Verbose mode").Short('v').Bool()
	releasePath = kingpin.Arg("release-path", "Path of the release file").Required().String()
)

func main() {
	kingpin.Parse()

	if _, err := os.Stat(*releasePath); os.IsNotExist(err) {
		panic(fmt.Errorf("File '%s' does not exist!", *releasePath))
	}

	releaseMetadata, err := bosh.ExploreReleaseMetadata(*releasePath)
	if err != nil {
		panic(err)
	}

	crazyGraph := createCrazyGraph(releaseMetadata)
	fmt.Println(crazyGraph)
}

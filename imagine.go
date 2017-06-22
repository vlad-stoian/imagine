package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/awalterschulze/gographviz"
	"github.com/vlad-stoian/imagine/bosh"
	"github.com/vlad-stoian/imagine/graphviz"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var attributes graphviz.Attributes

func addAllNodes(graph *gographviz.Escape, subGraphName string, releaseNodes map[string]ReleaseNode) {
	clusterName := fmt.Sprintf("cluster_%s", subGraphName)
	sameRankName := fmt.Sprintf("same_rank_%s", subGraphName)

	_ = graph.AddSubGraph(graph.Name, clusterName, attributes.GetClusterAttrsWithName(strings.Title(subGraphName)))
	_ = graph.AddSubGraph(clusterName, sameRankName, attributes.GetSubGraphAttrs())

	for nodeName := range releaseNodes {
		rn := releaseNodes[nodeName]

		_ = graph.AddNode(sameRankName, rn.ID(), attributes.GetNodeAttrsWithNameAndSize(rn.Name, rn.HumanReadableSize))
	}

}

type ReleaseNode struct {
	Name              string
	Path              string
	Size              int64
	HumanReadableSize string
	Type              string
}

func (rn ReleaseNode) ID() string {
	// fmt.Printf("%s-%s\n", rn.Type, rn.Name)
	return fmt.Sprintf("%s-%064b-%s", rn.Type, rn.Size, rn.Name)
}

func createCrazyNodes(nodeType string, releaseFiles []bosh.ReleaseFile) map[string]ReleaseNode {
	nodes := make(map[string]ReleaseNode)

	for _, rf := range releaseFiles {
		nodes[rf.Name()] = ReleaseNode{
			Name:              rf.Name(),
			Path:              rf.Path,
			Size:              rf.Size,
			HumanReadableSize: rf.HumanReadableSize(),
			Type:              nodeType,
		}
	}

	return nodes
}

func createCrazyGraph(releaseMetadata bosh.ReleaseMetadata) string {
	releaseName := releaseMetadata.Manifest.Name

	graph := gographviz.NewEscape()
	_ = graph.SetDir(true)
	_ = graph.SetName(releaseName)
	_ = graph.AddAttr(releaseName, "rankdir", "LR")
	_ = graph.AddAttr(releaseName, "nodesep", "0.5")
	_ = graph.AddAttr(releaseName, "ranksep", "2")

	jobNodes := createCrazyNodes("job", releaseMetadata.JobFiles)
	packageNodes := createCrazyNodes("package", releaseMetadata.PackageFiles)

	addAllNodes(graph, "jobs", jobNodes)
	addAllNodes(graph, "packages", packageNodes)

	for _, jobManifest := range releaseMetadata.JobManifests {
		for _, pkg := range jobManifest.Packages {
			jn := jobNodes[jobManifest.Name]
			pn := packageNodes[pkg]

			_ = graph.AddEdge(jn.ID(), pn.ID(), true, attributes.GetEdgeAttrsJobToPackage())
		}
	}

	for _, pkg := range releaseMetadata.Manifest.Packages {
		for _, pkgDep := range pkg.Dependencies {
			pnFrom := packageNodes[pkg.Name]
			pnTo := packageNodes[pkgDep]

			_ = graph.AddEdge(pnFrom.ID(), pnTo.ID(), true, attributes.GetEdgeAttrsPackageToPackage())
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

package graphviz

import "fmt"

type Attributes struct {
}

func (s Attributes) GetClusterAttrs() map[string]string {
	subGraphAttrs := map[string]string{
		"rank":  "same",
		"color": "blue",
		"style": "rounded",
	}

	return subGraphAttrs
}

func (s Attributes) GetClusterAttrsWithName(name string) map[string]string {
	subGraphAttrs := s.GetClusterAttrs()
	subGraphAttrs["label"] = name
	subGraphAttrs["fontsize"] = "32"

	return subGraphAttrs
}

func (s Attributes) GetSubGraphAttrs() map[string]string {
	subGraphAttrs := map[string]string{
		"rank": "same",
	}

	return subGraphAttrs
}

func (s Attributes) GetNodeAttrs() map[string]string {
	nodeAttrs := map[string]string{
		"shape": "Mrecord",
		"style": "striped",
		"color": "#ff000022;0.3:blue:yellow",
	}

	return nodeAttrs
}

func (s Attributes) GetNodeAttrsWithNameAndSize(name string, size string) map[string]string {
	nodeAttrs := s.GetNodeAttrs()
	nodeAttrs["label"] = fmt.Sprintf("{ %s | %s }", name, size)
	nodeAttrs["fontsize"] = "16"

	return nodeAttrs
}

func (s Attributes) GetEdgeAttrsJobToPackage() map[string]string {
	edgeAttrs := map[string]string{
		"arrowhead": "vee",
		"tailport":  "e",
		"headport":  "_w",
	}

	return edgeAttrs
}

func (s Attributes) GetEdgeAttrsPackageToPackage() map[string]string {
	edgeAttrs := s.GetEdgeAttrsJobToPackage()
	edgeAttrs["headport"] = "_e"
	edgeAttrs["color"] = "red"
	edgeAttrs["constraint"] = "true"

	return edgeAttrs
}

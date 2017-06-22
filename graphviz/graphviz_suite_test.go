package graphviz_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestGraphviz(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Graphviz Suite")
}

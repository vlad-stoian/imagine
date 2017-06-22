package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestImagine(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Imagine Suite")
}

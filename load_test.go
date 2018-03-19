package main

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/go-openapi/loads"
	"github.com/gotestyourself/gotestyourself/assert"
)

func TestLoad(t *testing.T) {
	doc, err := loads.Spec("swagger.yaml")
	assert.NilError(t, err)

	asSlice := []string{}
	for name := range doc.Spec().Definitions {
		asSlice = append(asSlice, name)
	}
	sort.Strings(asSlice)

	fmt.Println(strings.Join(asSlice, "\n"))
}

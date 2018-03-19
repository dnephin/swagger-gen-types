package main

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/go-openapi/spec"
)

func generateOperation(file *jen.File, opName, method, path string, op *spec.Operation) {
	for _, code := range sortResponses(op.Responses.StatusCodeResponses) {
		response := op.Responses.StatusCodeResponses[code]
		if response.Schema == nil {
			continue
		}

		schema := *response.Schema
		if schema.Ref.GetURL() != nil {
			// Skip references, they should be generated already as definitions
			continue
		}

		if len(schema.Type) > 1 {
			panic(fmt.Sprintf("%s: multi-type not supported yet", opName))
		}

		name := schema.Title
		if name == "" {
			name = opName + statusText(code) + "Response"
		}
		switch schema.Type[0] {
		case "object":
			addStatementsToFile(file, buildStructType(name, schema))
		case "array":
			addStatementsToFile(file, buildArrayType(name, schema))
		default:
			panic(fmt.Sprintf("%s: type %s not supported yet", opName, schema.Type))
		}
	}

	// TODO: parameters
}

func sortResponses(responses map[int]spec.Response) []int {
	var keys []int
	for key := range responses {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	return keys
}

func statusText(code int) string {
	return strings.Replace(http.StatusText(code), " ", "", -1)
}

func buildArrayType(name string, schema spec.Schema) []*jen.Statement {
	if schema.Items.Schema == nil {
		panic("multi-type array schema not yet supported")
	}

	comment := &jen.Statement{}
	addComment(comment, schema)

	itemName := schema.Items.Schema.Title
	if itemName == "" {
		itemName = name + "Item"
	}
	arrayType := jen.Type().Id(name).Index().Id(itemName)

	// TODO: clean this up, assumes item is always an object type
	_, extraTypes := buildType(itemName, *schema.Items.Schema)
	return append([]*jen.Statement{comment, arrayType}, extraTypes...)
}

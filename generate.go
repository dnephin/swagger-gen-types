package main

import (
	"bytes"
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode"

	"github.com/dave/jennifer/jen"
	"github.com/go-openapi/spec"
)

type generateContext struct {
	definitionsPackage string
	file               *jen.File
}

func generateDefinition(gen generateContext, name string, schema spec.Schema) {
	// TODO: can definitions be array type?
	addStatementsToFile(gen.file, buildStructType(gen, name, schema))
}

func addStatementsToFile(file *jen.File, stmts []*jen.Statement) {
	for _, stmt := range stmts {
		file.Add(stmt)
	}
}

func buildStructType(gen generateContext, name string, schema spec.Schema) []*jen.Statement {
	var extraTypes []*jen.Statement
	comment := &jen.Statement{}
	addComment(comment, schema)

	structName := schema.Title
	if structName == "" {
		structName = name
	}

	structType := jen.Type().Id(structName).StructFunc(func(g *jen.Group) {
		for _, propName := range sortedKeys(schema.Properties) {
			property := schema.Properties[propName]

			addComment(g, property)
			name := fieldName(propName, property)
			field := jen.Id(name)
			fieldType, extras := buildType(gen, structName+name, property)
			if len(extras) > 0 {
				extraTypes = append(extraTypes, extras...)
			}
			g.Add(field.Add(fieldType).Tag(fieldTags(propName, schema)))
		}
	})

	return append([]*jen.Statement{comment, structType}, extraTypes...)
}

func addComment(ast commentable, schema spec.Schema) *jen.Statement {
	if schema.Description != "" {
		return ast.Comment(schema.Description)
	}
	return ast.Add()
}

type commentable interface {
	Add(code ...jen.Code) *jen.Statement
	Comment(str string) *jen.Statement
}

func buildType(gen generateContext, propName string, schema spec.Schema) (*jen.Statement, []*jen.Statement) {
	if len(schema.Type) > 1 {
		panic(fmt.Sprintf("%s: multi-type schema not yet supported: %s", propName, schema.Type))
	}

	stmt := &jen.Statement{}
	if value, ok := schema.Extensions.GetBool("x-nullable"); ok && value {
		stmt.Op("*")
	}

	if len(schema.Type) == 0 {
		switch {
		case schema.Ref.String() != "":
			// TODO: this needs to lookup title in Ref schema?

			if gen.definitionsPackage != "" && isDefinitionRef(schema.Ref) {
				return stmt.Qual(gen.definitionsPackage, path.Base(schema.Ref.String())), nil
			}
			return stmt.Id(path.Base(schema.Ref.String())), nil
		default:
			panic(fmt.Sprintf("%s: missing type and ref", propName))
		}

	}

	switch schema.Type[0] {
	case "string":
		return stmt.String(), nil

	case "integer", "int":
		switch schema.Format {
		case "uint8":
			return stmt.Uint8(), nil
		case "uint16":
			return stmt.Uint16(), nil
		case "uint32":
			return stmt.Uint32(), nil
		case "int64", "":
			return stmt.Int64(), nil
		default:
			panic(fmt.Sprintf("int format %s not yet implemented", schema.Format))
		}

	case "boolean", "bool":
		return stmt.Bool(), nil

	case "object":
		switch {
		case len(schema.Properties) > 0:
			extras := buildStructType(gen, propName, schema)
			return stmt.Id(propName), extras
		case schema.AdditionalProperties != nil && schema.AdditionalProperties.Schema != nil:
			fieldType, extras := buildType(gen, propName, *schema.AdditionalProperties.Schema)
			return stmt.Map(jen.String()).Add(fieldType), extras
		case schema.AdditionalProperties != nil:
			return stmt.Map(jen.String()).Interface(), nil
		default:
			return jen.Interface(), nil
		}

	case "array":
		if schema.Items.Schema != nil {
			fieldType, extra := buildType(gen, propName, *schema.Items.Schema)
			return stmt.Index().Add(fieldType), extra
		} else {
			panic("multi-type array schema not yet supported")
		}
	default:
		panic(schema.Type[0] + " type not supported yet")
	}
}

func fieldTags(propName string, property spec.Schema) map[string]string {
	tags := []string{propName}
	if !isRequired(propName, property) {
		tags = append(tags, "omitempty")
	}
	return map[string]string{"json": strings.Join(tags, ",")}
}

func fieldName(propName string, property spec.Schema) string {
	if property.Title != "" {
		return property.Title
	}
	return toGoName(propName)
}

func isRequired(propName string, property spec.Schema) bool {
	for _, required := range property.Required {
		if required == propName {
			return true
		}
	}
	return false
}

func CamelCaseToUnderscore(source string) string {
	var buff bytes.Buffer
	var prevCharIsLower bool

	for _, char := range source {
		var n rune
		switch {
		case unicode.IsLower(char):
			n = char
		case prevCharIsLower:
			buff.WriteRune('_')
			n = unicode.ToLower(char)
		default:
			n = unicode.ToLower(char)
		}
		buff.WriteRune(n)
		prevCharIsLower = unicode.IsLower(char)
	}
	return buff.String()
}

func UnderscoreToCamelCase(source string) string {
	var buff bytes.Buffer
	var prevCharIsUnder = true

	for _, char := range source {
		switch {
		case char == '_':
			prevCharIsUnder = true
			continue
		case prevCharIsUnder:
			prevCharIsUnder = false
			buff.WriteRune(unicode.ToUpper(char))
		default:
			buff.WriteRune(char)
		}
	}
	return buff.String()
}

func toGoName(name string) string {
	switch {
	case strings.EqualFold(name, "id"):
		return "ID"
	}
	return UnderscoreToCamelCase(name)
}

func addCodeGeneratedComment(f *jen.File) {
	f.HeaderComment("Code generated by swagger-gen. DO NOT EDIT")
}

func sortedKeys(m map[string]spec.Schema) []string {
	var keys []string
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// TODO: is there a safer way to perform this check?
func isDefinitionRef(ref spec.Ref) bool {
	return strings.HasPrefix(ref.String(), "#/definitions/")
}

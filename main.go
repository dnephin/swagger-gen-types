package main

import (
	"log"
	"os"
	"sort"

	"github.com/dave/jennifer/jen"
	"github.com/go-openapi/loads"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

type options struct {
	spec               string
	pkgname            string
	definitions        []string
	operations         []string
	output             string
	definitionsPackage string
}

func main() {
	setupLogging()
	opts, err := parseOpts(os.Args)
	if err != nil {
		log.Fatal(err)
	}
	if err := run(opts); err != nil {
		log.Fatal(err)
	}
}

func setupLogging() {
	log.SetFlags(0)
}

func parseOpts(args []string) (*options, error) {
	flags := pflag.NewFlagSet(args[0], pflag.ContinueOnError)

	opts := &options{}
	flags.StringVarP(&opts.spec, "spec", "f", "swagger.yaml",
		"path to swagger api spec")
	flags.StringVar(&opts.pkgname, "package", "types",
		"go package name")
	flags.StringSliceVarP(&opts.definitions, "definition", "d", nil,
		"generate a type for this definition")
	flags.StringSliceVarP(&opts.operations, "operations", "p", nil,
		"generate response types for this operation")
	flags.StringVarP(&opts.output, "output", "o", "types.go", "write output to file")
	flags.StringVar(&opts.definitionsPackage, "definitions-package", "types",
		"use this package name when operations reference definition types")

	return opts, flags.Parse(args)
}

func run(opts *options) error {
	doc, err := loads.Spec(opts.spec)
	if err != nil {
		return errors.Wrapf(err, "failed to load spec file")
	}

	file := jen.NewFile(opts.pkgname)
	addCodeGeneratedComment(file)

	gen := generateContext{file: file}

	sort.Strings(opts.definitions)
	for _, definition := range opts.definitions {
		def, ok := doc.Spec().Definitions[definition]
		if !ok {
			return errors.Errorf("%s not found in spec", definition)
		}
		generateDefinition(gen, definition, def)
	}

	gen.definitionsPackage = opts.definitionsPackage
	sort.Strings(opts.operations)
	for _, operation := range opts.operations {
		method, path, op, ok := doc.Analyzer.OperationForName(operation)
		if !ok {
			return errors.Errorf("%s not found in spec", operation)
		}
		generateOperation(gen, operation, method, path, op)
	}

	return file.Save(opts.output)
}

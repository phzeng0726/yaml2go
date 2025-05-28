package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/phzeng0726/yaml2go/pkg/generator"
)

func main() {
	input := flag.String("i", "", "Path to YAML input file")
	output := flag.String("o", "", "Path to output Go file (optional, default to stdout)")
	structName := flag.String("struct", "YAMLToGoStruct", "Name of the Go struct")
	withJsonTag := flag.Bool("json", false, "Whether to include JSON tags in the struct fields")

	flag.Parse()

	if *input == "" {
		log.Fatal("input file is required")
	}

	data, err := os.ReadFile(*input)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}

	code, err := generator.GenerateGoStruct(string(data), *structName, withJsonTag)
	if err != nil {
		log.Fatalf("failed to generate go struct: %v", err)
	}

	if *output != "" {
		err = os.WriteFile(*output, []byte(code), 0644)
		if err != nil {
			log.Fatalf("failed to write output file: %v", err)
		}
		fmt.Println("Generated struct written to", *output)
	} else {
		fmt.Print(code)
	}
}

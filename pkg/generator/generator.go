package generator

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// represents a key-value pair in YAML
type yamlEntry struct {
	Key     string
	Value   string
	Comment string
	Kind    yaml.Kind
	Node    *yaml.Node // Reference to the original node
}

// converts snake_case to CamelCase
func toCamel(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	// Handle special characters by replacing them
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	s = reg.ReplaceAllString(s, "_")

	// Handle case where the string starts with a number
	if regexp.MustCompile(`^[0-9]`).MatchString(s) {
		s = "N" + s
	}

	parts := strings.Split(s, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][0:1]) + parts[i][1:]
		}
	}

	return strings.Join(parts, "")
}

// determines the corresponding Go type for a YAML node
func determineGoType(node *yaml.Node) string {
	switch node.Kind {
	case yaml.ScalarNode:
		switch node.Tag {
		case "!!int":
			return "int"
		case "!!float":
			return "float64"
		case "!!bool":
			return "bool"
		default:
			return "string"
		}
	case yaml.SequenceNode:
		if len(node.Content) > 0 {
			elemType := determineGoType(node.Content[0])
			return "[]" + elemType
		}
		return "[]interface{}"
	case yaml.MappingNode:
		return "struct" // This will be replaced with the actual struct name
	default:
		return "interface{}"
	}
}

// extract sorted YAML entries
func extractSortedYamlEntries(node *yaml.Node, structName string) ([]yamlEntry, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected mapping node for struct %s", structName)
	}

	entries := []yamlEntry{}
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break
		}
		keyNode := node.Content[i]
		valNode := node.Content[i+1]

		entries = append(entries, yamlEntry{
			Key:     keyNode.Value,
			Value:   valNode.Value,
			Comment: strings.TrimSpace(valNode.LineComment),
			Kind:    valNode.Kind,
			Node:    valNode,
		})
	}

	// Sort by key
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})

	return entries, nil
}

// processes a YAML node and generates Go struct code
func processNode(node *yaml.Node, structName string, indent string, withJsonTag *bool) (string, error) {
	entries, err := extractSortedYamlEntries(node, structName)
	if err != nil {
		return "", err
	}

	// Generate Go struct
	var b bytes.Buffer
	fmt.Fprintf(&b, "%stype %s struct {\n", indent, structName)

	subStructs := ""

	for _, e := range entries {
		fieldName := toCamel(e.Key)
		fieldType := determineGoType(e.Node)
		comment := ""
		if e.Comment != "" {
			comment = fmt.Sprintf(" // %s", e.Comment)
		}

		if e.Kind == yaml.MappingNode {
			subStructName := structName + fieldName
			subStruct, err := processNode(e.Node, subStructName, indent, withJsonTag)
			if err != nil {
				return "", err
			}
			subStructs += subStruct
			fieldType = subStructName
		} else if e.Kind == yaml.SequenceNode && len(e.Node.Content) > 0 && e.Node.Content[0].Kind == yaml.MappingNode {
			// Handle array of objects
			subStructName := structName + fieldName
			elemStruct, err := processNode(e.Node.Content[0], subStructName, indent, withJsonTag)
			if err != nil {
				return "", err
			}
			subStructs += elemStruct
			fieldType = "[]" + subStructName
		}

		// Add json tag if withJsonTag is true
		jsonTag := ""
		if *withJsonTag {
			jsonTag = fmt.Sprintf(" json:\"%s\"", e.Key)
		}
		fmt.Fprintf(&b, "%s\t%s %s `yaml:\"%s\"%s`%s\n", indent, fieldName, fieldType, e.Key, jsonTag, comment)
	}

	fmt.Fprintf(&b, "%s}\n\n", indent)
	return b.String() + subStructs, nil
}

// generates Go struct code from YAML content
func GenerateGoStruct(yamlContent string, structName string, withJsonTag *bool) (string, error) {
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(yamlContent), &node); err != nil {
		return "", err
	}

	// Find the first mapping node
	var rootNode *yaml.Node
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		rootNode = node.Content[0]
	} else if node.Kind == yaml.MappingNode {
		rootNode = &node
	} else {
		return "", fmt.Errorf("invalid YAML format: expected mapping node")
	}

	return processNode(rootNode, structName, "", withJsonTag)
}

package docset

import (
	"fmt"
	"strings"
)

// CreateConfig holds configuration for docset creation
type CreateConfig struct {
	Name      string
	URL       string
	Include   []string
	Exclude   []string
	Depth     int
	Version   string
	IconPath  string
	IndexPage string
	TypeMap   map[string]string
	Selector  string
	DryRun    bool
	Force     bool
	DataDir   string
}

// TypeMapping represents a single type mapping rule
type TypeMapping struct {
	Pattern string
	Type    string
}

// ParseTypeMap parses a type-map string into a map
// Format: "pattern:type,pattern:type"
func ParseTypeMap(typeMapStr string) (map[string]string, error) {
	mappings := make(map[string]string)

	if typeMapStr == "" {
		return mappings, nil
	}

	pairs := strings.Split(typeMapStr, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid type mapping: %s (expected format: pattern:type)", pair)
		}

		pattern := strings.TrimSpace(parts[0])
		entryType := strings.TrimSpace(parts[1])

		if pattern == "" || entryType == "" {
			return nil, fmt.Errorf("empty pattern or type in mapping: %s", pair)
		}

		mappings[pattern] = entryType
	}

	return mappings, nil
}

// IsValidEntryType checks if a type is valid per Kapeli's spec
func IsValidEntryType(entryType string) bool {
	validTypes := []string{
		"Annotation", "Attribute", "Binding", "Builtin", "Callback",
		"Category", "Class", "Command", "Component", "Constant",
		"Constructor", "Define", "Delegate", "Diagram", "Directive",
		"Element", "Entry", "Enum", "Environment", "Error",
		"Event", "Exception", "Extension", "Field", "File",
		"Filter", "Framework", "Function", "Global", "Guide",
		"Hook", "Instance", "Instruction", "Interface", "Keyword",
		"Library", "Literal", "Macro", "Method", "Mixin",
		"Modifier", "Module", "Namespace", "Notation", "Object",
		"Operator", "Option", "Package", "Parameter", "Plugin",
		"Procedure", "Property", "Protocol", "Provider", "Provisioner",
		"Query", "Record", "Resource", "Sample", "Section",
		"Service", "Setting", "Shortcut", "Statement", "Struct",
		"Style", "Subroutine", "Tag", "Test", "Trait",
		"Type", "Union", "Value", "Variable", "Word",
	}

	for _, valid := range validTypes {
		if strings.EqualFold(valid, entryType) {
			return true
		}
	}
	return false
}

package flow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parse reads a flow definition file (YAML or JSON) and returns a Flow
func Parse(filePath string) (*Flow, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return ParseBytes(data, filePath)
}

// ParseBytes parses flow definition from bytes, auto-detecting format
func ParseBytes(data []byte, filename string) (*Flow, error) {
	var flow Flow
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &flow); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &flow); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	default:
		// Try YAML first, then JSON
		if err := yaml.Unmarshal(data, &flow); err != nil {
			if err := json.Unmarshal(data, &flow); err != nil {
				return nil, fmt.Errorf("failed to parse as YAML or JSON: %w", err)
			}
		}
	}

	return &flow, nil
}

// Save writes a flow definition to a file in the specified format
func Save(flow *Flow, filePath string) error {
	ext := strings.ToLower(filepath.Ext(filePath))

	var data []byte
	var err error

	switch ext {
	case ".json":
		data, err = json.MarshalIndent(flow, "", "  ")
	case ".yaml", ".yml":
		data, err = yaml.Marshal(flow)
	default:
		return fmt.Errorf("unsupported file extension: %s (use .json, .yaml, or .yml)", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal flow: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

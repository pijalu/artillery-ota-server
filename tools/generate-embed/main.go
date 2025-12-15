package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"
)

type Config struct {
	Mappings []FirmwareMapping `json:"mappings"`
}

type FirmwareMapping struct {
	CustomerType string `json:"customerType"`
	FirmwareType string `json:"firmwareType"`
	FilePath     string `json:"filePath"`
	Embed        bool   `json:"embed"`
}

const embeddedFilesTemplate = `package main

import (
	"embed"
)

//go:embed {{range $i, $path := .EmbedPaths}}{{if $i}} {{end}}"{{$path}}"{{end}}
var embeddedFiles embed.FS
`

func main() {
	// Read config file (from project root, where this is executed via go:generate)
	data, err := os.ReadFile("config.json")
	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		os.Exit(1)
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("Error parsing config file: %v\n", err)
		os.Exit(1)
	}

	// Collect files to embed - always include config.json
	embedPaths := []string{"config.json"}

	for _, mapping := range config.Mappings {
		if mapping.Embed {
			// Convert relative path to a format suitable for embed
			// Remove leading ../ or ./ and normalize the path
			relativePath := strings.TrimPrefix(mapping.FilePath, "../")
			relativePath = strings.TrimPrefix(relativePath, "./")
			embedPaths = append(embedPaths, relativePath)
		}
	}

	// Verify all files exist before generating
	for _, path := range embedPaths {
		// Check relative to the project root (config paths are relative to project root)
		if path != "config.json" {  // Only verify non-config files exist
			if _, err := os.Stat(path); os.IsNotExist(err) {  // Path is relative to project root where config was read
				fmt.Printf("Error: File does not exist for embedding: %s\n", path)
				os.Exit(1)
			}
		}
	}

	// Generate the embedded files Go file
	tmpl, err := template.New("embedded").Parse(embeddedFilesTemplate)
	if err != nil {
		fmt.Printf("Error parsing template: %v\n", err)
		os.Exit(1)
	}

	file, err := os.Create("embedded_files.go")  // Write to current directory (project root)
	if err != nil {
		fmt.Printf("Error creating embedded_files.go: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	err = tmpl.Execute(file, map[string]interface{}{
		"EmbedPaths": embedPaths,
	})
	if err != nil {
		fmt.Printf("Error executing template: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated embedded_files.go with %d files to embed: %v\n", len(embedPaths), embedPaths)
}
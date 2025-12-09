package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

// convertJSONToYAML converts JSON content to YAML format
func convertJSONToYAML(jsonContent string) (string, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(jsonContent), &data); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func convert(ctx context.Context, cmd *cli.Command) error {
	inputFile := cmd.String("input")
	outputFile := cmd.String("output")

	// Handle positional arguments if flags not provided
	if inputFile == "" && cmd.Args().Len() > 0 {
		inputFile = cmd.Args().Get(0)
	}

	if outputFile == "" && cmd.Args().Len() > 1 {
		outputFile = cmd.Args().Get(1)
	}

	if inputFile == "" {
		return fmt.Errorf("input file is required")
	}

	// Read input JSON file
	fileBytes, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Convert JSON to YAML
	yamlData, err := convertJSONToYAML(string(fileBytes))
	if err != nil {
		return err
	}

	// Write output
	if outputFile != "" {
		err = os.WriteFile(outputFile, []byte(yamlData), 0o644)
		if err != nil {
			return fmt.Errorf("error writing output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Successfully converted %s to %s\n", inputFile, outputFile)
	} else {
		fmt.Print(yamlData)
	}

	return nil
}

func webMode(ctx context.Context, cmd *cli.Command) error {
	port := cmd.String("port")
	if port == "" {
		port = "8080"
	}

	fmt.Println("json2yaml - Web Mode")
	fmt.Println("Starting web interface...")

	return startWebServer(port)
}

func main() {
	// Check if no arguments provided - start web mode
	if len(os.Args) == 1 {
		fmt.Println("json2yaml - Web Mode")
		fmt.Println("Starting web interface...")
		err := startWebServer("8080")
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	cmd := &cli.Command{
		Name:    "json2yaml",
		Version: "1.0.0",
		Usage:   "Convert JSON files to YAML format",
		Description: `json2yaml converts JSON files to YAML format.

This is a sample tool demonstrating how to add a Web GUI to a CLI tool.

Usage:
  json2yaml                      # Start web interface
  json2yaml web                  # Start web interface
  json2yaml input.json           # Convert and output to stdout
  json2yaml input.json output.yaml  # Convert and save to file`,
		ArgsUsage: "[input.json] [output.yaml]",
		Commands: []*cli.Command{
			{
				Name:   "web",
				Usage:  "Start web interface",
				Action: webMode,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "port",
						Aliases: []string{"p"},
						Usage:   "Port to run web server on",
						Value:   "8080",
					},
				},
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "input",
				Aliases: []string{"i"},
				Usage:   "Input JSON file path",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output YAML file path (optional, defaults to stdout)",
			},
		},
		Action: convert,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

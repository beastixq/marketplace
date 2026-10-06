// Command testreport renders Go test JSON events as a standalone HTML report.
package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed report.html
var reportHTML string

type testEvent struct {
	Action     string
	Package    string
	ImportPath string
	Test       string
	Output     string
	Elapsed    float64
}

type testResult struct {
	Name        string
	Status      string
	Output      string
	Elapsed     float64
	HasChildren bool
}

type packageResult struct {
	Name   string
	Status string
	Output string
	Seed   string
	Tests  []*testResult
}

type report struct {
	Packages       []*packageResult
	Passed         int
	Failed         int
	Skipped        int
	Incomplete     int
	FailedPackages int
}

func main() {
	input := flag.String("input", "", "Go test JSONL file")
	output := flag.String("output", "", "HTML output file")
	flag.Parse()
	if err := writeReport(*input, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Test results: %s\n", *output)
}

func writeReport(input, output string) error {
	if input == "" || output == "" {
		return errors.New("both -input and -output are required")
	}
	inputPath, err := filepath.Abs(input)
	if err != nil {
		return fmt.Errorf("resolve input path: %w", err)
	}
	outputPath, err := filepath.Abs(output)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	if inputPath == outputPath {
		return errors.New("input and output must be different files")
	}
	file, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("open test events: %w", err)
	}
	defer file.Close()
	results, err := readReport(file)
	if err != nil {
		return err
	}
	var html bytes.Buffer
	if err := renderReport(&html, results); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}
	if err := os.WriteFile(output, html.Bytes(), 0644); err != nil {
		return fmt.Errorf("write HTML report: %w", err)
	}
	return nil
}

func readReport(input io.Reader) (report, error) {
	packages := make(map[string]*packageResult)
	tests := make(map[string]map[string]*testResult)
	decoder := json.NewDecoder(input)
	for {
		var event testEvent
		if err := decoder.Decode(&event); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return report{}, fmt.Errorf("decode test events: %w", err)
		}
		if event.Package == "" {
			event.Package = event.ImportPath
		}
		if event.Package == "" {
			continue
		}
		pkg := packages[event.Package]
		if pkg == nil {
			pkg = &packageResult{Name: event.Package, Status: "incomplete"}
			packages[event.Package] = pkg
			tests[event.Package] = make(map[string]*testResult)
		}
		if event.Test == "" {
			switch event.Action {
			case "pass", "fail", "skip":
				pkg.Status = event.Action
			case "build-fail":
				pkg.Status = "fail"
			case "output", "build-output":
				pkg.Output += event.Output
				if strings.HasPrefix(event.Output, "-test.shuffle ") {
					pkg.Seed = strings.TrimSpace(strings.TrimPrefix(event.Output, "-test.shuffle "))
				}
			}
			continue
		}
		result := tests[event.Package][event.Test]
		if result == nil {
			result = &testResult{Name: event.Test, Status: "incomplete"}
			tests[event.Package][event.Test] = result
		}
		switch event.Action {
		case "pass", "fail", "skip":
			result.Status = event.Action
			result.Elapsed = event.Elapsed
		case "output":
			result.Output += event.Output
		}
	}
	var results report
	for name, pkg := range packages {
		for testName := range tests[name] {
			for parent := testName; strings.Contains(parent, "/"); {
				parent = parent[:strings.LastIndex(parent, "/")]
				if result := tests[name][parent]; result != nil {
					result.HasChildren = true
				}
			}
		}
		for _, result := range tests[name] {
			if result.HasChildren {
				if result.Status == "fail" {
					pkg.Output += result.Output
				}
				continue
			}
			pkg.Tests = append(pkg.Tests, result)
			switch result.Status {
			case "pass":
				results.Passed++
			case "fail":
				results.Failed++
			case "skip":
				results.Skipped++
			default:
				results.Incomplete++
			}
		}
		if pkg.Status == "fail" || pkg.Status == "incomplete" {
			results.FailedPackages++
		}
		sort.Slice(pkg.Tests, func(i, j int) bool { return pkg.Tests[i].Name < pkg.Tests[j].Name })
		results.Packages = append(results.Packages, pkg)
	}
	if len(results.Packages) == 0 {
		return report{}, errors.New("no package test events found")
	}
	sort.Slice(results.Packages, func(i, j int) bool { return results.Packages[i].Name < results.Packages[j].Name })
	return results, nil
}

func renderReport(output io.Writer, results report) error {
	tmpl, err := template.New("report").Parse(reportHTML)
	if err != nil {
		return fmt.Errorf("parse report template: %w", err)
	}
	if err := tmpl.Execute(output, results); err != nil {
		return fmt.Errorf("render report: %w", err)
	}
	return nil
}

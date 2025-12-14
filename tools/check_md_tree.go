package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// findMarkdownLinks returns all .md files referenced in a markdown file
func findMarkdownLinks(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Regex for [text](file.md)
	re := regexp.MustCompile(`\[[^\]]+\]\(([^)]+\.md)\)`)
	matches := re.FindAllStringSubmatch(string(content), -1)
	var links []string
	for _, m := range matches {
		links = append(links, m[1])
	}
	return links, nil
}

func main() {
	verbose := flag.Bool("v", false, "verbose: show tree even when all files are linked")
	flag.Parse()

	root := "."
	mdFiles := map[string]bool{}
	parentLinks := map[string][]string{}

	// Find all .md files
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			mdFiles[path] = false // not yet referenced
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error walking files:", err)
		os.Exit(1)
	}

	// For each .md file, find links to other .md files
	for file := range mdFiles {
		links, err := findMarkdownLinks(file)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", file, err)
			continue
		}
		for _, link := range links {
			// Normalize path
			linkPath := filepath.Clean(filepath.Join(filepath.Dir(file), link))
			parentLinks[file] = append(parentLinks[file], linkPath)
			if _, ok := mdFiles[linkPath]; ok {
				mdFiles[linkPath] = true // referenced
			}
		}
	}

	// Check for unreferenced .md files (excluding root README.md)
	unrefCount := 0
	var unreferenced []string
	for file, referenced := range mdFiles {
		if !referenced && file != "./README.md" && file != "README.md" {
			unreferenced = append(unreferenced, file)
			unrefCount++
		}
	}

	// Print tree function
	var printTree func(string, string)
	printTree = func(file, prefix string) {
		// Clean up the path for display
		displayPath := strings.TrimPrefix(file, "./")
		fmt.Println(prefix + displayPath)
		for _, child := range parentLinks[file] {
			printTree(child, prefix+"  ")
		}
	}

	// Find the root README
	rootReadme := ""
	if _, ok := mdFiles["./README.md"]; ok {
		rootReadme = "./README.md"
	} else if _, ok := mdFiles["README.md"]; ok {
		rootReadme = "README.md"
	}

	// Show tree if verbose or if there are unreferenced files
	if *verbose || unrefCount > 0 {
		fmt.Println("Markdown file reference tree:")
		if rootReadme != "" {
			printTree(rootReadme, "")
		}

		// In verbose mode, also show all markdown files
		if *verbose && unrefCount == 0 {
			fmt.Println("\n✓ All markdown files are properly linked")
		}
	}

	// Only output unreferenced files if there are any
	if unrefCount > 0 {
		fmt.Println("\nUnreferenced .md files:")
		for _, file := range unreferenced {
			displayPath := strings.TrimPrefix(file, "./")
			fmt.Println("  ", displayPath)
		}
		os.Exit(1)
	}
}

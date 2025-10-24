package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// findMarkdownLinks returns all .md files referenced in a markdown file
func findMarkdownLinks(path string) ([]string, error) {
	content, err := ioutil.ReadFile(path)
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

	// Print tree
	fmt.Println("Markdown file reference tree:")
	var printTree func(string, string)
	printTree = func(file, prefix string) {
		fmt.Println(prefix + filepath.Base(file))
		for _, child := range parentLinks[file] {
			printTree(child, prefix+"  ")
		}
	}
	if _, ok := mdFiles["./README.md"]; ok {
		printTree("./README.md", "")
	}

	// Print unreferenced .md files (excluding README.md as root)
	fmt.Println("\nUnreferenced .md files:")
	unrefCount := 0
	for file, referenced := range mdFiles {
		if !referenced && filepath.Base(file) != "README.md" {
			fmt.Println("  ", file)
			unrefCount++
		}
	}
	if unrefCount == 0 {
		fmt.Println("  (none)")
	}
}

package contacts

import (
	"testing"
)

func TestExtractXHandleFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "Standard X URL",
			url:      "https://x.com/MoritzBoegli",
			expected: "@MoritzBoegli",
		},
		{
			name:     "Twitter URL",
			url:      "https://twitter.com/thuritch",
			expected: "@thuritch",
		},
		{
			name:     "URL without https",
			url:      "www.x.com/pascallamprecht",
			expected: "@pascallamprecht",
		},
		{
			name:     "URL in array format from YAML",
			url:      "https://x.com/HenzYves",
			expected: "@HenzYves",
		},
		{
			name:     "Invalid URL",
			url:      "https://facebook.com/something",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractXHandleFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("ExtractXHandleFromURL(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestGenerateNameVariants(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Two part name",
			input:    "Bögli Moritz",
			expected: []string{"Bögli Moritz", "Moritz Bögli"},
		},
		{
			name:     "Three part name",
			input:    "Garcia Nuñez David",
			expected: []string{"Garcia Nuñez David", "David Garcia Nuñez"},
		},
		{
			name:     "Single name",
			input:    "Madonna",
			expected: []string{"Madonna"},
		},
		{
			name:     "Name with extra spaces",
			input:    "  Moritz  Bögli  ",
			expected: []string{"Moritz Bögli", "Bögli Moritz"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateNameVariants(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("generateNameVariants(%q) returned %d variants, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, variant := range result {
				if variant != tt.expected[i] {
					t.Errorf("generateNameVariants(%q)[%d] = %q, want %q", tt.input, i, variant, tt.expected[i])
				}
			}
		})
	}
}

func TestTagXHandlesInText(t *testing.T) {
	// Create a test mapper with sample contacts
	mapper := &Mapper{
		contacts: map[string]Contact{
			"Bögli Moritz": {
				Name: "Bögli Moritz",
				X:    []string{"https://x.com/MoritzBoegli"},
			},
			"moritz bögli": {
				Name: "Bögli Moritz",
				X:    []string{"https://x.com/MoritzBoegli"},
			},
			"Garcia Nuñez David": {
				Name: "Garcia Nuñez David",
				X:    []string{"https://x.com/thuritch"},
			},
			"garcia nuñez david": {
				Name: "Garcia Nuñez David",
				X:    []string{"https://x.com/thuritch"},
			},
			"Christian Häberli": {
				Name: "Christian Häberli",
				X:    []string{},
			},
			"christian häberli": {
				Name: "Christian Häberli",
				X:    []string{},
			},
		},
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single name match",
			input:    "Dringliches Postulat von Moritz Bögli (AL)",
			expected: "Dringliches Postulat von Moritz Bögli @MoritzBoegli (AL)",
		},
		{
			name:     "Multiple names, some with handles",
			input:    "Postulat von Moritz Bögli (AL), Christian Häberli (AL) und Dr. David Garcia Nuñez (AL)",
			expected: "Postulat von Moritz Bögli @MoritzBoegli (AL), Christian Häberli (AL) und Dr. David Garcia Nuñez @thuritch (AL)",
		},
		{
			name:     "No matches",
			input:    "Postulat von Unknown Person (AL)",
			expected: "Postulat von Unknown Person (AL)",
		},
		{
			name:     "Name in different format",
			input:    "Motion von Bögli Moritz",
			expected: "Motion von Bögli Moritz @MoritzBoegli",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapper.TagXHandlesInText(tt.input)
			if result != tt.expected {
				t.Errorf("TagXHandlesInText() failed\nInput:    %q\nExpected: %q\nGot:      %q", tt.input, tt.expected, result)
			}
		})
	}
}

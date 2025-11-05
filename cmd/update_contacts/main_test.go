package main

import (
	"testing"
)

func TestMergeContacts_NewContact(t *testing.T) {
	// Given: empty existing contacts
	existing := make(map[string]*Contact)

	// When: API returns a new contact
	apiContacts := []Contact{
		{
			Name:     "Test Person",
			X:        []string{"@testperson"},
			Facebook: []string{"https://facebook.com/test"},
		},
	}

	// Then: contact should be added
	result, added, warnings := mergeContacts(existing, apiContacts)

	if len(result) != 1 {
		t.Errorf("Expected 1 contact, got %d", len(result))
	}
	if added != 1 {
		t.Errorf("Expected 1 added contact, got %d", added)
	}
	if warnings != 0 {
		t.Errorf("Expected 0 warnings, got %d", warnings)
	}
	if result[0].Name != "Test Person" {
		t.Errorf("Expected name 'Test Person', got '%s'", result[0].Name)
	}
}

func TestMergeContacts_NewPlatform(t *testing.T) {
	// Given: existing contact with only X
	existing := map[string]*Contact{
		"Test Person": {
			Name: "Test Person",
			X:    []string{"@testperson"},
		},
	}

	// When: API returns same contact with new Instagram
	apiContacts := []Contact{
		{
			Name:      "Test Person",
			X:         []string{"@testperson"},
			Instagram: []string{"https://instagram.com/test"},
		},
	}

	// Then: Instagram should be added, no warnings
	result, added, warnings := mergeContacts(existing, apiContacts)

	if len(result) != 1 {
		t.Errorf("Expected 1 contact, got %d", len(result))
	}
	if added != 0 {
		t.Errorf("Expected 0 new contacts, got %d", added)
	}
	if warnings != 0 {
		t.Errorf("Expected 0 warnings, got %d", warnings)
	}
	if len(result[0].Instagram) == 0 || result[0].Instagram[0] != "https://instagram.com/test" {
		t.Errorf("Expected Instagram to be added")
	}
}

func TestMergeContacts_ExistingAccountSame(t *testing.T) {
	// Given: existing contact with X account
	existing := map[string]*Contact{
		"Test Person": {
			Name: "Test Person",
			X:    []string{"@testperson"},
		},
	}

	// When: API returns same contact with same X account
	apiContacts := []Contact{
		{
			Name: "Test Person",
			X:    []string{"@testperson"},
		},
	}

	// Then: no changes, no warnings
	result, added, warnings := mergeContacts(existing, apiContacts)

	if len(result) != 1 {
		t.Errorf("Expected 1 contact, got %d", len(result))
	}
	if added != 0 {
		t.Errorf("Expected 0 new contacts, got %d", added)
	}
	if warnings != 0 {
		t.Errorf("Expected 0 warnings, got %d", warnings)
	}
	if len(result[0].X) != 1 || result[0].X[0] != "@testperson" {
		t.Errorf("Expected X to remain '@testperson', got '%v'", result[0].X)
	}
}

func TestMergeContacts_MultipleAccounts(t *testing.T) {
	// Given: existing contact with one X account
	existing := map[string]*Contact{
		"Test Person": {
			Name: "Test Person",
			X:    []string{"@testperson"},
		},
	}

	// When: API returns same contact with additional X account
	apiContacts := []Contact{
		{
			Name: "Test Person",
			X:    []string{"@testperson", "@testperson2"},
		},
	}

	// Then: second account should be added
	result, added, warnings := mergeContacts(existing, apiContacts)

	if len(result) != 1 {
		t.Errorf("Expected 1 contact, got %d", len(result))
	}
	if added != 0 {
		t.Errorf("Expected 0 new contacts, got %d", added)
	}
	if warnings != 0 {
		t.Errorf("Expected 0 warnings, got %d", warnings)
	}
	if len(result[0].X) != 2 {
		t.Errorf("Expected 2 X accounts, got %d", len(result[0].X))
	}
}

func TestMergeContacts_ConflictingAccount(t *testing.T) {
	// Given: existing contact with X account
	existing := map[string]*Contact{
		"Test Person": {
			Name: "Test Person",
			X:    []string{"@testperson"},
		},
	}

	// When: API returns same contact with different X account
	apiContacts := []Contact{
		{
			Name: "Test Person",
			X:    []string{"@differenthandle"},
		},
	}

	// Then: existing account kept, new account added
	result, added, warnings := mergeContacts(existing, apiContacts)

	if len(result) != 1 {
		t.Errorf("Expected 1 contact, got %d", len(result))
	}
	if added != 0 {
		t.Errorf("Expected 0 new contacts, got %d", added)
	}
	if warnings != 0 {
		t.Errorf("Expected 0 warnings (different accounts are both kept), got %d", warnings)
	}
	if len(result[0].X) != 2 {
		t.Errorf("Expected 2 X accounts, got %d", len(result[0].X))
	}
	if result[0].X[0] != "@testperson" || result[0].X[1] != "@differenthandle" {
		t.Errorf("Expected both accounts to be kept, got '%v'", result[0].X)
	}
}

func TestMergeContacts_MultiplePlatforms(t *testing.T) {
	// Given: existing contact with some platforms
	existing := map[string]*Contact{
		"Test Person": {
			Name:     "Test Person",
			X:        []string{"@testperson"},
			Facebook: []string{"https://facebook.com/test"},
		},
	}

	// When: API returns new platforms (Instagram, LinkedIn) and existing ones
	apiContacts := []Contact{
		{
			Name:      "Test Person",
			X:         []string{"@testperson"},
			Facebook:  []string{"https://facebook.com/test"},
			Instagram: []string{"https://instagram.com/test"},
			LinkedIn:  []string{"https://linkedin.com/in/test"},
		},
	}

	// Then: new platforms added, existing ones unchanged
	result, added, warnings := mergeContacts(existing, apiContacts)

	if len(result) != 1 {
		t.Errorf("Expected 1 contact, got %d", len(result))
	}
	if added != 0 {
		t.Errorf("Expected 0 new contacts, got %d", added)
	}
	if warnings != 0 {
		t.Errorf("Expected 0 warnings, got %d", warnings)
	}

	contact := result[0]
	if len(contact.Instagram) != 1 || contact.Instagram[0] != "https://instagram.com/test" {
		t.Errorf("Expected Instagram to be added")
	}
	if len(contact.LinkedIn) != 1 || contact.LinkedIn[0] != "https://linkedin.com/in/test" {
		t.Errorf("Expected LinkedIn to be added")
	}
	if len(contact.X) != 1 || contact.X[0] != "@testperson" {
		t.Errorf("Expected X to remain unchanged")
	}
	if len(contact.Facebook) != 1 || contact.Facebook[0] != "https://facebook.com/test" {
		t.Errorf("Expected Facebook to remain unchanged")
	}
}

func TestFormatYAMLForVSCode(t *testing.T) {
	// Given: Go YAML marshaler output (4-space for list, 6-space for fields)
	goYAML := `version: "1.0"
contacts:
    - name: Test Person
      x: '@test'
      facebook: https://facebook.com/test`

	// When: formatting for VS Code
	result := formatYAMLForVSCode(goYAML)

	// Then: should use 2-space for list, 4-space for fields
	expected := `version: "1.0"
contacts:
  - name: Test Person
    x: '@test'
    facebook: https://facebook.com/test`

	if result != expected {
		t.Errorf("formatYAMLForVSCode failed.\nGot:\n%s\n\nExpected:\n%s", result, expected)
	}
}

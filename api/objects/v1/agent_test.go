// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package objectsv1

import (
	"bytes"
	"os"
	"reflect"
	"sort"
	"testing"
)

func Ptr[T any](v T) *T {
	return &v
}

// Test Merge.
//
//nolint:cyclop
func TestAgent_Merge(t *testing.T) {
	agent1 := &Agent{
		Name:        "Agent1",
		Version:     "",
		Description: "Description1",
		Authors:     []string{"Author1"},
		Annotations: map[string]string{"key1": "value1"},
		Locators:    []*Locator{{Type: "type1", Url: "url1"}},
		Extensions:  []*Extension{{Name: "ext1", Version: "v1"}},
		Skills:      []*Skill{{CategoryUid: 1, ClassUid: 10101, CategoryName: Ptr("name1"), ClassName: Ptr("class1")}},
	}

	agent2 := &Agent{
		Name:        "",
		Version:     "v2",
		Description: "Description2",
		Authors:     []string{"Author2", "Author1"},
		Annotations: map[string]string{"key2": "value2"},
		Locators:    []*Locator{{Type: "type2", Url: "url2"}},
		Extensions:  []*Extension{{Name: "ext2", Version: "v2"}},
		Skills:      []*Skill{{CategoryUid: 2, ClassUid: 20101, CategoryName: Ptr("name2"), ClassName: Ptr("class2")}},
	}

	agent1.Merge(agent2)

	if agent1.GetName() != "Agent1" || agent1.GetVersion() != "v2" {
		t.Errorf("Merge failed for scalar fields")
	}

	if len(agent1.GetAuthors()) != 2 || agent1.GetAuthors()[0] != "Author2" || agent1.GetAuthors()[1] != "Author1" {
		t.Errorf("Merge failed for Authors")
	}

	if len(agent1.GetAnnotations()) != 2 || agent1.GetAnnotations()["key1"] != "value1" || agent1.GetAnnotations()["key2"] != "value2" {
		t.Errorf("Merge failed for Annotations")
	}

	if len(agent1.GetLocators()) != 2 {
		t.Errorf("Merge failed for Locators: expected 2, got %d", len(agent1.GetLocators()))
	}

	sort.Slice(agent1.GetLocators(), func(i, j int) bool {
		return agent1.GetLocators()[i].GetType() < agent1.GetLocators()[j].GetType()
	})

	if agent1.GetLocators()[0].GetType() != "type1" || agent1.GetLocators()[1].GetType() != "type2" {
		t.Errorf("Merge failed for Locators: got %v", agent1.GetLocators())
	}

	if len(agent1.GetExtensions()) != 2 {
		t.Errorf("Merge failed for Extensions: expected 2, got %d", len(agent1.GetExtensions()))
	}

	sort.Slice(agent1.GetExtensions(), func(i, j int) bool {
		return agent1.GetExtensions()[i].GetName() < agent1.GetExtensions()[j].GetName()
	})

	if agent1.GetExtensions()[0].GetName() != "ext1" || agent1.GetExtensions()[1].GetName() != "ext2" {
		t.Errorf("Merge failed for Extensions: got %v", agent1.GetExtensions())
	}

	if len(agent1.GetSkills()) != 2 {
		t.Errorf("Merge failed for Skills: expected 2, got %d", len(agent1.GetSkills()))
	}

	sort.Slice(agent1.GetSkills(), func(i, j int) bool {
		return agent1.GetSkills()[i].GetCategoryUid() < agent1.GetSkills()[j].GetCategoryUid()
	})

	if agent1.GetSkills()[0].GetCategoryUid() != 1 || agent1.GetSkills()[1].GetCategoryUid() != 2 {
		t.Errorf("Merge failed for Skills: got %v", agent1.GetSkills())
	}
}

// Test LoadFromReader.
func TestAgent_LoadFromReader(t *testing.T) {
	agent := &Agent{}
	data := `{"Name": "TestAgent", "Version": "1.0"}`
	reader := bytes.NewReader([]byte(data))

	_, err := agent.LoadFromReader(reader)
	if err != nil {
		t.Fatalf("LoadFromReader failed: %v", err)
	}

	if agent.GetName() != "TestAgent" || agent.GetVersion() != "1.0" {
		t.Errorf("LoadFromReader did not populate fields correctly")
	}
}

// Test LoadFromReader with invalid JSON.
func TestAgent_LoadFromReader_InvalidJSON(t *testing.T) {
	agent := &Agent{}
	data := `{"Name": "TestAgent", "Version":`
	reader := bytes.NewReader([]byte(data))

	_, err := agent.LoadFromReader(reader)
	if err == nil {
		t.Fatalf("LoadFromReader should have failed for invalid JSON")
	}
}

// Test LoadFromFile.
func TestAgent_LoadFromFile(t *testing.T) {
	agent := &Agent{}

	file, err := os.CreateTemp(t.TempDir(), "agent_test_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer os.Remove(file.Name())

	data := `{"Name": "TestAgent", "Version": "1.0"}`
	file.WriteString(data) //nolint:errcheck
	file.Close()

	_, err = agent.LoadFromFile(file.Name())
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if agent.GetName() != "TestAgent" || agent.GetVersion() != "1.0" {
		t.Errorf("LoadFromFile did not populate fields correctly")
	}
}

// Test LoadFromFile with non-existent file.
func TestAgent_LoadFromFile_NonExistent(t *testing.T) {
	agent := &Agent{}

	_, err := agent.LoadFromFile("non_existent_file.json")
	if err == nil {
		t.Fatalf("LoadFromFile should have failed for non-existent file")
	}
}

// Test removeDuplicates.
func TestRemoveDuplicates(t *testing.T) {
	input := []string{"a", "b", "a", "c"}
	expected := []string{"a", "b", "c"}

	result := removeDuplicates(input)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("removeDuplicates failed: got %v, want %v", result, expected)
	}
}

// Test firstNonEmptyString.
func TestFirstNonEmptyString(t *testing.T) {
	result := firstNonEmptyString("", "second")
	if result != "second" {
		t.Errorf("firstNonEmptyString failed: got %v, want %v", result, "second")
	}

	result = firstNonEmptyString("first", "second")
	if result != "first" {
		t.Errorf("firstNonEmptyString failed: got %v, want %v", result, "first")
	}
}

// Test mergeItems.
func TestMergeItems(t *testing.T) {
	item1 := &Locator{Type: "type1", Url: "url1"}
	item2 := &Locator{Type: "type2", Url: "url2"}
	item3 := &Locator{Type: "type1", Url: "url1"} // Duplicate of item1

	receiverItems := []*Locator{item1}
	otherItems := []*Locator{item2, item3}

	result := mergeItems(receiverItems, otherItems, func(item *Locator) string {
		return item.Key()
	})

	if len(result) != 2 {
		t.Errorf("mergeItems failed: expected 2 items, got %d", len(result))
	}
}

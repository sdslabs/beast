package api

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/swaggo/swag"
)

func TestSwaggerRoutesMatchAPI(t *testing.T) {
	var spec struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}

	document, err := swag.ReadDoc()
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(document), &spec); err != nil {
		t.Fatal(err)
	}

	ginParameter := regexp.MustCompile(`:(\w+)`)
	expected := make(map[string]bool)
	for _, route := range initGinRouter().Routes() {
		if route.Path == "/dummy" {
			continue
		}
		path := ginParameter.ReplaceAllString(route.Path, "{${1}}")
		expected[strings.ToLower(route.Method)+" "+path] = true
	}

	for path, methods := range spec.Paths {
		for method := range methods {
			operation := method + " " + path
			if !expected[operation] {
				t.Errorf("Swagger lists an unregistered route %s", operation)
			}
			delete(expected, operation)
		}
	}
	for operation := range expected {
		t.Errorf("Swagger is missing %s", operation)
	}

	canonical, err := os.ReadFile("docs/swagger.json")
	if err != nil {
		t.Fatal(err)
	}
	siteCopy, err := os.ReadFile("docs/swagger/swagger.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(canonical) != string(siteCopy) {
		t.Error("Swagger JSON used by the documentation site differs from the canonical spec")
	}
}

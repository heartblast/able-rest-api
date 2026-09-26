package docs

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed openapi.yaml
var OpenAPIYAML []byte

// OpenAPIJSON is derived from the embedded YAML contract, the sole source of truth.
var OpenAPIJSON = mustOpenAPIJSON()

func mustOpenAPIJSON() []byte {
	var document any
	if err := yaml.Unmarshal(OpenAPIYAML, &document); err != nil {
		panic(fmt.Errorf("parse embedded OpenAPI contract: %w", err))
	}
	data, err := json.Marshal(document)
	if err != nil {
		panic(fmt.Errorf("convert embedded OpenAPI contract to JSON: %w", err))
	}
	return data
}

package cli

import "gopkg.in/yaml.v3"

// decodeYAMLVars decodes a flat string map from YAML, used by --vars-file
// when the file doesn't end in .json.
func decodeYAMLVars(data []byte, into map[string]string) error {
	return yaml.Unmarshal(data, &into)
}

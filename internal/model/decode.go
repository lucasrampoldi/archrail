package model

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// DecodeStrict decodes YAML into out, rejecting unknown fields. The file path
// is used only to prefix error messages so users can locate the problem.
func DecodeStrict(file string, data []byte, out any) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}
	return nil
}

// Decode decodes YAML into out without rejecting unknown fields (used where
// inline parameter maps are expected).
func Decode(file string, data []byte, out any) error {
	if err := yaml.Unmarshal(data, out); err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}
	return nil
}

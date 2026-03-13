package yaml

import (
	"bytes"
	"fmt"
	"regexp"
)

const (
	// MaxBodySize is the maximum allowed YAML body size (2 MB).
	MaxBodySize = 2 << 20

	// MaxDocumentCount is the maximum number of documents in a multi-doc YAML.
	MaxDocumentCount = 100

	// maxExpansionRatio is the maximum allowed ratio of parsed JSON size to raw YAML size.
	// A ratio above this threshold indicates a possible YAML bomb (anchor/alias expansion).
	maxExpansionRatio = 100
)

// unsafeTags are YAML language-specific tags that can trigger code execution
// in unsafe parsers. Go's yaml libraries don't execute these, but we reject
// them as defense-in-depth.
var unsafeTags = []string{
	"!!python/",
	"!!ruby/",
	"!!perl/",
	"!!java/",
	"!!js/",
	"!!php/",
	"!!bash/",
}

// anchorAliasPattern matches YAML anchor (&name) and alias (*name) syntax.
// Kubernetes manifests never need anchors or aliases; rejecting them prevents
// YAML bomb attacks (exponential expansion via nested aliases).
var anchorAliasPattern = regexp.MustCompile(`(?m)^\s*[^#]*[&*][a-zA-Z_][a-zA-Z0-9_]*`)

// CheckSecurity performs pre-parse security validation on raw YAML bytes.
// It checks for:
//   - Body size exceeding MaxBodySize
//   - YAML anchors and aliases (bomb prevention)
//   - Unsafe YAML tags (code execution prevention)
func CheckSecurity(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty YAML body")
	}
	if len(data) > MaxBodySize {
		return fmt.Errorf("YAML body exceeds maximum size of %d bytes", MaxBodySize)
	}

	if err := rejectUnsafeTags(data); err != nil {
		return err
	}
	if err := rejectAnchorsAliases(data); err != nil {
		return err
	}

	return nil
}

// CheckExpansionRatio verifies that the parsed output is not suspiciously
// larger than the input, which would indicate a YAML bomb.
func CheckExpansionRatio(inputSize, outputSize int) error {
	if inputSize <= 0 {
		return nil
	}
	ratio := float64(outputSize) / float64(inputSize)
	if ratio > maxExpansionRatio {
		return fmt.Errorf("suspicious expansion ratio %.0fx detected (max %dx), possible YAML bomb", ratio, maxExpansionRatio)
	}
	return nil
}

// rejectUnsafeTags scans raw YAML bytes for language-specific tags that could
// trigger code execution in unsafe parsers.
func rejectUnsafeTags(data []byte) error {
	lower := bytes.ToLower(data)
	for _, tag := range unsafeTags {
		if bytes.Contains(lower, []byte(tag)) {
			return fmt.Errorf("unsafe YAML tag detected: %s", tag)
		}
	}
	return nil
}

// rejectAnchorsAliases scans raw YAML bytes for anchor (&) and alias (*)
// markers. Kubernetes manifests should not contain these.
func rejectAnchorsAliases(data []byte) error {
	if anchorAliasPattern.Match(data) {
		return fmt.Errorf("YAML anchors and aliases are not permitted in Kubernetes manifests")
	}
	return nil
}

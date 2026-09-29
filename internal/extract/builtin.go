package extract

import "regexp"

// Generic is a language-agnostic, regex-based extractor that recognizes the most
// common import forms across ecosystems. It is a safe default when no
// language-specific extractor is configured.
type Generic struct{}

func (Generic) Name() string { return "regex" }

var genericPatterns = []*regexp.Regexp{
	// ES / TS: import ... from 'x' ; import 'x'
	regexp.MustCompile(`(?m)\bimport\b[^'"]*?['"]([^'"]+)['"]`),
	// CommonJS: require('x')
	regexp.MustCompile(`(?m)\brequire\(\s*['"]([^'"]+)['"]\s*\)`),
	// Python: from x import ... ; import x
	regexp.MustCompile(`(?m)^\s*from\s+([a-zA-Z0-9_.]+)\s+import\b`),
	regexp.MustCompile(`(?m)^\s*import\s+([a-zA-Z0-9_.]+)`),
	// Go: import "x"
	regexp.MustCompile(`(?m)^\s*[_\w.]*\s*"([^"]+)"`),
}

func (Generic) Imports(content []byte) []string {
	return matchAll(genericPatterns, content)
}

// JS is a JavaScript/TypeScript-specific extractor.
type JS struct{}

func (JS) Name() string { return "js" }

var jsPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?m)\bimport\b[^'"]*?['"]([^'"]+)['"]`),
	regexp.MustCompile(`(?m)\bexport\b[^'"]*?\bfrom\b\s*['"]([^'"]+)['"]`),
	regexp.MustCompile(`(?m)\brequire\(\s*['"]([^'"]+)['"]\s*\)`),
	regexp.MustCompile(`(?m)\bimport\(\s*['"]([^'"]+)['"]\s*\)`),
}

func (JS) Imports(content []byte) []string {
	return matchAll(jsPatterns, content)
}

// Go is a Go-specific import extractor.
type Go struct{}

func (Go) Name() string { return "go" }

var (
	goSingle = regexp.MustCompile(`(?m)^\s*import\s+(?:[_\w.]+\s+)?"([^"]+)"`)
	goBlock  = regexp.MustCompile(`(?s)import\s*\((.*?)\)`)
	goInBlk  = regexp.MustCompile(`(?m)^\s*(?:[_\w.]+\s+)?"([^"]+)"`)
)

func (Go) Imports(content []byte) []string {
	var out []string
	for _, m := range goSingle.FindAllSubmatch(content, -1) {
		out = append(out, string(m[1]))
	}
	for _, blk := range goBlock.FindAllSubmatch(content, -1) {
		for _, m := range goInBlk.FindAllSubmatch(blk[1], -1) {
			out = append(out, string(m[1]))
		}
	}
	return dedupeSorted(out)
}

func matchAll(patterns []*regexp.Regexp, content []byte) []string {
	var out []string
	for _, re := range patterns {
		for _, m := range re.FindAllSubmatch(content, -1) {
			if len(m) > 1 {
				out = append(out, string(m[1]))
			}
		}
	}
	return dedupeSorted(out)
}

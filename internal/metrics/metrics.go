package metrics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// snakeCasePattern matches lowercase snake_case identifiers as required by the
// metric naming conventions documented in docs/troubleshooting.md.
var snakeCasePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// manifestMetric mirrors the subset of the observability manifest schema that
// is relevant to metric naming conventions.
type manifestMetric struct {
	Name   string   `json:"name"`
	Type   string   `json:"type"`
	Labels []string `json:"labels"`
}

// manifestDocument mirrors the top-level structure of
// docs/observability-manifest.json.
type manifestDocument struct {
	Metrics []manifestMetric `json:"metrics"`
}

// loadObservabilityManifest reads and parses docs/observability-manifest.json
// relative to the repository root (two directories above this package).
func loadObservabilityManifest(t *testing.T) manifestDocument {
	t.Helper()

	path := filepath.Join("..", "..", "docs", "observability-manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read observability manifest %q: %v", path, err)
	}

	var doc manifestDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse observability manifest %q: %v", path, err)
	}
	if len(doc.Metrics) == 0 {
		t.Fatalf("observability manifest %q declares no metrics", path)
	}
	return doc
}

// validateMetricName enforces the documented naming conventions for a single
// metric and returns a list of human-readable violations.
func validateMetricName(m manifestMetric) []string {
	var violations []string

	if !snakeCasePattern.MatchString(m.Name) {
		violations = append(violations, "name is not lowercase snake_case: "+m.Name)
	}

	switch strings.ToLower(m.Type) {
	case "counter":
		if !strings.HasSuffix(m.Name, "_total") {
			violations = append(violations, "counter name does not end with _total: "+m.Name)
		}
	case "gauge", "histogram":
		// Time-based gauges and histograms must express their unit in the name.
		if !strings.HasSuffix(m.Name, "_seconds") {
			violations = append(violations, strings.ToLower(m.Type)+" name does not end with _seconds: "+m.Name)
		}
	}

	for _, label := range m.Labels {
		if !snakeCasePattern.MatchString(label) {
			violations = append(violations, "label is not lowercase snake_case: "+label+" (metric "+m.Name+")")
		}
	}

	return violations
}

// TestObservabilityManifestMetricNamingConventions asserts that every metric
// declared in docs/observability-manifest.json follows the naming conventions
// documented in the troubleshooting guide.
func TestObservabilityManifestMetricNamingConventions(t *testing.T) {
	doc := loadObservabilityManifest(t)

	for _, m := range doc.Metrics {
		for _, violation := range validateMetricName(m) {
			t.Errorf("metric naming convention violation: %s", violation)
		}
	}
}

// TestValidateMetricNameNegative demonstrates the expected failure path for a
// non-conforming metric so the convention is enforceable.
func TestValidateMetricNameNegative(t *testing.T) {
	bad := manifestMetric{
		Name:   "RequestCount",
		Type:   "counter",
		Labels: []string{"StatusCode"},
	}

	violations := validateMetricName(bad)
	if len(violations) == 0 {
		t.Fatalf("expected violations for non-conforming metric %q, got none", bad.Name)
	}

	joined := strings.Join(violations, "\n")
	for _, want := range []string{
		"name is not lowercase snake_case",
		"counter name does not end with _total",
		"label is not lowercase snake_case",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected violation containing %q, got:\n%s", want, joined)
		}
	}
}

// TestValidateMetricNameConforming demonstrates the successful path for a
// metric that follows the documented conventions.
func TestValidateMetricNameConforming(t *testing.T) {
	good := manifestMetric{
		Name:   "http_requests_total",
		Type:   "counter",
		Labels: []string{"method", "status_code"},
	}

	if violations := validateMetricName(good); len(violations) != 0 {
		t.Fatalf("expected no violations for conforming metric %q, got: %v", good.Name, violations)
	}
}
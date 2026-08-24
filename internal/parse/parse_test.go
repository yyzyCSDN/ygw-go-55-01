package parse

import "testing"

func TestParseBasicLines(t *testing.T) {
	parser := NewParser()
	samples, err := parser.ParseBody([]byte("# comment\ncpu_usage 1.5 1000\nmem_bytes 2048 1000\n"), "node-a")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(samples) != 2 {
		t.Fatalf("expected 2 samples, got %d", len(samples))
	}
	if samples[0].Name != "cpu_usage" || samples[0].Value != 1.5 {
		t.Fatalf("unexpected first sample: %+v", samples[0])
	}
	if samples[0].Source != "node-a" {
		t.Fatalf("source not attached: %+v", samples[0])
	}
}

func TestParseRejectsBadValue(t *testing.T) {
	parser := NewParser()
	if _, err := parser.ParseBody([]byte("cpu_usage not-a-number 1000\n"), "node-a"); err == nil {
		t.Fatal("expected bad value to fail parsing")
	}
}

func TestNormalizeName(t *testing.T) {
	cases := map[string]bool{
		"cpu_usage":   true,
		"http_2xx":    true,
		"bad-name":    false,
		"has space":   false,
	}
	for name, want := range cases {
		got := NormalizeName(name) != ""
		if got != want {
			t.Fatalf("NormalizeName(%q) accepted=%v want=%v", name, got, want)
		}
	}
}

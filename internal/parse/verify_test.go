package parse

import "testing"

func TestParseHalfLineResumeCorrect(t *testing.T) {
	parser := NewParser()
	reader := NewStreamReader(parser, "node-a")

	first, err := reader.Feed([]byte("cpu_usage 1.0 1000\nmem_bytes 2048 1000\nhalf"), false)
	if err != nil {
		t.Fatalf("first chunk failed: %v", err)
	}
	if len(first) != 2 {
		t.Fatalf("first chunk should yield two complete lines, got %d", len(first))
	}

	second, err := reader.Feed([]byte("line_used 3.5 1000\n"), true)
	if err != nil {
		t.Fatalf("resume chunk failed: %v", err)
	}
	if len(second) != 1 {
		t.Fatalf("resume should complete the carried half line, got %d samples", len(second))
	}
	if second[0].Name != "halfline_used" || second[0].Value != 3.5 {
		t.Fatalf("resumed sample misaligned: %+v", second[0])
	}
}

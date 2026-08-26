package metric

import "testing"

func TestRegistryCounter(t *testing.T) {
	registry := NewRegistry()
	registry.Inc("a")
	registry.Add("a", 2)
	registry.Set("gauge", 5)
	if registry.Get("a") != 3 {
		t.Fatalf("expected counter a=3, got %v", registry.Get("a"))
	}
	snapshot := registry.Snapshot()
	if snapshot["gauge"] != 5 {
		t.Fatalf("expected gauge=5, got %v", snapshot["gauge"])
	}
}

func TestTargetSlotInRange(t *testing.T) {
	for _, id := range []string{"a", "b", "node-a", "node-b"} {
		slot := TargetSlot(id, 3)
		if slot < 0 || slot >= 3 {
			t.Fatalf("slot %d out of range for %q", slot, id)
		}
	}
	if TargetSlot("anything", 1) != 0 {
		t.Fatal("single slot must always be 0")
	}
}

package discover

import (
	"testing"

	"scrapehub/internal/model"
)

func TestStaticProviderAddRemoveList(t *testing.T) {
	provider := NewStaticProvider()
	provider.Add(model.Target{ID: "b", Endpoint: "http://b/metrics"})
	provider.Add(model.Target{ID: "a", Endpoint: "http://a/metrics"})
	list := provider.List()
	if len(list) != 2 || list[0].ID != "a" || list[1].ID != "b" {
		t.Fatalf("unexpected list order: %+v", list)
	}
	provider.Remove("a")
	list = provider.List()
	if len(list) != 1 || list[0].ID != "b" {
		t.Fatalf("remove failed: %+v", list)
	}
}

func TestListStoreSnapshotGeneration(t *testing.T) {
	provider := NewStaticProvider(model.Target{ID: "a", Endpoint: "http://a/metrics"})
	store := NewListStore(NewRefresher(provider))
	first := store.Refresh()
	if first.Generation != 1 || len(first.Targets) != 1 {
		t.Fatalf("first refresh unexpected: %+v", first)
	}
	second := store.Refresh()
	if second.Generation != 2 {
		t.Fatalf("second refresh should bump generation, got %d", second.Generation)
	}
	snapshot := store.Snapshot()
	if snapshot.Generation != 2 {
		t.Fatalf("snapshot should see generation 2, got %d", snapshot.Generation)
	}
}

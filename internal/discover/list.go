package discover

import "scrapehub/internal/model"

// Snapshot is a point-in-time view of the discovered targets.
type Snapshot struct {
	Targets    []model.Target
	Generation int
	CreatedAt  int64
}

// TargetIDs returns the IDs of every target in the snapshot.
func (s Snapshot) TargetIDs() []string {
	ids := make([]string, 0, len(s.Targets))
	for _, target := range s.Targets {
		ids = append(ids, target.ID)
	}
	return ids
}

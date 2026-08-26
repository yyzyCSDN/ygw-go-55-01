package metric

import "github.com/cespare/xxhash/v2"

// TargetSlot maps a target ID to a stable worker slot using a fast hash.
func TargetSlot(id string, slots int) int {
	if slots <= 0 {
		return 0
	}
	return int(xxhash.Sum64String(id) % uint64(slots))
}

package schedule

// windowIndex maps the i-th target of a sorted list to its stagger window.
// The list is split into slots windows of `per` targets; the first window
// covers [0, per), the second [per, 2*per) and so on, so boundaries are
// continuous and no target can fall into two windows.
func windowIndex(i, per, slots int) int {
	if per <= 0 {
		return 0
	}
	index := i / per
	if index >= slots {
		return slots - 1
	}
	return index
}

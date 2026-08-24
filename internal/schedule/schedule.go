package schedule

import (
	"context"
	"sync"
	"time"

	"scrapehub/internal/discover"
	"scrapehub/internal/health"
	"scrapehub/internal/metric"
	"scrapehub/internal/model"
)

// Dispatcher runs one scrape for a target.
type Dispatcher interface {
	Dispatch(ctx context.Context, target model.Target, cycle int64) error
}

// HealthView exposes the latest health snapshot to the scheduler.
type HealthView interface {
	Snapshot() health.Snapshot
	Record(id string, ok bool, now time.Time) model.HealthState
}

// Scheduler turns discovered targets into dispatch plans on a fixed interval.
type Scheduler struct {
	store      *discover.ListStore
	stagger    *Stagger
	dispatcher Dispatcher
	health     HealthView
	metric     *metric.Registry
	interval   time.Duration
	slots      int
	history    *JobHistory

	mu sync.Mutex
}

// NewScheduler wires a scheduler to its dependencies.
func NewScheduler(
	store *discover.ListStore,
	stagger *Stagger,
	dispatcher Dispatcher,
	health HealthView,
	registry *metric.Registry,
	interval time.Duration,
	slots int,
	history *JobHistory,
) *Scheduler {
	return &Scheduler{
		store:      store,
		stagger:    stagger,
		dispatcher: dispatcher,
		health:     health,
		metric:     registry,
		interval:   interval,
		slots:      slots,
		history:    history,
	}
}

// RunCycle executes one dispatch pass over the current target list.
func (s *Scheduler) RunCycle(now time.Time) CycleReport {
	s.mu.Lock()
	defer s.mu.Unlock()
	started := time.Now()

	report := CycleReport{Cycle: now.UnixMilli()}
	snapshot := s.currentSnapshot()
	targets := snapshot.Targets
	windows := s.stagger.Split(targets, s.slots)
	plan := spreadTargets(buildPlan(windows), s.slots)
	healthSnapshot := s.health.Snapshot()

	for _, target := range plan {
		if !healthSnapshot.Allow(target.ID) {
			report.Skipped = append(report.Skipped, target.ID)
			s.metric.Inc(metric.MetricSkipped)
			continue
		}
		err := s.dispatcher.Dispatch(context.Background(), target, report.Cycle)
		if err != nil {
			report.Failed = append(report.Failed, target.ID)
			s.metric.Inc(metric.MetricScrapeFailures)
			s.health.Record(target.ID, false, now)
			s.recordJob(target, report.Cycle, model.TaskFailed, 1, err.Error())
			continue
		}
		report.Dispatched = append(report.Dispatched, target.ID)
		s.metric.Inc(metric.MetricDispatched)
		s.health.Record(target.ID, true, now)
		s.recordJob(target, report.Cycle, model.TaskForwarded, 1, "")
	}

	removed := s.drainRemoved()
	report.Removed = removed
	report.Planned = len(plan)
	report.Generation = snapshot.Generation
	report.Slots = s.stagger.Windows()
	report.Duration = time.Since(started)
	return report
}

// currentSnapshot returns the latest discovery snapshot so a target that has
// been added or removed since the previous cycle is reflected immediately in
// the dispatch plan.
func (s *Scheduler) currentSnapshot() discover.Snapshot {
	return s.store.Snapshot()
}

// recordJob writes a task record into the history ring.
func (s *Scheduler) recordJob(target model.Target, cycle int64, state model.TaskState, attempt int, message string) {
	if s.history == nil {
		return
	}
	if state == model.TaskFailed {
		s.history.RecordFailure(target.ID, cycle, attempt, message)
		return
	}
	s.history.RecordDispatch(target.ID, cycle, attempt)
}

// drainRemoved physically removes targets the health judge marked as removed.
func (s *Scheduler) drainRemoved() []string {
	var removed []string
	for _, id := range s.store.Snapshot().TargetIDs() {
		if s.health.Snapshot().State(id) == model.HealthRemoved {
			s.store.Remove(id)
			removed = append(removed, id)
		}
	}
	return removed
}

// Interval returns the configured cycle interval.
func (s *Scheduler) Interval() time.Duration {
	return s.interval
}

// walkTargets drains the discovery store once per target using the cursor,
// guarding against accidental duplicates.
func walkTargets(store *discover.ListStore, cycle int64, budget int) []model.Target {
	targets := make([]model.Target, 0, budget)
	seen := make(map[string]struct{}, budget)
	for i := 0; i < budget; i++ {
		target, ok := store.Next(cycle)
		if !ok {
			break
		}
		if _, exists := seen[target.ID]; exists {
			continue
		}
		seen[target.ID] = struct{}{}
		targets = append(targets, target)
	}
	return targets
}

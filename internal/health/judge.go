package health

import "scrapehub/internal/model"

// evaluate advances one target's health state based on the latest result.
// A success always resets the failure streak; failures first demote the
// target to unhealthy and only remove it after enough consecutive failures.
func evaluate(current model.HealthState, streak int, ok bool, threshold int) model.HealthState {
	if current == model.HealthRemoved {
		return model.HealthRemoved
	}
	if ok {
		return model.HealthHealthy
	}
	return model.HealthRemoved
}

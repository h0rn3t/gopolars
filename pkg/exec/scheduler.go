package exec

import (
	"runtime"

	"github.com/h0rn3t/gopolars/pkg/plan/physical"
)

type Scheduler struct {
	Workers int
}

func NewScheduler() Scheduler {
	return Scheduler{Workers: runtime.GOMAXPROCS(0)}
}

func (s Scheduler) Run(plan physical.Plan, run func(op physical.Operator) error) error {
	for _, op := range plan.Operators {
		done := make(chan error, 1)
		go func(current physical.Operator) {
			done <- run(current)
		}(op)
		if err := <-done; err != nil {
			return err
		}
	}
	return nil
}

package scheduler

import "fmt"

func NewScheduler(algorithm string, config SchedulerConfig) (Scheduler, error) {
	switch algorithm {
	case "round-robin":
		return NewRoundRobinScheduler(config), nil
	case "cooperative":
		return NewCooperativeScheduler(config), nil
	case "priority":
		return NewPriorityScheduler(config), nil
	default:
		return nil, fmt.Errorf("unknown scheduler algorithm: %s", algorithm)
	}
}

func GetAvailableAlgorithms() []string {
	return []string{"round-robin", "cooperative", "priority"}
}

func GetDefaultConfig() SchedulerConfig {
	return SchedulerConfig{
		Algorithm:  "round-robin",
		TimeSlice:  100 * 1000 * 1000, // 100ms in nanoseconds
		ResourceLimits: ResourceLimits{
			MaxMemory:  1024 * 1024 * 64, // 64MB
			MaxCPUTime: 30 * 1000 * 1000 * 1000, // 30s in nanoseconds
			MaxTasks:   100,
			TimeSlice:  100 * 1000 * 1000, // 100ms
		},
		MetricsEnabled: true,
	}
}
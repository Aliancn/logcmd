package cmd

import "runtime"

func workerCount(total int) int {
	if total <= 1 {
		if total <= 0 {
			return 1
		}
		return 1
	}

	workers := runtime.NumCPU()
	if workers < 2 {
		workers = 2
	}
	if workers > 8 {
		workers = 8
	}

	if total < workers {
		return total
	}

	return workers
}

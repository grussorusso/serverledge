package scheduling

import (
	"time"
)

type TimeOp struct {
	time time.Time
	name string
}

type ExpiringList struct {
	windowSize time.Duration
	list       []TimeOp
}

func createExpiringList(windowSize time.Duration) ExpiringList {
	return ExpiringList{windowSize: windowSize, list: make([]TimeOp, 0)}
}

func (e *ExpiringList) Add(name string, arrival time.Time) {
	current := TimeOp{time: arrival, name: name}

	e.DeleteExpired(time.Now())
	e.list = append(e.list, current)
}

func (e *ExpiringList) GetList() []string {
	//e.DeleteExpired(now)
	names := make([]string, len(e.list))

	for i := 0; i < len(e.list); i++ {
		names[i] = e.list[i].name
	}

	return names
}

func (e *ExpiringList) DeleteExpired(now time.Time) {
	tmp := e.list
	var p int

	windowStart := now.Add(-e.windowSize)
	//windowStart := now

	index := 0
	for {
		if len(tmp) == 0 {
			break
		}

		if len(tmp) == 1 {
			a := tmp[0]
			if a.time.Before(windowStart) || a.time.Equal(windowStart) {
				index++
				break
			}
		}

		p = len(tmp) / 2
		a := tmp[p]

		if a.time.Equal(windowStart) {
			index += p + 1
			break
		}

		if a.time.Before(windowStart) {
			index += p
			tmp = tmp[p:]
		} else {
			tmp = tmp[:p]
		}
	}

	e.list = e.list[index:]
}

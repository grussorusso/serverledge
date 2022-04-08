package scheduling

import (
	"sync/atomic"
	"time"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/node"
)

type DropManager struct {
	dropChan   chan time.Time
	dropCount  int64
	expiration int64
}

func InitDropManager() *DropManager {
	dropManager := &DropManager{
		dropCount:  0,
		dropChan:   make(chan time.Time, 1),
		expiration: time.Now().UnixNano(),
	}

	go dropManager.dropRun()
	return dropManager
}

func (d *DropManager) sendDropAlert() {
	dropTime := time.Now()
	if dropTime.UnixNano() > d.expiration {
		select { //non-blocking write on channel
		case d.dropChan <- dropTime:
			return
		default:
			return
		}
	}
}

func (d *DropManager) dropRun() {
	var expirationInterval = time.Duration(config.GetInt(config.DROP_PERIOD, 30))
	ticker := time.NewTicker(time.Duration(config.GetInt(config.DROP_PERIOD, 30)) * time.Second)
	for {
		select {
		case tick := <-d.dropChan:
			//update expiration
			d.expiration = tick.Add(expirationInterval * time.Second).UnixNano()
			d.dropCount++
			atomic.StoreInt64(&node.Resources.DropCount, d.dropCount)
		case <-ticker.C:
			if time.Now().UnixNano() >= d.expiration {
				d.dropCount = 0
				atomic.StoreInt64(&node.Resources.DropCount, d.dropCount)
			}
		}
	}
}

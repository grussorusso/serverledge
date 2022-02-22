package scheduling

import (
	"github.com/grussorusso/serverledge/internal/config"
	"log"
	"sync/atomic"
	"time"
)

type DropManager struct {
	dropChan   chan time.Time
	dropCount  int64
	expiration int64
}

var dropManager *DropManager

func InitDropManager() {
	dropManager = &DropManager{
		dropCount:  0,
		dropChan:   make(chan time.Time, 1),
		expiration: time.Now().UnixNano(),
	}

	go dropManager.dropRun()

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
	ticker := time.NewTicker(time.Duration(config.GetInt("policy.drop.expiration", 30)) * time.Second)
	for {
		select {
		case tick := <-d.dropChan:
			log.Printf("drop occurred")
			//update expiration
			d.expiration = tick.Add(expirationInterval * time.Second).UnixNano()
			d.dropCount++
			atomic.StoreInt64(&Node.DropCount, d.dropCount)
		case <-ticker.C:
			if time.Now().UnixNano() >= d.expiration {
				d.dropCount = 0
				atomic.StoreInt64(&Node.DropCount, d.dropCount)
			}
		}
	}
}

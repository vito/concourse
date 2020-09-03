package db

import (
	"sync"
	"time"

	"code.cloudfoundry.org/clock"
	"golang.org/x/time/rate"
)

type CheckableCounter struct {
	conn Conn

	clock     clock.Clock
	limiter   *rate.Limiter
	lastCount int
	countLock *sync.Mutex
}

func NewCheckableCounter(conn Conn, c clock.Clock, refreshInterval time.Duration) *CheckableCounter {
	return &CheckableCounter{
		conn:      conn,
		clock:     c,
		limiter:   rate.NewLimiter(rate.Every(refreshInterval), 1),
		countLock: new(sync.Mutex),
	}
}

// Returns the number of resource config scopes in the database. This
// represents the number of things that can be checked by lidar.
func (c *CheckableCounter) CheckableCount() (int, error) {
	c.countLock.Lock()
	defer c.countLock.Unlock()

	if c.limiter.AllowN(c.clock.Now(), 1) {
		err := psql.Select("COUNT(id)").
			From("resource_config_scopes").
			RunWith(c.conn).
			QueryRow().
			Scan(&c.lastCount)
		if err != nil {
			return 0, err
		}
	}

	return c.lastCount, nil
}

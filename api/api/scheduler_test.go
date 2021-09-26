package api

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/go-redis/redis/v8"
)

type redisNoSET struct {
	redis.Cmdable
}

func (r *redisNoSET) Set(
	ctx context.Context,
	key string,
	val interface{},
	ttl time.Duration,
) *redis.StatusCmd {
	return redis.NewStatusResult("set-fails", fmt.Errorf("SET failure"))
}

func TestScheduleFailsOnSETError(t *testing.T) {
	s   := NewScheduler(&redisNoSET{})
	err := s.Schedule(context.Background(), "<pid>", &QueryPlan{})
	msg := "SET failure"
	assert.EqualErrorf(t, err, msg, "want err = %v; was %v", msg, err)
}

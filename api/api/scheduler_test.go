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

type redisContextAware struct {
	redis.Cmdable
}

func (r *redisContextAware) Set(
	ctx context.Context,
	key string,
	val interface{},
	ttl time.Duration,
) *redis.StatusCmd {
	return redis.NewStatusResult("-", ctx.Err())
}

func TestSetCompletedCalledIfContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ctxaware    := &redisContextAware{}
	cancel()
	s   := NewScheduler(ctxaware)
	err := s.Schedule(ctx, "<pid>", &QueryPlan{})
	msg := "context canceled"
	assert.EqualErrorf(t, err, msg, "want err = %v; was %v", msg, err)
}

type redisNoXADD struct {
	redis.Cmdable
}

func (r *redisNoXADD) Set(
	ctx context.Context,
	key string,
	val interface{},
	ttl time.Duration,
) *redis.StatusCmd {
	return redis.NewStatusResult("OK", nil)
}

func (r *redisNoXADD) XAdd(
	ctx  context.Context,
	args *redis.XAddArgs,
) *redis.StringCmd {
	return redis.NewStringResult("xadd-fails", fmt.Errorf("XADD failure"))
}

func TestScheduleFailsOnXADDError(t *testing.T) {
	s   := NewScheduler(&redisNoXADD{})
	qp  := &QueryPlan{plan: make([][]byte, 2)}
	err := s.Schedule(context.Background(), "<pid>", qp)
	msg := "XADD failure"
	assert.Containsf(t, err.Error(), msg, "want err = %v; was %v", msg, err)
}


func TestErrorOnDisconnectedClient(t *testing.T) {
	dcd := redis.NewClient(&redis.Options{})
	s   := NewScheduler(dcd)
	qp  := &QueryPlan{plan: make([][]byte, 2)}
	err := s.Schedule(context.Background(), "<pid>", qp)
	assert.Error(t, err, "Scheduling on disconnected redis did not fail")
}

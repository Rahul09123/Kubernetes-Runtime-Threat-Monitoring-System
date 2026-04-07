package common

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type Queue struct {
	conn *nats.Conn
}

func NewQueue(url string) (*Queue, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("connect nats: %w", err)
	}
	return &Queue{conn: nc}, nil
}

func (q *Queue) Close() {
	if q != nil && q.conn != nil {
		q.conn.Close()
	}
}

func (q *Queue) Publish(ctx context.Context, subject string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	if err := q.conn.Publish(subject, b); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	flushCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		flushCtx, cancel = context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
	}
	if err := q.conn.FlushWithContext(flushCtx); err != nil {
		return fmt.Errorf("flush publish: %w", err)
	}
	return nil
}

func (q *Queue) Subscribe(subject string, handler func([]byte)) (*nats.Subscription, error) {
	sub, err := q.conn.Subscribe(subject, func(msg *nats.Msg) {
		handler(msg.Data)
	})
	if err != nil {
		return nil, fmt.Errorf("subscribe %s: %w", subject, err)
	}
	return sub, nil
}

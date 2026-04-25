package initiator_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"yadro.com/course/search/adapters/initiator"
)

var testLog = slog.New(slog.NewTextHandler(os.Stderr, nil))

type mockBuilder struct {
	calls int
	err   error
}

func (m *mockBuilder) BuildIndex(_ context.Context) error {
	m.calls++
	return m.err
}

// New возвращает ненулевой указатель на Initiator
func TestNew_ReturnsNonNil(t *testing.T) {
	i := initiator.New(testLog, &mockBuilder{}, time.Minute)
	require.NotNil(t, i)
}

// Run вызывает BuildIndex сразу при старте до первого тика
func TestRun_CallsBuildOnStart(t *testing.T) {
	b := &mockBuilder{}
	i := initiator.New(testLog, b, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		i.Run(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	require.GreaterOrEqual(t, b.calls, 1)
}

// Run не паникует если BuildIndex возвращает ошибку
func TestRun_BuildErrorDoesNotPanic(t *testing.T) {
	b := &mockBuilder{err: errors.New("index failed")}
	i := initiator.New(testLog, b, time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	require.NotPanics(t, func() { i.Run(ctx) })
	require.GreaterOrEqual(t, b.calls, 1)
}

// Run перестраивает индекс повторно по тику ttl
func TestRun_RebuildOnTick(t *testing.T) {
	b := &mockBuilder{}
	i := initiator.New(testLog, b, 30*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	i.Run(ctx)
	require.GreaterOrEqual(t, b.calls, 2)
}

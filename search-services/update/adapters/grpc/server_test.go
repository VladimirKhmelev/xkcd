package grpc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	updatepb "yadro.com/course/proto/update"
	grpcadapter "yadro.com/course/update/adapters/grpc"
	"yadro.com/course/update/core"
)

type mockUpdater struct {
	updateErr error
	statsRes  core.ServiceStats
	statsErr  error
	dropErr   error
	st        core.ServiceStatus
}

func (m *mockUpdater) Update(_ context.Context) error { return m.updateErr }
func (m *mockUpdater) Stats(_ context.Context) (core.ServiceStats, error) {
	return m.statsRes, m.statsErr
}
func (m *mockUpdater) Status(_ context.Context) core.ServiceStatus { return m.st }
func (m *mockUpdater) Drop(_ context.Context) error                { return m.dropErr }

// Ping всегда возвращает пустой ответ без ошибки
func TestPing(t *testing.T) {
	srv := grpcadapter.NewServer(&mockUpdater{})
	_, err := srv.Ping(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
}

// Статус idle корректно маппится в gRPC-константу
func TestStatus_Idle(t *testing.T) {
	srv := grpcadapter.NewServer(&mockUpdater{st: core.StatusIdle})
	resp, err := srv.Status(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, updatepb.Status_STATUS_IDLE, resp.Status)
}

// Статус running корректно маппится в gRPC-константу
func TestStatus_Running(t *testing.T) {
	srv := grpcadapter.NewServer(&mockUpdater{st: core.StatusRunning})
	resp, err := srv.Status(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, updatepb.Status_STATUS_RUNNING, resp.Status)
}

// Успешное обновление возвращает пустой ответ без ошибки
func TestUpdate_OK(t *testing.T) {
	srv := grpcadapter.NewServer(&mockUpdater{})
	_, err := srv.Update(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
}

// Если обновление уже запущено — возвращается код Aborted
func TestUpdate_AlreadyRunning(t *testing.T) {
	srv := grpcadapter.NewServer(&mockUpdater{updateErr: core.ErrUpdateRunning})
	_, err := srv.Update(context.Background(), &emptypb.Empty{})
	require.Error(t, err)
	require.Equal(t, codes.Aborted, status.Code(err))
}

// Произвольная ошибка сервиса возвращается как Internal
func TestUpdate_InternalError(t *testing.T) {
	srv := grpcadapter.NewServer(&mockUpdater{updateErr: errors.New("boom")})
	_, err := srv.Update(context.Background(), &emptypb.Empty{})
	require.Equal(t, codes.Internal, status.Code(err))
}

// Статистика корректно преобразуется в proto-ответ
func TestStats_OK(t *testing.T) {
	srv := grpcadapter.NewServer(&mockUpdater{statsRes: core.ServiceStats{ComicsTotal: 42}})
	resp, err := srv.Stats(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, int64(42), resp.ComicsTotal)
}

// Ошибка получения статистики возвращается как Internal
func TestStats_Error(t *testing.T) {
	srv := grpcadapter.NewServer(&mockUpdater{statsErr: errors.New("db down")})
	_, err := srv.Stats(context.Background(), &emptypb.Empty{})
	require.Equal(t, codes.Internal, status.Code(err))
}

// Успешный drop возвращает пустой ответ без ошибки
func TestDrop_OK(t *testing.T) {
	srv := grpcadapter.NewServer(&mockUpdater{})
	_, err := srv.Drop(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
}

// Ошибка drop возвращается как Internal
func TestDrop_Error(t *testing.T) {
	srv := grpcadapter.NewServer(&mockUpdater{dropErr: errors.New("fail")})
	_, err := srv.Drop(context.Background(), &emptypb.Empty{})
	require.Equal(t, codes.Internal, status.Code(err))
}

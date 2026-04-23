package update_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"yadro.com/course/api/adapters/update"
	"yadro.com/course/api/core"
	updatepb "yadro.com/course/proto/update"
)

var log = slog.New(slog.NewTextHandler(os.Stderr, nil))

// mockUpdateClient имитирует gRPC UpdateClient без реального сетевого соединения
type mockUpdateClient struct {
	pingErr    error
	statusResp *updatepb.StatusReply
	statusErr  error
	statsResp  *updatepb.StatsReply
	statsErr   error
	updateErr  error
	dropErr    error
}

func (m *mockUpdateClient) Ping(_ context.Context, _ *emptypb.Empty, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, m.pingErr
}

func (m *mockUpdateClient) Status(_ context.Context, _ *emptypb.Empty, _ ...grpc.CallOption) (*updatepb.StatusReply, error) {
	return m.statusResp, m.statusErr
}

func (m *mockUpdateClient) Stats(_ context.Context, _ *emptypb.Empty, _ ...grpc.CallOption) (*updatepb.StatsReply, error) {
	return m.statsResp, m.statsErr
}

func (m *mockUpdateClient) Update(_ context.Context, _ *emptypb.Empty, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, m.updateErr
}

func (m *mockUpdateClient) Drop(_ context.Context, _ *emptypb.Empty, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, m.dropErr
}

// Создаёт update.Client с подменённым gRPC-клиентом
func newClient(mock updatepb.UpdateClient) *update.Client {
	return update.NewClientWithGRPC(mock, log)
}

// Проверяет, что Ping возвращает nil при успешном ответе сервера
func TestPing_OK(t *testing.T) {
	c := newClient(&mockUpdateClient{})
	require.NoError(t, c.Ping(context.Background()))
}

// Проверяет, что ошибка gRPC пробрасывается наружу
func TestPing_Error(t *testing.T) {
	c := newClient(&mockUpdateClient{pingErr: fmt.Errorf("unavailable")})
	require.Error(t, c.Ping(context.Background()))
}

// Проверяет, что статус STATUS_IDLE преобразуется в core.StatusUpdateIdle
func TestStatus_Idle(t *testing.T) {
	c := newClient(&mockUpdateClient{
		statusResp: &updatepb.StatusReply{Status: updatepb.Status_STATUS_IDLE},
	})
	s, err := c.Status(context.Background())
	require.NoError(t, err)
	require.Equal(t, core.StatusUpdateIdle, s)
}

// Проверяет, что статус STATUS_RUNNING преобразуется в core.StatusUpdateRunning
func TestStatus_Running(t *testing.T) {
	c := newClient(&mockUpdateClient{
		statusResp: &updatepb.StatusReply{Status: updatepb.Status_STATUS_RUNNING},
	})
	s, err := c.Status(context.Background())
	require.NoError(t, err)
	require.Equal(t, core.StatusUpdateRunning, s)
}

// Проверяет, что неизвестный статус возвращает ошибку
func TestStatus_Unknown(t *testing.T) {
	c := newClient(&mockUpdateClient{
		statusResp: &updatepb.StatusReply{Status: updatepb.Status_STATUS_UNSPECIFIED},
	})
	_, err := c.Status(context.Background())
	require.Error(t, err)
}

// Проверяет, что ошибка gRPC при запросе статуса пробрасывается наружу
func TestStatus_Error(t *testing.T) {
	c := newClient(&mockUpdateClient{statusErr: fmt.Errorf("rpc error")})
	_, err := c.Status(context.Background())
	require.Error(t, err)
}

// Проверяет, что Stats корректно преобразует proto-ответ в core.UpdateStats
func TestStats_OK(t *testing.T) {
	c := newClient(&mockUpdateClient{
		statsResp: &updatepb.StatsReply{
			WordsTotal:    100,
			WordsUnique:   80,
			ComicsTotal:   50,
			ComicsFetched: 45,
		},
	})
	stats, err := c.Stats(context.Background())
	require.NoError(t, err)
	require.Equal(t, 100, stats.WordsTotal)
	require.Equal(t, 80, stats.WordsUnique)
	require.Equal(t, 50, stats.ComicsTotal)
	require.Equal(t, 45, stats.ComicsFetched)
}

// Проверяет, что ошибка gRPC при запросе статистики пробрасывается наружу
func TestStats_Error(t *testing.T) {
	c := newClient(&mockUpdateClient{statsErr: fmt.Errorf("stats unavailable")})
	_, err := c.Stats(context.Background())
	require.Error(t, err)
}

// Проверяет, что успешный запрос на обновление возвращает nil
func TestUpdate_OK(t *testing.T) {
	c := newClient(&mockUpdateClient{})
	require.NoError(t, c.Update(context.Background()))
}

// Проверяет, что gRPC-статус Aborted преобразуется в core.ErrAlreadyExists
func TestUpdate_AlreadyRunning(t *testing.T) {
	c := newClient(&mockUpdateClient{
		updateErr: status.Error(codes.Aborted, "already running"),
	})
	err := c.Update(context.Background())
	require.ErrorIs(t, err, core.ErrAlreadyExists)
}

// Проверяет, что прочие ошибки gRPC пробрасываются без преобразования
func TestUpdate_Error(t *testing.T) {
	c := newClient(&mockUpdateClient{
		updateErr: status.Error(codes.Internal, "internal error"),
	})
	err := c.Update(context.Background())
	require.Error(t, err)
	require.NotErrorIs(t, err, core.ErrAlreadyExists)
}

// Проверяет, что успешный запрос на удаление базы возвращает nil
func TestDrop_OK(t *testing.T) {
	c := newClient(&mockUpdateClient{})
	require.NoError(t, c.Drop(context.Background()))
}

// Проверяет, что ошибка gRPC при удалении пробрасывается наружу
func TestDrop_Error(t *testing.T) {
	c := newClient(&mockUpdateClient{dropErr: fmt.Errorf("drop failed")})
	require.Error(t, c.Drop(context.Background()))
}

package words_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"yadro.com/course/api/adapters/words"
	wordspb "yadro.com/course/proto/words"
)

var log = slog.New(slog.NewTextHandler(os.Stderr, nil))

// Иимитирует gRPC WordsClient без реального сетевого соединения
type mockWordsClient struct {
	pingErr  error
	normResp *wordspb.WordsReply
	normErr  error
}

func (m *mockWordsClient) Ping(_ context.Context, _ *emptypb.Empty, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, m.pingErr
}

func (m *mockWordsClient) Norm(_ context.Context, _ *wordspb.WordsRequest, _ ...grpc.CallOption) (*wordspb.WordsReply, error) {
	return m.normResp, m.normErr
}

// newClient создаёт words.Client с подменённым gRPC-клиентом
func newClient(mock wordspb.WordsClient) *words.Client {
	return words.NewClientWithGRPC(mock, log)
}

// Проверяет, что Ping возвращает nil при успешном ответе сервера
func TestPing_OK(t *testing.T) {
	c := newClient(&mockWordsClient{})
	require.NoError(t, c.Ping(context.Background()))
}

// Проверяет, что ошибка gRPC при пинге пробрасывается наружу
func TestPing_Error(t *testing.T) {
	c := newClient(&mockWordsClient{pingErr: fmt.Errorf("unavailable")})
	require.Error(t, c.Ping(context.Background()))
}

// Проверяет, что Norm возвращает корректный список нормализованных слов
func TestNorm_OK(t *testing.T) {
	c := newClient(&mockWordsClient{
		normResp: &wordspb.WordsReply{Words: []string{"linux", "kernel"}},
	})
	result, err := c.Norm(context.Background(), "linux kernel")
	require.NoError(t, err)
	require.Equal(t, []string{"linux", "kernel"}, result)
}

// Проверяет, что Norm возвращает пустой срез при пустом ответе сервера
func TestNorm_Empty(t *testing.T) {
	c := newClient(&mockWordsClient{normResp: &wordspb.WordsReply{}})
	result, err := c.Norm(context.Background(), "")
	require.NoError(t, err)
	require.Empty(t, result)
}

// Проверяет, что ошибка gRPC при нормализации пробрасывается наружу
func TestNorm_Error(t *testing.T) {
	c := newClient(&mockWordsClient{normErr: fmt.Errorf("norm failed")})
	_, err := c.Norm(context.Background(), "linux")
	require.Error(t, err)
}

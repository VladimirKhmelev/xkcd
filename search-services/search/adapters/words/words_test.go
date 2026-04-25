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
	wordspb "yadro.com/course/proto/words"
	"yadro.com/course/search/adapters/words"
)

var testLog = slog.New(slog.NewTextHandler(os.Stderr, nil))

type mockWordsClient struct {
	normResp *wordspb.WordsReply
	normErr  error
}

func (m *mockWordsClient) Ping(_ context.Context, _ *emptypb.Empty, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (m *mockWordsClient) Norm(_ context.Context, _ *wordspb.WordsRequest, _ ...grpc.CallOption) (*wordspb.WordsReply, error) {
	return m.normResp, m.normErr
}

func newClient(mock wordspb.WordsClient) *words.Client {
	return words.NewClientWithGRPC(mock, testLog)
}

// Norm возвращает нормализованные слова при успешном ответе сервера
func TestNorm_OK(t *testing.T) {
	c := newClient(&mockWordsClient{
		normResp: &wordspb.WordsReply{Words: []string{"linux", "kernel"}},
	})
	result, err := c.Norm(context.Background(), "linux kernel")
	require.NoError(t, err)
	require.Equal(t, []string{"linux", "kernel"}, result)
}

// Norm возвращает пустой срез при пустом ответе сервера
func TestNorm_Empty(t *testing.T) {
	c := newClient(&mockWordsClient{normResp: &wordspb.WordsReply{}})
	result, err := c.Norm(context.Background(), "")
	require.NoError(t, err)
	require.Empty(t, result)
}

// Ошибка gRPC при нормализации пробрасывается наружу
func TestNorm_Error(t *testing.T) {
	c := newClient(&mockWordsClient{normErr: fmt.Errorf("norm failed")})
	_, err := c.Norm(context.Background(), "linux")
	require.Error(t, err)
}

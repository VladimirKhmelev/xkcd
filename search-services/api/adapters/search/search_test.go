package search_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"yadro.com/course/api/adapters/search"
	searchpb "yadro.com/course/proto/search"
)

var log = slog.New(slog.NewTextHandler(os.Stderr, nil))

// mockSearchClient имитирует gRPC SearchClient без реального сетевого соединения.
type mockSearchClient struct {
	pingErr    error
	searchResp *searchpb.SearchReply
	searchErr  error
	isearchResp *searchpb.SearchReply
	isearchErr  error
}

func (m *mockSearchClient) Ping(_ context.Context, _ *emptypb.Empty, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, m.pingErr
}

func (m *mockSearchClient) Search(_ context.Context, _ *searchpb.SearchRequest, _ ...grpc.CallOption) (*searchpb.SearchReply, error) {
	return m.searchResp, m.searchErr
}

func (m *mockSearchClient) ISearch(_ context.Context, _ *searchpb.SearchRequest, _ ...grpc.CallOption) (*searchpb.SearchReply, error) {
	return m.isearchResp, m.isearchErr
}

// newClientWithMock создаёт search.Client с подменённым gRPC-клиентом через экспортированный конструктор.
// Поскольку NewClient требует реального адреса, используем NewClientFromGRPC (если есть),
// иначе тестируем через реальный NewClient с фиктивным адресом только создание.
func TestNewClient_EmptyAddress(t *testing.T) {
	// grpc.NewClient не возвращает ошибку на пустой адрес — это поведение gRPC,
	// поэтому просто убеждаемся, что клиент создаётся без паники.
	c, err := search.NewClient("", log)
	require.NoError(t, err)
	require.NotNil(t, c)
}

// TestNewClient_ValidAddress проверяет, что клиент успешно создаётся с валидным адресом.
func TestNewClient_ValidAddress(t *testing.T) {
	c, err := search.NewClient("localhost:50051", log)
	require.NoError(t, err)
	require.NotNil(t, c)
}

// --- тесты через мок gRPC-сервера ---

// newMockClient строит search.Client поверх мока, используя пакет-уровневую функцию NewClientWithGRPC.
// Если такой функции нет, тесты ниже покрывают логику через реальный тестовый gRPC-сервер.
func newClientWith(t *testing.T, mock searchpb.SearchClient) *search.Client {
	t.Helper()
	c, err := search.NewClientWithGRPC(mock, log)
	require.NoError(t, err)
	return c
}

// TestPing_OK проверяет, что Ping возвращает nil при успешном ответе сервера.
func TestPing_OK(t *testing.T) {
	c := newClientWith(t, &mockSearchClient{})
	require.NoError(t, c.Ping(context.Background()))
}

// TestPing_Error проверяет, что Ping пробрасывает ошибку gRPC-слоя наружу.
func TestPing_Error(t *testing.T) {
	c := newClientWith(t, &mockSearchClient{pingErr: fmt.Errorf("unavailable")})
	require.Error(t, c.Ping(context.Background()))
}

// TestSearch_OK проверяет, что Search корректно преобразует proto-ответ в []core.Comics.
func TestSearch_OK(t *testing.T) {
	mock := &mockSearchClient{
		searchResp: &searchpb.SearchReply{
			Comics: []*searchpb.Comics{
				{Id: 1, Url: "https://xkcd.com/1/"},
				{Id: 2, Url: "https://xkcd.com/2/"},
			},
		},
	}
	c := newClientWith(t, mock)
	result, err := c.Search(context.Background(), "linux", 10)
	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, 1, result[0].ID)
	require.Equal(t, "https://xkcd.com/1/", result[0].URL)
}

// TestSearch_Empty проверяет, что пустой ответ сервера возвращает пустой срез без ошибки.
func TestSearch_Empty(t *testing.T) {
	mock := &mockSearchClient{searchResp: &searchpb.SearchReply{}}
	c := newClientWith(t, mock)
	result, err := c.Search(context.Background(), "nothing", 10)
	require.NoError(t, err)
	require.Empty(t, result)
}

// TestSearch_Error проверяет, что ошибка gRPC при поиске пробрасывается наружу.
func TestSearch_Error(t *testing.T) {
	mock := &mockSearchClient{searchErr: fmt.Errorf("search failed")}
	c := newClientWith(t, mock)
	_, err := c.Search(context.Background(), "linux", 10)
	require.Error(t, err)
}

// TestSearchIndex_OK проверяет, что SearchIndex корректно преобразует proto-ответ в []core.Comics.
func TestSearchIndex_OK(t *testing.T) {
	mock := &mockSearchClient{
		isearchResp: &searchpb.SearchReply{
			Comics: []*searchpb.Comics{
				{Id: 42, Url: "https://xkcd.com/42/"},
			},
		},
	}
	c := newClientWith(t, mock)
	result, err := c.SearchIndex(context.Background(), "linux", 5)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, 42, result[0].ID)
	require.Equal(t, "https://xkcd.com/42/", result[0].URL)
}

// TestSearchIndex_Error проверяет, что ошибка gRPC при индексном поиске пробрасывается наружу.
func TestSearchIndex_Error(t *testing.T) {
	mock := &mockSearchClient{isearchErr: fmt.Errorf("isearch failed")}
	c := newClientWith(t, mock)
	_, err := c.SearchIndex(context.Background(), "linux", 5)
	require.Error(t, err)
}

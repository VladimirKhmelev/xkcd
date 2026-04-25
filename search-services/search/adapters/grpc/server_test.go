package grpc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	searchpb "yadro.com/course/proto/search"
	grpcadapter "yadro.com/course/search/adapters/grpc"
	"yadro.com/course/search/core"
)

type mockSearcher struct {
	searchRes  []core.Comics
	searchErr  error
	isearchRes []core.Comics
	isearchErr error
}

func (m *mockSearcher) Search(_ context.Context, _ string, _ int) ([]core.Comics, error) {
	return m.searchRes, m.searchErr
}
func (m *mockSearcher) ISearch(_ context.Context, _ string, _ int) ([]core.Comics, error) {
	return m.isearchRes, m.isearchErr
}

// Ping всегда возвращает пустой ответ без ошибки
func TestPing(t *testing.T) {
	srv := grpcadapter.NewServer(&mockSearcher{})
	_, err := srv.Ping(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
}

// Search возвращает найденные комиксы с корректным total
func TestSearch_OK(t *testing.T) {
	srv := grpcadapter.NewServer(&mockSearcher{
		searchRes: []core.Comics{{ID: 1, URL: "url1"}},
	})
	resp, err := srv.Search(context.Background(), &searchpb.SearchRequest{Phrase: "linux", Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), resp.Total)
	require.Equal(t, int64(1), resp.Comics[0].Id)
}

// Search с пустым результатом возвращает total=0
func TestSearch_Empty(t *testing.T) {
	srv := grpcadapter.NewServer(&mockSearcher{searchRes: []core.Comics{}})
	resp, err := srv.Search(context.Background(), &searchpb.SearchRequest{Phrase: "nothing"})
	require.NoError(t, err)
	require.Equal(t, int64(0), resp.Total)
}

// Ошибка сервиса при Search возвращается как Internal
func TestSearch_Error(t *testing.T) {
	srv := grpcadapter.NewServer(&mockSearcher{searchErr: errors.New("db down")})
	_, err := srv.Search(context.Background(), &searchpb.SearchRequest{Phrase: "linux"})
	require.Equal(t, codes.Internal, status.Code(err))
}

// ISearch возвращает найденные комиксы с корректным total
func TestISearch_OK(t *testing.T) {
	srv := grpcadapter.NewServer(&mockSearcher{
		isearchRes: []core.Comics{{ID: 2, URL: "url2"}, {ID: 3, URL: "url3"}},
	})
	resp, err := srv.ISearch(context.Background(), &searchpb.SearchRequest{Phrase: "kernel", Limit: 5})
	require.NoError(t, err)
	require.Equal(t, int64(2), resp.Total)
}

// Ошибка сервиса при ISearch возвращается как Internal
func TestISearch_Error(t *testing.T) {
	srv := grpcadapter.NewServer(&mockSearcher{isearchErr: errors.New("index broken")})
	_, err := srv.ISearch(context.Background(), &searchpb.SearchRequest{Phrase: "linux"})
	require.Equal(t, codes.Internal, status.Code(err))
}

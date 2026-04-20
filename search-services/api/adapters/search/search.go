package search

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"yadro.com/course/api/core"
	searchpb "yadro.com/course/proto/search"
)

type Client struct {
	log    *slog.Logger
	client searchpb.SearchClient
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		client: searchpb.NewSearchClient(conn),
		log:    log,
	}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &emptypb.Empty{})
	return err
}

func (c *Client) Search(ctx context.Context, phrase string, limit int) ([]core.Comics, error) {
	resp, err := c.client.Search(ctx, &searchpb.SearchRequest{
		Phrase: phrase,
		Limit:  int64(limit),
	})
	if err != nil {
		return nil, err
	}
	comics := make([]core.Comics, 0, len(resp.Comics))
	for _, c := range resp.Comics {
		comics = append(comics, core.Comics{ID: int(c.Id), URL: c.Url})
	}
	return comics, nil
}

func (c *Client) SearchIndex(ctx context.Context, phrase string, limit int) ([]core.Comics, error) {
	resp, err := c.client.ISearch(ctx, &searchpb.SearchRequest{
		Phrase: phrase,
		Limit:  int64(limit),
	})
	if err != nil {
		return nil, err
	}
	comics := make([]core.Comics, 0, len(resp.Comics))
	for _, c := range resp.Comics {
		comics = append(comics, core.Comics{ID: int(c.Id), URL: c.Url})
	}
	return comics, nil
}

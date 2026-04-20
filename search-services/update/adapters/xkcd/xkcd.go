package xkcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"yadro.com/course/update/core"
)

const lastPath = "/info.0.json"

type Client struct {
	log    *slog.Logger
	client http.Client
	url    string
}

func NewClient(url string, timeout time.Duration, log *slog.Logger) (*Client, error) {
	if url == "" {
		return nil, fmt.Errorf("empty base url specified")
	}
	return &Client{
		client: http.Client{Timeout: timeout},
		log:    log,
		url:    url,
	}, nil
}

type xkcdResponse struct {
	Num        int    `json:"num"`
	Img        string `json:"img"`
	Title      string `json:"title"`
	Alt        string `json:"alt"`
	Transcript string `json:"transcript"`
}

func (c Client) Get(ctx context.Context, id int) (core.XKCDInfo, error) {
	url := fmt.Sprintf("%s/%d/info.0.json", c.url, id)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return core.XKCDInfo{}, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return core.XKCDInfo{}, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Error("failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		return core.XKCDInfo{}, fmt.Errorf("comic not found")
	}

	if resp.StatusCode != http.StatusOK {
		return core.XKCDInfo{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var data xkcdResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return core.XKCDInfo{}, err
	}

	return core.XKCDInfo{
		ID:          data.Num,
		URL:         data.Img,
		Description: data.Title + " " + data.Alt + " " + data.Transcript,
	}, nil
}

func (c Client) LastID(ctx context.Context) (int, error) {
	url := fmt.Sprintf("%s%s", c.url, lastPath)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Error("failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var data xkcdResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}

	return data.Num, nil
}

package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	searchpb "yadro.com/course/proto/search"
	"yadro.com/course/search/core"
)

type Server struct {
	searchpb.UnimplementedSearchServer
	service core.Searcher
}

func NewServer(service core.Searcher) *Server {
	return &Server{service: service}
}

func (s *Server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *Server) Search(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchReply, error) {
	comics, err := s.service.Search(ctx, req.Phrase, int(req.Limit))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	pbComics := make([]*searchpb.Comics, 0, len(comics))
	for _, c := range comics {
		pbComics = append(pbComics, &searchpb.Comics{Id: int64(c.ID), Url: c.URL})
	}
	return &searchpb.SearchReply{
		Comics: pbComics,
		Total:  int64(len(pbComics)),
	}, nil
}

func (s *Server) ISearch(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchReply, error) {
	comics, err := s.service.ISearch(ctx, req.Phrase, int(req.Limit))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	pbComics := make([]*searchpb.Comics, 0, len(comics))
	for _, c := range comics {
		pbComics = append(pbComics, &searchpb.Comics{Id: int64(c.ID), Url: c.URL})
	}
	return &searchpb.SearchReply{
		Comics: pbComics,
		Total:  int64(len(pbComics)),
	}, nil
}

// Package server contains grpc server logic
package server

import (
	"context"
	"sync"

	"github.com/EgMeln/position_service/internal/model"
	"github.com/EgMeln/position_service/internal/service"
	"github.com/EgMeln/position_service/protocol"
	"github.com/google/uuid"
)

// PositionServer struct for grpc server logic
type PositionServer struct {
	mu           *sync.RWMutex
	generatedMap map[string]*model.GeneratedPrice
	posService   service.PositionService
	*protocol.UnimplementedPositionServiceServer
}

// NewPositionServer returns new service instance
func NewPositionServer(serv *service.PositionService, mu *sync.RWMutex, priceMap map[string]*model.GeneratedPrice) *PositionServer {
	return &PositionServer{generatedMap: priceMap, mu: mu, posService: *serv}
}

// OpenPositionAsk method open position record by ask
func (srv *PositionServer) OpenPositionAsk(ctx context.Context, in *protocol.OpenRequest) (*protocol.OpenResponse, error) {
	position := model.Transaction{
		ID:        uuid.New(),
		PriceOpen: (srv.generatedMap)[in.Trans.Symbol].Ask,
		IsBay:     true,
		Symbol:    in.Trans.Symbol,
	}
	id, err := srv.posService.OpenPosition(ctx, &position, "Ask")
	if err != nil {
		return nil, err
	}
	return &protocol.OpenResponse{ID: id.String()}, nil
}

// OpenPositionBid method open position record by bid
func (srv *PositionServer) OpenPositionBid(ctx context.Context, in *protocol.OpenRequest) (*protocol.OpenResponse, error) {
	position := model.Transaction{
		ID:        uuid.New(),
		PriceOpen: (srv.generatedMap)[in.Trans.Symbol].Bid,
		IsBay:     true,
		Symbol:    in.Trans.Symbol,
	}
	id, err := srv.posService.OpenPosition(ctx, &position, "Bid")
	if err != nil {
		return nil, err
	}
	return &protocol.OpenResponse{ID: id.String()}, nil
}

// ClosePositionAsk method close position record by ask
func (srv *PositionServer) ClosePositionAsk(ctx context.Context, in *protocol.CloseRequest) (*protocol.CloseResponse, error) {
	id, err := uuid.Parse(in.ID)
	if err != nil {
		return &protocol.CloseResponse{}, err
	}
	result, err := srv.posService.ClosePosition(ctx, &(srv.generatedMap)[in.Symbol].Ask, &id)
	if err != nil {
		return &protocol.CloseResponse{}, err
	}
	return &protocol.CloseResponse{Result: result}, nil
}

// ClosePositionBid method open position record by bid
func (srv *PositionServer) ClosePositionBid(ctx context.Context, in *protocol.CloseRequest) (*protocol.CloseResponse, error) {
	id, err := uuid.Parse(in.ID)
	if err != nil {
		return &protocol.CloseResponse{}, err
	}
	result, err := srv.posService.ClosePosition(ctx, &(srv.generatedMap)[in.Symbol].Bid, &id)
	if err != nil {
		return &protocol.CloseResponse{}, err
	}
	return &protocol.CloseResponse{Result: result}, nil
}

// Package service contains business logic
package service

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/EgMeln/position_service/internal/model"
	"github.com/EgMeln/position_service/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/sirupsen/logrus"
)

// PositionService struct for
type PositionService struct {
	pool           *pgxpool.Pool
	rep            repository.PriceTransaction
	mu             *sync.RWMutex
	positionMap    map[string]map[string]*chan *model.GeneratedPrice
	transactionMap map[string]*model.Transaction
	ch             chan *model.GeneratedPrice
	BayByAsk       string
	BayByBid       string
}

// NewPositionService used for setting position services
func NewPositionService(ctx context.Context, rep *repository.PostgresPrice, pos map[string]map[string]*chan *model.GeneratedPrice,
	pool *pgxpool.Pool, mute *sync.RWMutex, ch chan *model.GeneratedPrice) *PositionService {
	PosService := PositionService{rep: rep, mu: mute, positionMap: pos, pool: pool, transactionMap: make(map[string]*model.Transaction), ch: ch, BayByAsk: "Ask", BayByBid: "Bid"}
	go PosService.waitForNotification(ctx)
	return &PosService
}

// OpenPosition add record about position
func (src *PositionService) OpenPosition(ctx context.Context, trans *model.Transaction, str string) (*uuid.UUID, error) {
	return src.rep.OpenPosition(ctx, trans, str)
}

// ClosePosition update record about position
func (src *PositionService) ClosePosition(ctx context.Context, closePrice *float64, id *uuid.UUID) (string, error) {
	return src.rep.ClosePosition(ctx, closePrice, id)
}

func (src *PositionService) getProfitByAsk(ch chan *model.GeneratedPrice, trans *model.Transaction) {
	for {
		price, ok := <-ch
		if ok {
			log.Infof("For position %v profit if close: %v", trans.ID, price.Ask-trans.PriceOpen)
		} else {
			log.Infof("Position with id %v close", trans.ID)
			return
		}
	}
}

func (src *PositionService) getProfitByBid(ch chan *model.GeneratedPrice, trans *model.Transaction) {
	for {
		price, ok := <-ch
		if ok {
			log.Infof("For position %v profit if close: %v", trans.ID, price.Bid-trans.PriceOpen)
		} else {
			log.Infof("Position with id %v close", trans.ID)
			return
		}
	}
}
func (src *PositionService) waitForNotification(ctx context.Context) {
	conn, err := src.pool.Acquire(ctx)
	if err != nil {
		log.Errorf("Error connection %v", err)
	}
	defer conn.Release()
	_, err = conn.Exec(ctx, "listen positions")
	if err != nil {
		log.Errorf(" conn exec %v", err)
	}
	for {
		notification, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			log.Errorf("error waiting for notification: %v", err)
		}
		position := model.Transaction{}
		if marshErr := json.Unmarshal([]byte(notification.Payload), &position); marshErr != nil {
			log.Errorf("Unmarshal error %v", err)
		}
		ch := make(chan *model.GeneratedPrice)
		if position.IsBay {
			src.mu.Lock()
			src.positionMap[position.Symbol][position.ID.String()] = &ch
			src.transactionMap[position.ID.String()] = &position
			go src.SystemStop(ctx, src.ch, &position)
			src.mu.Unlock()
			if position.BayBy == src.BayByAsk {
				go src.getProfitByAsk(ch, &position)
			} else if position.BayBy == src.BayByBid {
				go src.getProfitByBid(ch, &position)
			}
		} else {
			src.mu.Lock()
			if src.positionMap[position.Symbol][position.ID.String()] != nil {
				close(*src.positionMap[position.Symbol][position.ID.String()])
				delete(src.positionMap[position.Symbol], position.ID.String())
				delete(src.transactionMap, position.ID.String())
			}
			src.mu.Unlock()
		}
	}
}

// SystemStop stop trade positions if stopLoss/takeProfit/marginCall
func (src *PositionService) SystemStop(ctx context.Context, ch chan *model.GeneratedPrice, transaction *model.Transaction) { //nolint:gocognit //nothing to change
	for {
		select {
		case newPrice := <-ch:
			if newPrice.Symbol == transaction.Symbol {
				src.mu.Lock()
				if src.stopLoss(newPrice, transaction) || src.takeProfit(newPrice, transaction) {
					var stopPrice float64
					if transaction.BayBy == src.BayByAsk {
						stopPrice = newPrice.Ask
					} else if transaction.BayBy == src.BayByBid {
						stopPrice = newPrice.Bid
					}
					profit, err := src.ClosePosition(ctx, &stopPrice, &transaction.ID)
					if err != nil {
						log.Errorf("close positins error")
					}
					log.Info("profit ", profit)
				}

				positionID, priceClose, ifClose := src.marginLiquidation(newPrice)
				if ifClose {
					profit, err := src.ClosePosition(ctx, &priceClose, &positionID)
					if err != nil {
						log.Errorf("close positins error")
					}
					log.Info("profit ", profit)
				}
				src.mu.Unlock()
			}
		case <-ctx.Done():
			return
		}
	}
}

func (src *PositionService) marginLiquidation(pos *model.GeneratedPrice) (uuid.UUID, float64, bool) {
	var balance float64
	var positionID uuid.UUID
	var priceClose float64
	for _, position := range src.transactionMap {
		if position.Symbol == pos.Symbol {
			if position.BayBy == src.BayByAsk {
				balance += position.PriceOpen - pos.Ask
				if (position.PriceOpen - pos.Ask) <= 0 {
					positionID = position.ID
					priceClose = pos.Ask
				}
			} else if position.BayBy == src.BayByBid {
				balance += position.PriceOpen - pos.Bid
				if (position.PriceOpen - pos.Bid) <= 0 {
					positionID = position.ID
					priceClose = pos.Bid
				}
			}
		}
	}
	return positionID, priceClose, balance < 0.0
}
func (src *PositionService) takeProfit(pos *model.GeneratedPrice, trans *model.Transaction) bool {
	if trans.IsBay {
		if trans.BayBy == src.BayByAsk {
			return trans.TakeProfit <= pos.Ask
		} else if trans.BayBy == src.BayByBid {
			return trans.TakeProfit <= pos.Bid
		}
	}
	return false
}
func (src *PositionService) stopLoss(pos *model.GeneratedPrice, trans *model.Transaction) bool {
	if trans.IsBay {
		if trans.BayBy == src.BayByAsk {
			return trans.StopLoss >= pos.Ask
		} else if trans.BayBy == src.BayByBid {
			return trans.StopLoss >= pos.Bid
		}
	}
	return false
}

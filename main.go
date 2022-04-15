package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/EgMeln/position_service/internal/config"
	"github.com/EgMeln/position_service/internal/model"
	"github.com/EgMeln/position_service/internal/repository"
	"github.com/EgMeln/position_service/internal/server"
	"github.com/EgMeln/position_service/internal/service"
	"github.com/EgMeln/position_service/protocol"
	protocolPrice "github.com/EgMeln/price_service/protocol"
	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	initLog()
	cfg, err := config.New()
	if err != nil {
		log.Warnf("Config error %v", err)
	}
	ctx := context.Background()
	cfg.DBURL = fmt.Sprintf("%s://%s:%s@%s:%d/%s", cfg.DB, cfg.User, cfg.Password, cfg.Host, cfg.PortPostgres, cfg.DBNamePostgres)
	log.Infof("DB URL: %s", cfg.DBURL)
	pool := connectPostgres(ctx, cfg.DBURL)
	log.Infof("Connected!")
	mu := new(sync.RWMutex)

	transactionMap := map[string]*model.GeneratedPrice{}
	positionMap := map[string]map[string]*chan *model.GeneratedPrice{
		"Aeroflot": {},
		"ALROSA":   {},
		"Akron":    {},
	}
	ch := make(chan *model.GeneratedPrice)
	connectionPriceServer := connectPriceServer()
	go subscribePrices(ctx, "Aeroflot", connectionPriceServer, mu, transactionMap, positionMap, ch)
	go subscribePrices(ctx, "ALROSA", connectionPriceServer, mu, transactionMap, positionMap, ch)
	go subscribePrices(ctx, "Akron", connectionPriceServer, mu, transactionMap, positionMap, ch)
	transactionService := service.NewPositionService(ctx, &repository.PostgresPrice{PoolPrice: pool}, positionMap, pool, mu, ch)

	transactionServer := server.NewPositionServer(transactionService, mu, transactionMap)

	err = runGRPC(transactionServer)

	if err != nil {
		log.Printf("err in grpc run %v", err)
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	log.Println("received signal", <-c)
	log.Info("END")
}
func initLog() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}
func connectPriceServer() protocolPrice.PriceServiceClient {
	addressGRPC := "localhost:8089"
	con, err := grpc.Dial(addressGRPC, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatal("cannot dial server: ", err)
	}
	log.Info("Success connect grpc server")
	return protocolPrice.NewPriceServiceClient(con)
}

func connectPostgres(ctx context.Context, URL string) *pgxpool.Pool {
	pool, err := pgxpool.Connect(ctx, URL)
	if err != nil {
		log.Warnf("Error connection to DB %v", err)
	}
	return pool
}
func runGRPC(recServer protocol.PositionServiceServer) error {
	port := "localhost:8083"
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	protocol.RegisterPositionServiceServer(grpcServer, recServer)
	log.Infof("server listening at %v", listener.Addr())
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
	return grpcServer.Serve(listener)
}

func subscribePrices(ctx context.Context, symbol string, client protocolPrice.PriceServiceClient, mu *sync.RWMutex, transactionMap map[string]*model.GeneratedPrice,
	positionMap map[string]map[string]*chan *model.GeneratedPrice, ch chan *model.GeneratedPrice) {
	req := protocolPrice.GetRequest{Symbol: symbol}
	for {
		stream, err := client.GetPrice(ctx, &req)
		if err != nil {
			log.Fatalf("%v get price error, %v", client, err)
		}
		in, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatalf("Failed to receive a note : %v", err)
		}
		cur := &model.GeneratedPrice{Symbol: in.Price.Symbol, Ask: float64(in.Price.Ask), Bid: float64(in.Price.Bid), DoteTime: in.Price.Time}
		mu.Lock()
		transactionMap[cur.Symbol] = cur
		for _, v := range positionMap[cur.Symbol] {
			*v <- cur
			ch <- cur
		}
		mu.Unlock()

		log.Infof("Got currency data Name: %v Ask: %v Bid: %v  at time %v",
			in.Price.Symbol, in.Price.Ask, in.Price.Bid, in.Price.Time)
	}
}

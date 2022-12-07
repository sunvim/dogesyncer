package rpc

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/hashicorp/go-hclog"
	"github.com/sunvim/dogesyncer/blockchain"
)

type RpcServer struct {
	logger     hclog.Logger
	ctx        context.Context
	blockchain *blockchain.Blockchain
	addr       string
	port       string
}

func NewRpcServer(logger hclog.Logger,
	blockchain *blockchain.Blockchain,
	addr, port string) *RpcServer {
	return &RpcServer{
		logger:     logger.Named("rpc"),
		addr:       addr,
		port:       port,
		blockchain: blockchain,
	}
}

func (s *RpcServer) Start(ctx context.Context) error {
	go func() {
		svc := fiber.New(fiber.Config{
			Prefork:               false,
			ServerHeader:          "doge syncer team",
			DisableStartupMessage: true,
		})

		ap := fmt.Sprintf("%s:%s", s.addr, s.port)

		s.logger.Info("boot", "address", s.addr, "port", s.port)

		svc.Listen(ap)
	}()

	return nil
}

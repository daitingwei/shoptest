package data

import (
	"context"

	"bff/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	kratosgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"google.golang.org/grpc"
)

var ProviderSet = wire.NewSet(NewData, NewBFFRepo)

type Data struct {
	log       *log.Helper
	pcConn    *grpc.ClientConn
	orderConn *grpc.ClientConn
}

func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	pcConn, err := kratosgrpc.DialInsecure(
		context.Background(),
		kratosgrpc.WithEndpoint("127.0.0.1:9003"),
	)
	if err != nil {
		return nil, nil, err
	}

	orderConn, err := kratosgrpc.DialInsecure(
		context.Background(),
		kratosgrpc.WithEndpoint("127.0.0.1:9004"),
	)
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
		pcConn.Close()
		orderConn.Close()
	}
	return &Data{log: log.NewHelper(logger), pcConn: pcConn, orderConn: orderConn}, cleanup, nil
}

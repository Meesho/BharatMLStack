package grpc

import (
	"context"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

func GetConnection(host string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error().Msg("Unable to create client connection GRPC")
		return nil, err
	}
	return conn, nil
}

func SendGRPCRequest(ctx context.Context, conn *grpc.ClientConn, method string, req proto.Message, resp proto.Message, md metadata.MD, opts ...grpc.CallOption) error {
	if md != nil {
		ctx = metadata.NewOutgoingContext(ctx, md)
	}
	return conn.Invoke(ctx, method, req, resp, opts...)
}

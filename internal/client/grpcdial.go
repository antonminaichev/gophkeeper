package client

import (
	"context"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// clientDial creates grpc connection.
func (cli *CLI) clientDial(ctx context.Context) (*grpc.ClientConn, error) {
	if !cli.TLSEnable {
		return grpc.DialContext(ctx, cli.ServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	tcfg, err := NewClientTLSConfig(ClientTLSParams{
		CAPath:             cli.TLSCAPath,
		ServerName:         cli.TLSServerName,
		InsecureSkipVerify: cli.TLSInsecureSkipVerify,
		MinVersionTLS12:    true,
	})
	if err != nil {
		return nil, err
	}
	return grpc.DialContext(ctx, cli.ServerAddr, grpc.WithTransportCredentials(credentials.NewTLS(tcfg)))
}

// dialVaultClient makes client vault.
func (cli *CLI) dialVaultClient(ctx context.Context) (pb.VaultServiceClient, *grpc.ClientConn, error) {
	conn, err := cli.clientDial(ctx)
	if err != nil {
		return nil, nil, err
	}
	return pb.NewVaultServiceClient(conn), conn, nil
}

// dialAuthClient makes client auth.
func (cli *CLI) dialAuthClient(ctx context.Context) (pb.AuthServiceClient, *grpc.ClientConn, error) {
	conn, err := cli.clientDial(ctx)
	if err != nil {
		return nil, nil, err
	}
	return pb.NewAuthServiceClient(conn), conn, nil
}

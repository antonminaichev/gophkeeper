package client

import (
	"context"
	"errors"
	"fmt"
	"strings"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
	"github.com/antonminaichev/gophkeeper/internal/auth"
	"google.golang.org/grpc/metadata"
)

// attachAuth добавляет Bearer-заголовок из флага или сессии.
func (cli *CLI) attachAuth(ctx context.Context) (context.Context, *Session, error) {
	if t := strings.TrimSpace(cli.AccessToken); t != "" {
		return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+t), nil, nil
	}
	return cli.EnsureAuth(ctx)
}

// EnsureAuth загружает сессию, обновляет при необходимости и возвращает ctx с заголовком.
func (cli *CLI) EnsureAuth(ctx context.Context) (context.Context, *Session, error) {
	s, err := LoadSession()
	if err != nil {
		return ctx, nil, fmt.Errorf("not logged in: %w", err)
	}
	if s.AccessValid() {
		return s.WithAuth(ctx), s, nil
	}
	if s.RefreshToken == "" {
		return ctx, nil, errors.New("no refresh token; please login")
	}
	newS, err := cli.refreshSession(ctx, s.RefreshToken)
	if err != nil {
		return ctx, nil, fmt.Errorf("refresh failed: %w", err)
	}
	newS.SyncCursor = s.SyncCursor
	if err := SaveSession(newS); err != nil {
		return ctx, nil, fmt.Errorf("save session: %w", err)
	}
	return newS.WithAuth(ctx), newS, nil
}

// refreshSession вызывает AuthService.Refresh и собирает новую Session.
func (cli *CLI) refreshSession(ctx context.Context, refresh string) (*Session, error) {
	conn, err := cli.clientDial(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	c := pb.NewAuthServiceClient(conn)
	resp, err := c.Refresh(ctx, &pb.RefreshRequest{RefreshToken: refresh})
	if err != nil {
		return nil, err
	}
	exp, err := auth.Expiration(resp.GetAccessToken())
	if err != nil {
		return nil, err
	}
	return &Session{
		AccessToken:  resp.GetAccessToken(),
		RefreshToken: resp.GetRefreshToken(),
		AccessExp:    exp,
	}, nil
}

package client

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/term"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// readLine reads a line from stdin.
func readLine(prompt string) (string, error) {
	fmt.Print(prompt)
	r := bufio.NewReader(os.Stdin)
	s, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(s), nil
}

// readPassword reads a password from stdin without echoing.
func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// validateEmail performs basic email validation.
func validateEmail(email string) bool {
	email = strings.TrimSpace(email)
	return email != "" &&
		strings.Count(email, "@") == 1 &&
		!strings.HasPrefix(email, "@") &&
		!strings.HasSuffix(email, "@")
}

// LooksLikeUUID checks if string looks like a UUID.
func LooksLikeUUID(s string) bool {
	s = strings.TrimSpace(s)
	return len(s) == 36 && strings.Count(s, "-") == 4
}

// sliceMetaToMap converts protobuf meta entries to map.
func sliceMetaToMap(meta []*pb.ItemMetaEntry) map[string]string {
	out := make(map[string]string, len(meta))
	for _, kv := range meta {
		k := strings.TrimSpace(kv.GetKey())
		if k == "" {
			continue
		}
		out[k] = kv.GetValue()
	}
	return out
}

// metaTitle extracts title from meta entries.
func metaTitle(meta []*pb.ItemMetaEntry) string {
	for _, kv := range meta {
		if kv.Key == "title" {
			return kv.Value
		}
	}
	return ""
}

// buildItemRef builds ItemRef from selector string.
func (cli *CLI) buildItemRef(selector string) *pb.ItemRef {
	selector = strings.TrimSpace(selector)
	if cache, err := LoadCache(); err == nil && cache != nil {
		if id, ok := cache.LookupID(selector); ok && LooksLikeUUID(id) {
			return &pb.ItemRef{Ref: &pb.ItemRef_Id{Id: id}}
		}
	}
	switch {
	case LooksLikeUUID(selector):
		return &pb.ItemRef{Ref: &pb.ItemRef_Id{Id: selector}}
	case strings.HasPrefix(selector, "@"):
		return &pb.ItemRef{Ref: &pb.ItemRef_Alias{Alias: strings.TrimPrefix(selector, "@")}}
	default:
		if n, err := strconv.ParseInt(selector, 10, 64); err == nil && n > 0 {
			return &pb.ItemRef{Ref: &pb.ItemRef_HumanId{HumanId: n}}
		}
	}
	return &pb.ItemRef{Ref: &pb.ItemRef_Id{Id: selector}}
}

// dialVaultClient creates vault service client.
func (cli *CLI) dialVaultClient(ctx context.Context) (pb.VaultServiceClient, *grpc.ClientConn, error) {
	conn, err := cli.clientDial(ctx)
	if err != nil {
		return nil, nil, err
	}
	return pb.NewVaultServiceClient(conn), conn, nil
}

// dialAuthClient creates auth service client.
func (cli *CLI) dialAuthClient(ctx context.Context) (pb.AuthServiceClient, *grpc.ClientConn, error) {
	conn, err := cli.clientDial(ctx)
	if err != nil {
		return nil, nil, err
	}
	return pb.NewAuthServiceClient(conn), conn, nil
}

// clientDial creates gRPC connection with TLS if enabled.
func (cli *CLI) clientDial(ctx context.Context) (*grpc.ClientConn, error) {
	if !cli.TLSEnable {
		return grpc.DialContext(ctx, cli.ServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	tcfg, err := cli.makeClientTLSConfig()
	if err != nil {
		return nil, err
	}
	return grpc.DialContext(ctx, cli.ServerAddr, grpc.WithTransportCredentials(credentials.NewTLS(tcfg)))
}

// makeClientTLSConfig builds TLS config from flags.
func (cli *CLI) makeClientTLSConfig() (*tls.Config, error) {
	cfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: cli.TLSInsecureSkipVerify, // for local tests only
	}
	if cli.TLSServerName != "" {
		cfg.ServerName = cli.TLSServerName
	}
	if strings.TrimSpace(cli.TLSCAPath) != "" {
		data, err := os.ReadFile(cli.TLSCAPath)
		if err != nil {
			return nil, fmt.Errorf("read CA file: %w", err)
		}
		cp := x509.NewCertPool()
		if !cp.AppendCertsFromPEM(data) {
			return nil, fmt.Errorf("parse CA file: no certs found")
		}
		cfg.RootCAs = cp
	}
	return cfg, nil
}

// attachAuth attaches auth from token flag or session.
func (cli *CLI) attachAuth(ctx context.Context) (context.Context, *Session, error) {
	if t := strings.TrimSpace(cli.AccessToken); t != "" {
		return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+t), nil, nil
	}
	return cli.EnsureAuth(ctx)
}

// EnsureAuth loads session, refreshes if needed, and returns ctx with header.
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

// refreshSession calls AuthService.Refresh and builds a new Session.
func (cli *CLI) refreshSession(ctx context.Context, refresh string) (*Session, error) {
	// Use a short-lived connection with the same TLS/transport settings.
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
	exp, err := parseExp(resp.GetAccessToken())
	if err != nil {
		return nil, err
	}
	return &Session{
		AccessToken:  resp.GetAccessToken(),
		RefreshToken: resp.GetRefreshToken(),
		AccessExp:    exp,
	}, nil
}

// parseExp extracts "exp" claim as time.Time from a JWT.
func parseExp(token string) (time.Time, error) {
	parser := jwt.Parser{}
	claims := jwt.MapClaims{}
	_, _, err := parser.ParseUnverified(token, claims)
	if err != nil {
		return time.Time{}, err
	}
	switch v := claims["exp"].(type) {
	case float64:
		return time.Unix(int64(v), 0), nil
	case json.Number:
		n, _ := v.Int64()
		return time.Unix(n, 0), nil
	default:
		return time.Time{}, errors.New("exp not found")
	}
}

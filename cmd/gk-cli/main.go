package main

// GophKeeper CLI — small interactive client for auth and vault operations.

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Set via -ldflags at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Global flags.
var (
	serverAddr  string
	accessToken string // optional: overrides session if provided

	// TLS flags
	tlsEnable             bool
	tlsCAPath             string
	tlsServerName         string
	tlsInsecureSkipVerify bool
)

const rpcTimeout = 10 * time.Second

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "gk",
	Short: "GophKeeper CLI",
}

func init() {
	// Connection & auth flags
	rootCmd.PersistentFlags().StringVar(&serverAddr, "server", "localhost:8090", "gRPC server address")
	rootCmd.PersistentFlags().StringVar(&accessToken, "token", os.Getenv("GK_ACCESS_TOKEN"), "Access JWT (optional, overrides saved session)")

	// TLS flags
	rootCmd.PersistentFlags().BoolVar(&tlsEnable, "tls", false, "Use TLS for gRPC transport")
	rootCmd.PersistentFlags().StringVar(&tlsCAPath, "tls-ca", "", "Path to custom CA bundle (PEM)")
	rootCmd.PersistentFlags().StringVar(&tlsServerName, "tls-server-name", "", "Override TLS server name (SNI/verify)")
	rootCmd.PersistentFlags().BoolVar(&tlsInsecureSkipVerify, "tls-insecure-skip-verify", false, "Skip certificate verification (NOT recommended)")

	// Commands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)

	itemCmd.AddCommand(itemAddTextCmd)
	itemCmd.AddCommand(itemListCmd)
	itemCmd.AddCommand(itemGetCmd)
	rootCmd.AddCommand(itemCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("gk %s (commit %s, built %s)\n", version, commit, date)
	},
}

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new user (interactive)",
	RunE: func(cmd *cobra.Command, args []string) error {
		email, err := readLine("Email: ")
		if err != nil {
			return err
		}
		if !validateEmail(email) {
			return fmt.Errorf("invalid email")
		}
		pass1, err := readPassword("Password: ")
		if err != nil {
			return err
		}
		if len(pass1) < 8 {
			return fmt.Errorf("password must be at least 8 characters")
		}
		pass2, err := readPassword("Repeat password: ")
		if err != nil {
			return err
		}
		if pass1 != pass2 {
			return fmt.Errorf("passwords do not match")
		}

		ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
		defer cancel()
		c, conn, err := dialAuthClient(ctx)
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := c.Register(ctx, &pb.RegisterRequest{
			Email:    strings.TrimSpace(email),
			Password: pass1,
		})
		if err != nil {
			return err
		}
		fmt.Println("Registered user:", resp.UserId)
		return nil
	},
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login (interactive) and save session",
	RunE: func(cmd *cobra.Command, args []string) error {
		email, err := readLine("Email: ")
		if err != nil {
			return err
		}
		if !validateEmail(email) {
			return fmt.Errorf("invalid email")
		}
		pass, err := readPassword("Password: ")
		if err != nil {
			return err
		}
		if pass == "" {
			return fmt.Errorf("password cannot be empty")
		}

		ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
		defer cancel()
		c, conn, err := dialAuthClient(ctx)
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := c.Login(ctx, &pb.LoginRequest{
			Email:    strings.TrimSpace(email),
			Password: pass,
		})
		if err != nil {
			return err
		}

		if err := SaveLoginTokens(resp.AccessToken, resp.RefreshToken); err != nil {
			return fmt.Errorf("save session: %w", err)
		}
		fmt.Println("Logged in. Session saved.")
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear saved session",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ClearSession(); err != nil {
			return err
		}
		fmt.Println("Session cleared.")
		return nil
	},
}

var itemCmd = &cobra.Command{
	Use:   "item",
	Short: "Manage vault items",
}

var itemAddTextCmd = &cobra.Command{
	Use:   "add text",
	Short: "Add TEXT item (interactive)",
	RunE: func(cmd *cobra.Command, args []string) error {
		title, err := readLine("Title (meta.title): ")
		if err != nil {
			return err
		}
		content, err := readLine("Content: ")
		if err != nil {
			return err
		}
		alias, _ := readLine("Alias (optional, like @github): ")
		alias = strings.TrimSpace(strings.TrimPrefix(alias, "@"))

		baseCtx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
		defer cancel()
		ctx, _, err := attachAuth(baseCtx)
		if err != nil {
			return err
		}

		c, conn, err := dialVaultClient(ctx)
		if err != nil {
			return err
		}
		defer conn.Close()

		meta := []*pb.ItemMetaEntry{{Key: "title", Value: strings.TrimSpace(title)}}
		resp, err := c.CreateItem(ctx, &pb.CreateItemRequest{
			Type:    pb.ItemType_TEXT,
			Payload: []byte(content),
			Meta:    meta,
			Alias:   alias,
		})
		if err != nil {
			return err
		}
		fmt.Println("Created item:", resp.Id)
		return nil
	},
}

var itemListCmd = &cobra.Command{
	Use:   "list",
	Short: "List items (meta only)",
	RunE: func(cmd *cobra.Command, args []string) error {
		baseCtx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
		defer cancel()
		ctx, _, err := attachAuth(baseCtx)
		if err != nil {
			return err
		}

		c, conn, err := dialVaultClient(ctx)
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := c.ListItems(ctx, &pb.ListRequest{Limit: 100})
		if err != nil {
			return err
		}
		if len(resp.Items) == 0 {
			fmt.Println("No items.")
			return nil
		}

		fmt.Printf("%-22s  %-6s  %-4s  %s\n", "ID", "TYPE", "VER", "TITLE")
		fmt.Printf("%-22s  %-6s  %-4s  %s\n",
			strings.Repeat("-", 22),
			strings.Repeat("-", 6),
			strings.Repeat("-", 4),
			strings.Repeat("-", 30),
		)

		for _, it := range resp.Items {
			title := metaTitle(it.Meta)
			tag := humanTag(it)
			fmt.Printf("%-22s  %-6s  %-4s  %s\n",
				tag,
				it.Type.String(),
				fmt.Sprintf("v%d", it.Version),
				title,
			)
		}
		return nil
	},
}

var itemGetCmd = &cobra.Command{
	Use:   "get <uuid|human_id|@alias>",
	Short: "Get full item by UUID, human id (#) or alias",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		selector := strings.TrimSpace(args[0])
		req := &pb.ItemRef{}

		switch {
		case looksLikeUUID(selector):
			req.Ref = &pb.ItemRef_Id{Id: selector}
		case strings.HasPrefix(selector, "@"):
			req.Ref = &pb.ItemRef_Alias{Alias: strings.TrimPrefix(selector, "@")}
		default:
			if n, err := strconv.ParseInt(selector, 10, 64); err == nil && n > 0 {
				req.Ref = &pb.ItemRef_HumanId{HumanId: n}
			} else {
				return fmt.Errorf("bad selector: use UUID, @alias or numeric human id")
			}
		}

		baseCtx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
		defer cancel()
		ctx, _, err := attachAuth(baseCtx)
		if err != nil {
			return err
		}

		c, conn, err := dialVaultClient(ctx)
		if err != nil {
			return err
		}
		defer conn.Close()

		it, err := c.GetItemByRef(ctx, req)
		if err != nil {
			return err
		}

		fmt.Println("ID      :", it.Id)
		if it.HumanId > 0 {
			fmt.Println("HumanID :", it.HumanId)
		}
		if it.Alias != "" {
			fmt.Println("Alias   :", "@"+it.Alias)
		}
		fmt.Println("Type    :", it.Type.String())
		fmt.Println("Version :", it.Version)
		if t := metaTitle(it.Meta); t != "" {
			fmt.Println("Title   :", t)
		}

		fmt.Println("Payload :")
		_, _ = os.Stdout.Write(it.Payload)
		fmt.Println()
		return nil
	},
}

// Helpers: IO, validation, auth attachment, dialing.

func readLine(prompt string) (string, error) {
	fmt.Print(prompt)
	r := bufio.NewReader(os.Stdin)
	s, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(s), nil
}

func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func validateEmail(email string) bool {
	email = strings.TrimSpace(email)
	return email != "" &&
		strings.Count(email, "@") == 1 &&
		!strings.HasPrefix(email, "@") &&
		!strings.HasSuffix(email, "@")
}

func looksLikeUUID(s string) bool {
	s = strings.TrimSpace(s)
	return len(s) == 36 && strings.Count(s, "-") == 4
}

// attachAuth chooses the best available auth method.
// Priority: --token/env > saved session (with auto-refresh).
func attachAuth(ctx context.Context) (context.Context, *Session, error) {
	if t := strings.TrimSpace(accessToken); t != "" {
		return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+t), nil, nil
	}
	// NOTE: refresh in session.go пока использует свои настройки. Позже можно
	// прокинуть TLS-параметры и туда, чтобы refresh тоже ходил по TLS.
	return EnsureAuth(ctx, serverAddr)
}

func dialVaultClient(ctx context.Context) (pb.VaultServiceClient, *grpc.ClientConn, error) {
	conn, err := clientDial(ctx)
	if err != nil {
		return nil, nil, err
	}
	return pb.NewVaultServiceClient(conn), conn, nil
}

func dialAuthClient(ctx context.Context) (pb.AuthServiceClient, *grpc.ClientConn, error) {
	conn, err := clientDial(ctx)
	if err != nil {
		return nil, nil, err
	}
	return pb.NewAuthServiceClient(conn), conn, nil
}

// clientDial builds a client connection honoring TLS flags.
func clientDial(ctx context.Context) (*grpc.ClientConn, error) {
	if !tlsEnable {
		return grpc.DialContext(ctx, serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	tcfg, err := makeClientTLSConfig()
	if err != nil {
		return nil, err
	}
	return grpc.DialContext(ctx, serverAddr, grpc.WithTransportCredentials(credentials.NewTLS(tcfg)))
}

// makeClientTLSConfig assembles tls.Config from CLI flags.
func makeClientTLSConfig() (*tls.Config, error) {
	cfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: tlsInsecureSkipVerify, // NOT recommended; for quick local tests only
	}
	if tlsServerName != "" {
		cfg.ServerName = tlsServerName
	}
	if strings.TrimSpace(tlsCAPath) != "" {
		data, err := os.ReadFile(tlsCAPath)
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

func metaTitle(meta []*pb.ItemMetaEntry) string {
	for _, kv := range meta {
		if kv.Key == "title" {
			return kv.Value
		}
	}
	return ""
}

// Builds printable tag: prefer "#<human_id>" and "(@alias)"; fallback to UUID.
func humanTag(it *pb.Item) string {
	var parts []string
	if it.HumanId > 0 {
		parts = append(parts, fmt.Sprintf("#%d", it.HumanId))
	}
	if it.Alias != "" {
		parts = append(parts, "(@"+it.Alias+")")
	}
	if len(parts) == 0 {
		return it.Id
	}
	return strings.Join(parts, " ")
}

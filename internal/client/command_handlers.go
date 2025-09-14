package client

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
)

const rpcTimeout = 10 * time.Second

// register implements the register command.
func (cli *CLI) register() error {
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
	c, conn, err := cli.dialAuthClient(ctx)
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
}

// login implements the login command.
func (cli *CLI) login() error {
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
	c, conn, err := cli.dialAuthClient(ctx)
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
}

// logout implements the logout command.
func (cli *CLI) logout() error {
	var hadErr bool
	if err := ClearSession(); err != nil {
		hadErr = true
		fmt.Println("Failed to clear session:", err)
	}
	if err := ClearCache(); err != nil {
		hadErr = true
		fmt.Println("Failed to clear cache:", err)
	}
	if hadErr {
		return fmt.Errorf("logout completed with errors")
	}
	fmt.Println("Session and cache cleared.")
	return nil
}

// sync implements the sync command.
func (cli *CLI) sync() error {
	baseCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ctxAuth, sess, err := cli.EnsureAuth(baseCtx)
	if err != nil {
		return err
	}
	cache, err := LoadCache()
	if err != nil {
		return err
	}

	c, conn, err := cli.dialVaultClient(ctxAuth)
	if err != nil {
		return err
	}
	defer conn.Close()

	cursor := strings.TrimSpace(sess.SyncCursor)
	total := 0
	const page = 200

	for {
		resp, err := c.ListItems(ctxAuth, &pb.ListRequest{Limit: page, Cursor: cursor})
		if err != nil {
			return err
		}
		if len(resp.Items) == 0 {
			if total == 0 {
				fmt.Println("No changes.")
			}
			break
		}

		cache.ApplyChanges(resp.Items)
		for _, it := range resp.Items {
			status := "UPD"
			if it.Deleted {
				status = "DEL"
			}
			fmt.Printf("[%s] %s\n", status, (&CachedItem{
				ID:            it.Id,
				HumanID:       it.HumanId,
				Alias:         it.Alias,
				Type:          it.Type.String(),
				Version:       it.Version,
				UpdatedAtUnix: it.UpdatedAtUnix,
				Deleted:       it.Deleted,
				Meta:          sliceMetaToMap(it.Meta),
			}).DebugString())
		}
		total += len(resp.Items)
		if next := strings.TrimSpace(resp.GetNextCursor()); next != "" && next != cursor {
			cursor = next
		} else {
			break
		}
	}

	if err := SaveCache(cache); err != nil {
		return fmt.Errorf("save cache: %w", err)
	}
	sess.SyncCursor = cursor
	if err := SaveSession(sess); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	if total > 0 {
		fmt.Printf("Applied %d change(s). Cursor saved.\n", total)
	}
	return nil
}

// itemList implements the item list command.
func (cli *CLI) itemList() error {
	cache, err := LoadCache()
	if err != nil {
		return err
	}
	rows := make([]*CachedItem, 0, len(cache.Items))
	for _, it := range cache.Items {
		if !it.Deleted {
			rows = append(rows, it)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].UpdatedAtUnix == rows[j].UpdatedAtUnix {
			return rows[i].ID > rows[j].ID
		}
		return rows[i].UpdatedAtUnix > rows[j].UpdatedAtUnix
	})
	if len(rows) == 0 {
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
	for _, ci := range rows {
		fmt.Println(ci.DebugString())
	}
	return nil
}

// itemGet implements the item get command.
func (cli *CLI) itemGet(selector string) error {
	req := cli.buildItemRef(selector)

	baseCtx, cancel := context.WithTimeout(context.Background(), 2*rpcTimeout)
	defer cancel()
	ctx, _, err := cli.attachAuth(baseCtx)
	if err != nil {
		return err
	}
	c, conn, err := cli.dialVaultClient(ctx)
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

	switch it.Type {
	case pb.ItemType_TEXT:
		fmt.Println("Payload :")
		_, _ = os.Stdout.Write(it.Payload)
		fmt.Println()
	case pb.ItemType_LOGIN:
		lp, err := DecodeLogin(it.Payload)
		if err != nil {
			fmt.Println("Payload :", "(invalid LOGIN payload)", err)
			break
		}
		pass := MaskPassword(lp.Password)
		// TODO: Add reveal flag support
		fmt.Println("Payload : LOGIN")
		if lp.URL != "" {
			fmt.Println("  URL     :", lp.URL)
		}
		fmt.Println("  Username:", lp.Username)
		fmt.Println("  Password:", pass)
		if lp.Note != "" {
			fmt.Println("  Note    :", lp.Note)
		}
	case pb.ItemType_CARD:
		cp, err := DecodeCard(it.Payload)
		if err != nil {
			fmt.Println("Payload :", "(invalid CARD payload)", err)
			break
		}
		cvv := MaskCVV(cp.CVV)
		pan := MaskPAN(cp.PAN)
		// TODO: Add reveal flag support
		fmt.Println("Payload : CARD")
		fmt.Println("  Holder :", cp.Holder)
		fmt.Println("  PAN    :", pan)
		fmt.Printf("  Exp    : %02d/%d\n", cp.ExpMonth, cp.ExpYear)
		fmt.Println("  CVC    :", cvv)
		if cp.Note != "" {
			fmt.Println("  Note   :", cp.Note)
		}
	case pb.ItemType_BINARY:
		filename := ""
		mime := ""
		for _, kv := range it.Meta {
			switch strings.ToLower(kv.Key) {
			case "filename":
				filename = kv.Value
			case "mime":
				mime = kv.Value
			}
		}
		fmt.Println("Payload : BINARY")
		if filename != "" {
			fmt.Println("  Name   :", filename)
		}
		if mime != "" {
			fmt.Println("  MIME   :", mime)
		}
		fmt.Println("  Size   :", HumanSize(int64(len(it.Payload))))
		// TODO: Add save path support
	default:
		fmt.Printf("Payload : %d bytes\n", len(it.Payload))
	}

	if cache, _ := LoadCache(); cache != nil {
		cache.UpsertFromItem(it)
		_ = SaveCache(cache)
	}
	return nil
}

// itemUpdate implements the item update command.
func (cli *CLI) itemUpdate(selector string) error {
	baseCtx, cancel := context.WithTimeout(context.Background(), 2*rpcTimeout)
	defer cancel()
	ctx, _, err := cli.attachAuth(baseCtx)
	if err != nil {
		return err
	}
	c, conn, err := cli.dialVaultClient(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	ref := cli.buildItemRef(selector)
	cur, err := c.GetItemByRef(ctx, ref)
	if err != nil {
		return err
	}

	fmt.Printf("Current type: %s  version: %d\n", cur.Type.String(), cur.Version)
	oldTitle := metaTitle(cur.Meta)
	newTitle, _ := readLine(fmt.Sprintf("New title (empty=keep) [%s]: ", oldTitle))
	newContent, _ := readLine("New content (empty=keep): ")

	var meta []*pb.ItemMetaEntry
	if strings.TrimSpace(newTitle) != "" {
		meta = []*pb.ItemMetaEntry{{Key: "title", Value: strings.TrimSpace(newTitle)}}
	}
	var payload []byte
	if strings.TrimSpace(newContent) != "" && cur.Type == pb.ItemType_TEXT {
		payload = []byte(newContent)
	}

	resp, err := c.UpdateItem(ctx, &pb.UpdateItemRequest{
		Id:              cur.Id,
		ExpectedVersion: cur.Version,
		Payload:         payload,
		Meta:            meta,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Updated: %s v%d -> v%d\n", resp.Id, cur.Version, resp.Version)

	if cache, _ := LoadCache(); cache != nil {
		cache.UpsertFromItem(resp)
		_ = SaveCache(cache)
	}
	return nil
}

// itemDelete implements the item delete command.
func (cli *CLI) itemDelete(selector string) error {
	baseCtx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()
	ctx, _, err := cli.attachAuth(baseCtx)
	if err != nil {
		return err
	}
	c, conn, err := cli.dialVaultClient(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	ref := cli.buildItemRef(selector)
	it, err := c.GetItemByRef(ctx, ref)
	if err != nil {
		return err
	}
	if _, err := c.DeleteItem(ctx, &pb.ItemID{Id: it.Id}); err != nil {
		return err
	}
	fmt.Println("Deleted:", it.Id)

	if cache, _ := LoadCache(); cache != nil {
		cache.ApplyChanges([]*pb.Item{{Id: it.Id, Deleted: true}})
		_ = SaveCache(cache)
	}
	return nil
}

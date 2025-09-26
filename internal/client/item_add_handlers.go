package client

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
)

// itemAddText implements the item add text command.
func (cli *CLI) itemAddText() error {
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
	ctx, _, err := cli.attachAuth(baseCtx)
	if err != nil {
		return err
	}
	c, conn, err := cli.dialVaultClient(ctx)
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
	if it, err := c.GetItemByRef(ctx, &pb.ItemRef{Ref: &pb.ItemRef_Id{Id: resp.Id}}); err == nil {
		if cache, _ := LoadCache(); cache != nil {
			cache.UpsertFromItem(it)
			_ = SaveCache(cache)
		}
	}
	fmt.Println("Created item:", resp.Id)
	return nil
}

// itemAddLogin implements the item add login command.
func (cli *CLI) itemAddLogin() error {
	title, _ := readLine("Title (meta.title): ")
	alias, _ := readLine("Alias (optional, like @site): ")
	alias = strings.TrimSpace(strings.TrimPrefix(alias, "@"))
	site, _ := readLine("Site/URL (optional): ")
	username, _ := readLine("Username: ")
	password, _ := readPassword("Password: ")
	note, _ := readLine("Note (optional): ")

	payload, err := EncodeLogin(LoginPayload{
		Username: strings.TrimSpace(username),
		Password: password,
		URL:      strings.TrimSpace(site),
		Note:     strings.TrimSpace(note),
	})
	if err != nil {
		return err
	}

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

	meta := []*pb.ItemMetaEntry{{Key: "title", Value: strings.TrimSpace(title)}}
	resp, err := c.CreateItem(ctx, &pb.CreateItemRequest{
		Type:    pb.ItemType_LOGIN,
		Payload: payload,
		Meta:    meta,
		Alias:   alias,
	})
	if err != nil {
		return err
	}
	if it, err := c.GetItemByRef(ctx, &pb.ItemRef{Ref: &pb.ItemRef_Id{Id: resp.Id}}); err == nil {
		if cache, _ := LoadCache(); cache != nil {
			cache.UpsertFromItem(it)
			_ = SaveCache(cache)
		}
	}
	fmt.Println("Created LOGIN item:", resp.Id)
	return nil
}

// itemAddCard implements the item add card command.
func (cli *CLI) itemAddCard() error {
	title, _ := readLine("Title (meta.title): ")
	alias, _ := readLine("Alias (optional, like @visa): ")
	alias = strings.TrimSpace(strings.TrimPrefix(alias, "@"))
	holder, _ := readLine("Card holder (e.g., JOHN DOE): ")
	pan, _ := readLine("PAN (digits only; spaces allowed): ")
	expStr, _ := readLine("Expiration (MM/YY or MM/YYYY): ")
	cvc, _ := readLine("CVC (3-4 digits): ")
	note, _ := readLine("Note (optional): ")

	expStr = strings.TrimSpace(expStr)
	var mm, yy int
	if parts := strings.Split(expStr, "/"); len(parts) == 2 {
		mm, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
		yy, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
		if yy < 100 {
			yy += 2000
		}
	}

	payload, err := EncodeCard(CardPayload{
		Holder:   strings.TrimSpace(holder),
		PAN:      pan,
		ExpMonth: mm,
		ExpYear:  yy,
		CVV:      cvc,
		Note:     strings.TrimSpace(note),
	})
	if err != nil {
		return err
	}

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

	meta := []*pb.ItemMetaEntry{{Key: "title", Value: strings.TrimSpace(title)}}
	resp, err := c.CreateItem(ctx, &pb.CreateItemRequest{
		Type:    pb.ItemType_CARD,
		Payload: payload,
		Meta:    meta,
		Alias:   alias,
	})
	if err != nil {
		return err
	}
	if it, err := c.GetItemByRef(ctx, &pb.ItemRef{Ref: &pb.ItemRef_Id{Id: resp.Id}}); err == nil {
		if cache, _ := LoadCache(); cache != nil {
			cache.UpsertFromItem(it)
			_ = SaveCache(cache)
		}
	}
	fmt.Println("Created CARD item:", resp.Id)
	return nil
}

// itemAddFile implements the item add file command.
func (cli *CLI) itemAddFile() error {
	// This would need to be implemented with flag parsing
	// For now, just return an error
	return fmt.Errorf("file upload not implemented in this refactored version")
}

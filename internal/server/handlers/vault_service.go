package handlers

// Vault gRPC service: CRUD for user items, with sync cursor support.

import (
	"context"
	"strings"
	"time"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
	gauth "github.com/antonminaichev/gophkeeper/internal/auth"
	"github.com/antonminaichev/gophkeeper/internal/server/storage"
	"github.com/antonminaichev/gophkeeper/internal/server/vault"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ItemCreator defines operations for creating items
type ItemCreator interface {
	Create(ctx context.Context, owner uuid.UUID, typ int16, payload, metaJSON []byte, alias *string) (uuid.UUID, int64, error)
}

// ItemReader defines basic read operations for items
type ItemReader interface {
	Get(ctx context.Context, owner uuid.UUID, id uuid.UUID) (*storage.Item, error)
}

// ItemSearcher defines search operations for items
type ItemSearcher interface {
	GetByHuman(ctx context.Context, owner uuid.UUID, hid int64) (*storage.Item, error)
	GetByAlias(ctx context.Context, owner uuid.UUID, alias string) (*storage.Item, error)
}

// ItemRetriever composes read and search operations
type ItemRetriever interface {
	ItemReader
	ItemSearcher
}

// ItemSnapshotLister defines operations for listing item snapshots
type ItemSnapshotLister interface {
	List(ctx context.Context, owner uuid.UUID, limit int32) ([]storage.Item, error)
}

// ItemChangeLister defines operations for listing item changes
type ItemChangeLister interface {
	ListChanges(ctx context.Context, owner uuid.UUID, after time.Time, afterID uuid.UUID, limit int32) ([]storage.Item, error)
}

// ItemLister composes snapshot and change listing operations
type ItemLister interface {
	ItemSnapshotLister
	ItemChangeLister
}

// ItemUpdater defines operations for updating items
type ItemUpdater interface {
	Update(ctx context.Context, owner uuid.UUID, id uuid.UUID, payload, metaJSON []byte, expectedVersion int64) (*storage.Item, error)
}

// ItemDeleter defines operations for deleting items
type ItemDeleter interface {
	Delete(ctx context.Context, owner uuid.UUID, id uuid.UUID) error
}

// ItemsRepository composes all item operations using interface composition
type ItemsRepository interface {
	ItemCreator
	ItemRetriever
	ItemLister
	ItemUpdater
	ItemDeleter
}

type VaultServer struct {
	pb.UnimplementedVaultServiceServer
	uc *vault.Usecase
}

func NewVaultServer(items ItemsRepository) *VaultServer {
	return &VaultServer{uc: vault.NewUsecase(items)}
}

// NewVaultServerWithComposition creates a vault server using interface composition
func NewVaultServerWithComposition(creator ItemCreator, retriever ItemRetriever, lister ItemLister, updater ItemUpdater, deleter ItemDeleter) *VaultServer {
	// Create a composed interface
	items := &composedItemsRepository{
		ItemCreator:   creator,
		ItemRetriever: retriever,
		ItemLister:    lister,
		ItemUpdater:   updater,
		ItemDeleter:   deleter,
	}
	return &VaultServer{uc: vault.NewUsecase(items)}
}

// composedItemsRepository implements ItemsRepository using composition
type composedItemsRepository struct {
	ItemCreator
	ItemRetriever
	ItemLister
	ItemUpdater
	ItemDeleter
}

// CreateItem creates a new item owned by the authenticated user.
func (s *VaultServer) CreateItem(ctx context.Context, r *pb.CreateItemRequest) (*pb.ItemID, error) {
	owner, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	id, err := s.uc.CreateItem(ctx, owner, r)
	if err != nil {
		return nil, err
	}
	return &pb.ItemID{Id: id.String()}, nil
}

// GetItemByRef fetches an item by UUID / human id / alias.
func (s *VaultServer) GetItemByRef(ctx context.Context, ref *pb.ItemRef) (*pb.Item, error) {
	owner, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.GetItemByRef(ctx, owner, ref)
}

// ListItems returns either a snapshot list or change feed (cursor-based).
func (s *VaultServer) ListItems(ctx context.Context, r *pb.ListRequest) (*pb.ListResponse, error) {
	owner, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.ListItems(ctx, owner, r)
}

// UpdateItem applies optimistic update by version.
func (s *VaultServer) UpdateItem(ctx context.Context, r *pb.UpdateItemRequest) (*pb.Item, error) {
	owner, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.UpdateItem(ctx, owner, r)
}

// DeleteItem marks item as deleted (tombstone).
func (s *VaultServer) DeleteItem(ctx context.Context, r *pb.ItemID) (*emptypb.Empty, error) {
	owner, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(strings.TrimSpace(r.GetId()))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad id")
	}
	if err := s.uc.DeleteItem(ctx, owner, id); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// userIDFromCtx extracts and validates owner id from context.
func userIDFromCtx(ctx context.Context) (uuid.UUID, error) {
	uid, ok := gauth.UserIDFrom(ctx)
	if !ok || strings.TrimSpace(uid) == "" {
		return uuid.Nil, status.Error(codes.Unauthenticated, "missing user")
	}
	id, err := uuid.Parse(uid)
	if err != nil {
		return uuid.Nil, status.Error(codes.Unauthenticated, "bad user id")
	}
	return id, nil
}

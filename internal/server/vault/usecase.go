package vault

import (
	"context"
	"strings"
	"time"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
	"github.com/antonminaichev/gophkeeper/internal/server/storage"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

type Usecase struct {
	items ItemsRepository
}

// NewUsecase creates a new vault usecase with the required item operations
func NewUsecase(items ItemsRepository) *Usecase {
	return &Usecase{items: items}
}

// NewUsecaseWithComposition creates a new vault usecase using interface composition
func NewUsecaseWithComposition(creator ItemCreator, retriever ItemRetriever, lister ItemLister, updater ItemUpdater, deleter ItemDeleter) *Usecase {
	// Create a composed interface
	items := &composedItemsRepository{
		ItemCreator:   creator,
		ItemRetriever: retriever,
		ItemLister:    lister,
		ItemUpdater:   updater,
		ItemDeleter:   deleter,
	}
	return &Usecase{items: items}
}

// composedItemsRepository implements ItemsRepository using composition
type composedItemsRepository struct {
	ItemCreator
	ItemRetriever
	ItemLister
	ItemUpdater
	ItemDeleter
}

func (u *Usecase) CreateItem(ctx context.Context, owner uuid.UUID, r *pb.CreateItemRequest) (uuid.UUID, error) {
	t := r.GetType()
	if t == 0 {
		t = pb.ItemType_LOGIN
	}
	if !validType(t) {
		return uuid.Nil, status.Error(codes.InvalidArgument, "unknown item type")
	}

	metaJSON, err := metaToJSON(r.GetMeta())
	if err != nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, "invalid meta")
	}

	var alias *string
	if a := strings.TrimSpace(r.GetAlias()); a != "" {
		alias = &a
	}

	id, _, err := u.items.Create(ctx, owner, int16(t), r.GetPayload(), metaJSON, alias)
	if err != nil {
		return uuid.Nil, status.Error(codes.Internal, "cannot create item")
	}
	return id, nil
}

// CreateItemWithCreator creates an item using only the ItemCreator interface
func CreateItemWithCreator(creator ItemCreator, ctx context.Context, owner uuid.UUID, r *pb.CreateItemRequest) (uuid.UUID, error) {
	t := r.GetType()
	if t == 0 {
		t = pb.ItemType_LOGIN
	}
	if !validType(t) {
		return uuid.Nil, status.Error(codes.InvalidArgument, "unknown item type")
	}

	metaJSON, err := metaToJSON(r.GetMeta())
	if err != nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, "invalid meta")
	}

	var alias *string
	if a := strings.TrimSpace(r.GetAlias()); a != "" {
		alias = &a
	}

	id, _, err := creator.Create(ctx, owner, int16(t), r.GetPayload(), metaJSON, alias)
	if err != nil {
		return uuid.Nil, status.Error(codes.Internal, "cannot create item")
	}
	return id, nil
}

func (u *Usecase) GetItemByRef(ctx context.Context, owner uuid.UUID, ref *pb.ItemRef) (*pb.Item, error) {
	var it *storage.Item
	var err error

	switch x := ref.GetRef().(type) {
	case *pb.ItemRef_Id:
		idStr := strings.TrimSpace(x.Id)
		itemID, parseErr := uuid.Parse(idStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "bad id")
		}
		it, err = u.items.Get(ctx, owner, itemID)
	case *pb.ItemRef_HumanId:
		it, err = u.items.GetByHuman(ctx, owner, x.HumanId)
	case *pb.ItemRef_Alias:
		a := strings.TrimSpace(x.Alias)
		if a == "" {
			return nil, status.Error(codes.InvalidArgument, "empty alias")
		}
		it, err = u.items.GetByAlias(ctx, owner, a)
	default:
		return nil, status.Error(codes.InvalidArgument, "empty ref")
	}
	if err != nil {
		return nil, status.Error(codes.NotFound, "item not found")
	}
	return toPB(it), nil
}

func (u *Usecase) ListItems(ctx context.Context, owner uuid.UUID, r *pb.ListRequest) (*pb.ListResponse, error) {
	limit := r.GetLimit()
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	cursorStr := strings.TrimSpace(r.GetCursor())
	// Snapshot mode (no cursor): non-deleted live items.
	if cursorStr == "" {
		items, err := u.items.List(ctx, owner, limit)
		if err != nil {
			return nil, status.Error(codes.Internal, "cannot list items")
		}
		out := make([]*pb.Item, 0, len(items))
		for i := range items {
			p := toPB(&items[i])
			p.Payload = nil // lightweight list
			out = append(out, p)
		}
		return &pb.ListResponse{Items: out}, nil
	}

	// Change feed mode.
	afterT, afterID, err := decodeCursor(cursorStr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad cursor")
	}
	changes, err := u.items.ListChanges(ctx, owner, afterT, afterID, limit)
	if err != nil {
		return nil, status.Error(codes.Internal, "cannot list changes")
	}

	resp := &pb.ListResponse{Items: make([]*pb.Item, 0, len(changes))}
	for i := range changes {
		p := toPB(&changes[i])
		p.Payload = nil
		resp.Items = append(resp.Items, p)
	}
	if n := len(changes); n > 0 {
		last := changes[n-1]
		resp.NextCursor = encodeCursor(last.UpdatedAt, last.ID)
	}
	return resp, nil
}

func (u *Usecase) UpdateItem(ctx context.Context, owner uuid.UUID, r *pb.UpdateItemRequest) (*pb.Item, error) {
	id, err := uuid.Parse(strings.TrimSpace(r.GetId()))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad id")
	}
	if r.GetExpectedVersion() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "expected_version is required")
	}
	metaJSON, err := metaToJSON(r.GetMeta())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid meta")
	}
	it, err := u.items.Update(ctx, owner, id, r.GetPayload(), metaJSON, r.GetExpectedVersion())
	if err != nil {
		// у репозитория может быть своя ошибка "версия не совпала" — здесь можно
		// отобразить её на codes.Aborted (optimistic lock failed)
		return nil, err
	}
	return toPB(it), nil
}

func (u *Usecase) DeleteItem(ctx context.Context, owner uuid.UUID, id uuid.UUID) error {
	if err := u.items.Delete(ctx, owner, id); err != nil {
		return status.Error(codes.Internal, "cannot delete item")
	}
	return nil
}

// validType checks if the item type is valid
func validType(t pb.ItemType) bool {
	switch t {
	case pb.ItemType_LOGIN, pb.ItemType_TEXT, pb.ItemType_BINARY, pb.ItemType_CARD:
		return true
	default:
		return false
	}
}

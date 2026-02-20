package clockwork

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInMemoryStorage_EvictsByMaxEntries(t *testing.T) {
	store := NewInMemoryStorage(1, 1024*1024)
	ctx := context.Background()

	require.NoError(t, store.Store(ctx, &Metadata{ID: "first"}))
	require.NoError(t, store.Store(ctx, &Metadata{ID: "second"}))

	_, err := store.Get(ctx, "first")
	require.Error(t, err)

	second, err := store.Get(ctx, "second")
	require.NoError(t, err)
	require.Equal(t, "second", second.ID)
}

func TestInMemoryStorage_ListMostRecentFirst(t *testing.T) {
	store := NewInMemoryStorage(10, 1024*1024)
	ctx := context.Background()

	require.NoError(t, store.Store(ctx, &Metadata{ID: "a"}))
	require.NoError(t, store.Store(ctx, &Metadata{ID: "b"}))
	require.NoError(t, store.Store(ctx, &Metadata{ID: "c"}))

	items, err := store.List(ctx, 2)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "c", items[0].ID)
	require.Equal(t, "b", items[1].ID)
}

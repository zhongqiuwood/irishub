package keeper

import (
	"github.com/irisnet/irishub/app/v3/oracle/internal/types"
	sdk "github.com/irisnet/irishub/types"
)

func (k Keeper) GetFeed(ctx sdk.Context, feedName string) (feed types.Feed, found bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(GetFeedKey(feedName))
	if bz == nil {
		return feed, false
	}
	k.cdc.MustUnmarshalBinaryLengthPrefixed(bz, feed)
	return feed, true
}

func (k Keeper) GetFeedByReqCtxID(ctx sdk.Context, requestContextID []byte) (feed types.Feed, found bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(GetReqCtxIDKey(requestContextID))
	feedName := string(bz)
	return k.GetFeed(ctx, feedName)
}

func (k Keeper) GetFeedResults(ctx sdk.Context, feedName string) (result types.FeedResults) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStoreReversePrefixIterator(store, GetFeedResultPrefixKey(feedName))
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var res types.FeedResult
		k.cdc.MustUnmarshalBinaryLengthPrefixed(iterator.Value(), &res)
		result = append(result, res)
	}
	return
}

func (k Keeper) setFeed(ctx sdk.Context, feed types.Feed) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshalBinaryLengthPrefixed(feed)
	store.Set(GetFeedKey(feed.FeedName), bz)
}

func (k Keeper) deleteFeed(ctx sdk.Context, feed types.Feed) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(GetFeedKey(feed.FeedName))
	store.Delete(GetReqCtxIDKey(feed.RequestContextID))
	store.Delete(GetFeedResultPrefixKey(feed.FeedName))
}

func (k Keeper) setFeedResult(ctx sdk.Context, feedName string, batchCounter uint64, latestHistory uint64, data types.Value) {
	store := ctx.KVStore(k.storeKey)
	result := types.FeedResult{
		Data:      data,
		Timestamp: ctx.BlockTime(),
	}
	delta := batchCounter - latestHistory
	if delta >= 1 {
		k.deleteOldestFeedResult(ctx, feedName, int(delta))
	}
	bz := k.cdc.MustMarshalBinaryLengthPrefixed(result)
	store.Set(GetFeedResultKey(feedName, batchCounter), bz)
}

func (k Keeper) deleteOldestFeedResult(ctx sdk.Context, feedName string, delta int) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, GetFeedResultPrefixKey(feedName))
	defer iterator.Close()
	for i := 1; iterator.Valid() && i <= delta; iterator.Next() {
		store.Delete(iterator.Key())
		i++
	}
}
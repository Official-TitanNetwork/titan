package asset

import (
	"context"
	"math/rand"
	"sort"

	"github.com/Filecoin-Titan/titan/node/asset/index"
	"github.com/Filecoin-Titan/titan/node/asset/storage"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-libipfs/blocks"
	"golang.org/x/xerrors"
)

// randomCheck implements validate.Asset interface
type randomCheck struct {
	randomSeed int64
	rand       *rand.Rand
	root       *cid.Cid
	storage.Storage
	idx *index.MultiIndexSorted
	lru *lruCache
}

// newRandomCheck creates a new instance of randomCheck
func newRandomCheck(randomSeed int64, storage storage.Storage, lru *lruCache) (*randomCheck, error) {
	check := &randomCheck{randomSeed: randomSeed, Storage: storage, lru: lru}
	if err := check.init(context.Background()); err != nil {
		return nil, err
	}
	log.Debugf("NewRandomCheck, randomSeed:%d, asset %s", randomSeed, check.root.String())
	return check, nil
}

func randomBlockFromAsset(root cid.Cid, randomSeed int64, lru *lruCache) (*randomCheck, error) {
	check := &randomCheck{randomSeed: randomSeed, lru: lru}
	check.rand = rand.New(rand.NewSource(randomSeed))
	check.root = &root

	idx, err := check.lru.assetIndex(root)
	if err != nil {
		return nil, xerrors.Errorf("asset index %w", err)
	}

	var ok bool
	if check.idx, ok = idx.(*index.MultiIndexSorted); !ok {
		return nil, xerrors.Errorf("idx is not MultiIndexSorted")
	}
	return check, nil
}

func (rc *randomCheck) init(ctx context.Context) error {
	if rc.rand == nil {
		rc.rand = rand.New(rand.NewSource(rc.randomSeed))
	}

	if rc.root == nil {
		asset, err := rc.randomAsset(ctx)
		if err != nil {
			return xerrors.Errorf("random asset %w", err)
		}
		rc.root = asset
	}

	if rc.idx == nil {
		idx, err := rc.lru.assetIndex(*rc.root)
		if err != nil {
			return xerrors.Errorf("asset index %w", err)
		}

		var ok bool
		if rc.idx, ok = idx.(*index.MultiIndexSorted); !ok {
			return xerrors.Errorf("idx is not MultiIndexSorted")
		}
	}

	return nil

}

// GetBlock returns a random block from the storage
func (rc *randomCheck) GetBlock(ctx context.Context) (blocks.Block, error) {
	sizeOfBucket := rc.idx.BucketCount()
	index := rc.rand.Intn(int(sizeOfBucket))
	_, records, err := rc.idx.GetBucketRecords(uint32(index))
	if err != nil {
		return nil, xerrors.Errorf("get bucket %w", err)
	}

	if len(records) == 0 {
		return nil, xerrors.Errorf("no block in bucket, index %d", index)
	}

	index = rc.rand.Intn(len(records))
	record := records[index]
	return rc.lru.getBlock(ctx, *rc.root, record.Cid)
}

// randomAsset returns a random asset from the storage
func (rc *randomCheck) randomAsset(ctx context.Context) (*cid.Cid, error) {
	bucketHashes, err := rc.GetBucketHashes(ctx)
	if err != nil {
		return nil, err
	}

	if len(bucketHashes) == 0 {
		return nil, xerrors.Errorf("no asset exist")
	}

	bucketIDs := make([]int, 0, len(bucketHashes))
	for k := range bucketHashes {
		bucketIDs = append(bucketIDs, int(k))
	}

	sort.Ints(bucketIDs)

	r := rand.New(rand.NewSource(rc.randomSeed))
	index := r.Intn(len(bucketIDs))
	bucketID := bucketIDs[index]

	cids, err := rc.GetAssetsInBucket(ctx, uint32(bucketID))
	if err != nil {
		return nil, xerrors.Errorf("get assets of bucket %w", err)
	}

	if len(cids) == 0 {
		return nil, xerrors.Errorf("no car exist in bucket %d", bucketID)
	}

	index = r.Intn(len(cids))
	cid := cids[index]
	return &cid, nil
}

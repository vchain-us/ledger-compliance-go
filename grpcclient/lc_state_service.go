package grpcclient

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"github.com/codenotary/immudb/pkg/api/schema"
	"github.com/codenotary/immudb/pkg/client/state"
	"github.com/codenotary/immudb/pkg/logger"
	"sync"
)

type lcStateService struct {
	stateProvider state.StateProvider
	uuidProvider  state.UUIDProvider
	cache         *FileCache
	serverUUID    string
	logger        logger.Logger
	sync.RWMutex
}

// NewLcStateService ...
func NewLcStateService(cache *FileCache,
	logger logger.Logger,
	stateProvider state.StateProvider,
	uuidProvider state.UUIDProvider) (state.StateService, error) {

	serverUUID, err := uuidProvider.CurrentUUID(context.Background())
	if err != nil {
		if err != ErrNoServerUuid {
			return nil, err
		}
		logger.Warningf(err.Error())
	}

	return &lcStateService{
		stateProvider: stateProvider,
		uuidProvider:  uuidProvider,
		cache:         cache,
		logger:        logger,
		serverUUID:    serverUUID,
	}, nil
}

func (r *lcStateService) GetState(ctx context.Context, apiKey string) (*schema.ImmutableState, error) {
	r.Lock()
	defer r.Unlock()

	st, err := r.cache.Get(r.serverUUID, obfuscateApiKey(apiKey))
	if err == nil {
		return st, nil
	}
	if err == ErrStateNotFound {
		if st, err := r.cache.GetAndClean(r.serverUUID, apiKey); err == nil {
			return st, nil
		}
	}

	st, err = r.stateProvider.CurrentState(ctx)
	if err != nil {
		return nil, err
	}

	if err := r.cache.Set(r.serverUUID, obfuscateApiKey(apiKey), st); err != nil {
		return nil, err
	}

	return st, nil
}

func (r *lcStateService) SetState(apiKey string, state *schema.ImmutableState) error {
	r.Lock()
	defer r.Unlock()
	return r.cache.Set(r.serverUUID, obfuscateApiKey(apiKey), state)
}

func (r *lcStateService) CacheLock() error {
	return r.cache.Lock(r.serverUUID)
}

func (r *lcStateService) CacheUnlock() error {
	return r.cache.Unlock()
}

func obfuscateApiKey(apiKey string) string {
	sha := sha256.Sum256([]byte(apiKey))
	return base64.StdEncoding.EncodeToString(sha[:])
}

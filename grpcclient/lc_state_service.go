package grpcclient

import (
	"context"
	"crypto/sha256"
	"encoding/base64"

	"github.com/codenotary/immudb/pkg/api/schema"
	"github.com/codenotary/immudb/pkg/client/cache"
	"github.com/codenotary/immudb/pkg/client/state"
)

type lcStateService struct {
	ss state.StateService
}

// NewLcStateService ...
func NewLcStateService(cache cache.Cache,
	logger LcLogger,
	stateProvider state.StateProvider,
	uuidProvider state.UUIDProvider) (state.StateService, error) {

	ss, err := state.NewStateService(cache, logger, stateProvider, uuidProvider)
	if err != nil {
		return nil, err
	}

	return &lcStateService{ss: ss}, nil

}

func (r *lcStateService) GetState(ctx context.Context, apiKey string) (*schema.ImmutableState, error) {
	return r.ss.GetState(ctx, obfuscateApiKey(apiKey))
}

func (r *lcStateService) SetState(apiKey string, state *schema.ImmutableState) error {
	return r.ss.SetState(obfuscateApiKey(apiKey), state)
}

func (r *lcStateService) CacheLock() error {
	return r.ss.CacheLock()
}

func (r *lcStateService) CacheUnlock() error {
	return r.ss.CacheUnlock()
}

func (r *lcStateService) SetServerIdentity(identity string) {
	r.ss.SetServerIdentity(identity)
}

func obfuscateApiKey(apiKey string) string {
	sha := sha256.Sum256([]byte(apiKey))
	return base64.StdEncoding.EncodeToString(sha[:])
}

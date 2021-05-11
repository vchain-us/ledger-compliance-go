package grpcclient

import "fmt"

var ErrStateNotFound = fmt.Errorf("could not find previous state")
var ErrCacheNotLocked = fmt.Errorf("cache is not locked")
var ErrCacheAlreadyLocked = fmt.Errorf("cache is already locked")
var ErrPrevStateNotFound   = fmt.Errorf("could not find previous state")
var ErrLocalStateCorrupted = fmt.Errorf("local state is corrupted")
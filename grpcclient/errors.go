package grpcclient

import "fmt"

var ErrNotLockedFile = fmt.Errorf("try to lock a not locked file")
var ErrStateNotFound = fmt.Errorf("could not find previous state")

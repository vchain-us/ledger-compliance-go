package grpcclient

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"github.com/codenotary/immudb/pkg/client/cache"
	"github.com/rogpeppe/go-internal/lockedfile"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/codenotary/immudb/pkg/api/schema"
	"github.com/golang/protobuf/proto"
)

// STATE_FN ...
const STATE_FN = ".state-"

type FileCache struct {
	Dir string
}

// NewFileCache returns a new file cache
func NewLcFileCache(dir string) *FileCache {
	return &FileCache{Dir: dir}
}

func (w *FileCache) Get(serverUUID, db string) (*schema.ImmutableState, error) {
	fn := w.getStateFileName(serverUUID)

	raw, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(raw), "\n")
	for _, line := range lines {
		if strings.Contains(line, db+":") {
			r := strings.Split(line, ":")
			if r[1] == "" {
				return nil, ErrStateNotFound
			}
			oldState, err := base64.StdEncoding.DecodeString(r[1])
			if err != nil {
				return nil, ErrStateNotFound
			}
			state := &schema.ImmutableState{}
			if err = proto.Unmarshal(oldState, state); err != nil {
				return nil, err
			}
			return state, nil
		}
	}
	return nil, ErrStateNotFound
}

// GetAndClean retrieve the state for db identifier and remove it from state file
func (w *FileCache) GetAndClean(serverUUID, db string) (*schema.ImmutableState, error) {
	state, err := w.Get(serverUUID, db)
	if err != nil && err != ErrStateNotFound {
		return nil, err
	}
	f, err := os.Open(w.getStateFileName(serverUUID))
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	var lines [][]byte
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, db+":") && line != "" {
			lines = append(lines, []byte(line))
		}
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	err = os.Truncate(w.getStateFileName(serverUUID), 0)
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(w.getStateFileName(serverUUID), bytes.Join(lines, []byte("\n")), 0644)
	if err != nil {
		return nil, err
	}
	return state, err
}

func (w *FileCache) Set(serverUUID, db string, state *schema.ImmutableState) error {
	raw, err := proto.Marshal(state)
	if err != nil {
		return err
	}
	fn := w.getStateFileName(serverUUID)

	input, _ := ioutil.ReadFile(fn)
	lines := strings.Split(string(input), "\n")

	newState := db + ":" + base64.StdEncoding.EncodeToString(raw)
	var exists bool
	for i, line := range lines {
		if strings.Contains(line, db+":") {
			exists = true
			lines[i] = newState
		}
	}
	if !exists {
		lines = append(lines, newState)
	}
	output := strings.Join(lines, "\n")

	if err = ioutil.WriteFile(fn, []byte(output), 0644); err != nil {
		return err
	}
	return nil
}

func getRootFileName(prefix []byte, serverUUID []byte) []byte {
	l1 := len(prefix)
	l2 := len(serverUUID)
	var fn = make([]byte, l1+l2)
	copy(fn[:], STATE_FN)
	copy(fn[l1:], serverUUID)
	return fn
}

func (w *FileCache) GetLocker(serverUUID string) cache.Locker {
	fn := w.getStateFileName(serverUUID)
	fm := lockedfile.MutexAt(fn)
	return &FileLocker{lm: fm}
}

type FileLocker struct {
	lm         *lockedfile.Mutex
	unlockFunc func()
}

func (fl *FileLocker) Lock() (err error) {
	fl.unlockFunc, err = fl.lm.Lock()
	return err
}

func (fl *FileLocker) Unlock() (err error) {
	if fl.unlockFunc == nil {
		return ErrNotLockedFile
	}
	fl.unlockFunc()
	return nil
}

func (w *FileCache) getStateFileName(serverUUID string) string {
	return filepath.Join(w.Dir, string(getRootFileName([]byte(STATE_FN), []byte(serverUUID))))
}

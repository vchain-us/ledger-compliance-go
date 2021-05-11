package grpcclient

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"github.com/codenotary/immudb/pkg/client/cache"
	"github.com/rogpeppe/go-internal/lockedfile"
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
	stateFile *lockedfile.File
}

// NewFileCache returns a new file cache
func NewLcFileCache(dir string) *FileCache {
	return &FileCache{Dir: dir}
}

func (w *FileCache) Get(serverUUID string, db string) (*schema.ImmutableState, error) {
	if w.stateFile == nil {
		return nil, ErrCacheNotLocked
	}
	scanner := bufio.NewScanner(w.stateFile)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, db+":") {
			r := strings.Split(line, ":")
			if r[1] == "" {
				return nil, ErrStateNotFound
			}
			oldState, err := base64.StdEncoding.DecodeString(r[1])
			if err != nil {
				return nil, cache.ErrLocalStateCorrupted
			}
			state := &schema.ImmutableState{}
			if err = proto.Unmarshal(oldState, state); err != nil {
				return nil, cache.ErrLocalStateCorrupted
			}
			return state, nil
		}
	}
	return nil, ErrStateNotFound
}

// GetAndClean retrieve the state for db identifier and remove it from state file
func (w *FileCache) GetAndClean(serverUUID, db string) (*schema.ImmutableState, error) {
	state, err := w.Get(serverUUID, db)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(w.stateFile)
	scanner.Split(bufio.ScanLines)
	var lines [][]byte
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, db+":") && line != "" {
			lines = append(lines, []byte(line))
		}
	}
	output := bytes.Join(lines, []byte("\n"))

	_, err = w.stateFile.WriteAt(output,0)
	if err != nil {
		return nil, err
	}

	return state, err
}

func (w *FileCache) Set(serverUUID string, db string, state *schema.ImmutableState) error {
	if w.stateFile == nil {
		return ErrCacheNotLocked
	}
	raw, err := proto.Marshal(state)
	if err != nil {
		return err
	}

	newState := db + ":" + base64.StdEncoding.EncodeToString(raw)
	var exists bool

	scanner := bufio.NewScanner(w.stateFile)
	scanner.Split(bufio.ScanLines)
	var lines [][]byte
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, db+":") {
			exists = true
			lines = append(lines, []byte(newState))
		}
	}
	if !exists {
		lines = append(lines, []byte(newState))
	}
	output := bytes.Join(lines, []byte("\n"))

	_, err = w.stateFile.WriteAt( output, 0)
	if err != nil {
		return err
	}
	return nil
}


func (w *FileCache) Lock(serverUUID string) (err error) {
	if w.stateFile != nil {
		return ErrCacheAlreadyLocked
	}
	w.stateFile, err = lockedfile.OpenFile(w.getStateFilePath(serverUUID), os.O_RDWR | os.O_CREATE, 0755)
	return err
}

func (w *FileCache) Unlock() (err error) {
	defer func() {
		w.stateFile = nil
	}()
	return w.stateFile.Close()
}

func (w *FileCache) getStateFilePath(UUID string) string {
	return filepath.Join(w.Dir, STATE_FN+UUID)
}

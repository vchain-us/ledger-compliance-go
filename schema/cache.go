package schema

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

func (r *VCNArtifactsGetRequest) GetKey(prefix ...string) string {
	pieces := make([]string, 0, len(r.Hashes)+len(prefix))
	pieces = append(pieces, prefix...)
	copy(pieces, r.Hashes)
	sort.Strings(pieces)
	pieces = append(pieces, fmt.Sprintf("%d", r.GetProveSinceTx()))
	fullKey := strings.Join(pieces, "")
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fullKey)))
}

package schema

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

func (r *VCNArtifactsGetRequest) GetKey(prefix ...string) string {
	pieces := make([]string, len(r.Hashes))
	copy(pieces, r.Hashes)
	pieces = append(pieces, prefix...)
	sort.Strings(pieces)
	pieces = append(pieces, fmt.Sprintf("%d", r.GetProveSinceTx()))
	fullKey := strings.Join(pieces, "")
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fullKey)))
}

func (r *VCNSearchRequest) GetKey(prefix ...string) string {
	pieces := make([]string, 0)
	pieces = append(pieces, prefix...)
	pieces = append(pieces, r.Hash)
	pieces = append(pieces, r.SignerID)
	pieces = append(pieces, r.AType.String())
	pieces = append(pieces, r.Type.String())
	pieces = append(pieces, fmt.Sprintf("%d", r.Limit))
	pieces = append(pieces, fmt.Sprintf("%t", r.Desc))
	pieces = append(pieces, fmt.Sprintf("%d", r.StartTS))
	pieces = append(pieces, fmt.Sprintf("%d", r.EndTS))
	pieces = append(pieces, r.AttachName)
	pieces = append(pieces, r.ArtifactName)
	pieces = append(pieces, r.UID)
	sort.Strings(pieces)
	fullKey := strings.Join(pieces, "")
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fullKey)))
}

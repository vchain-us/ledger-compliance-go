package schema

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

func (r *VCNArtifactsGetRequest) GetKey(prefix ...string) (string, error) {
	pieces := make([]string, len(r.Hashes))
	copy(pieces, r.Hashes)
	pieces = append(pieces, prefix...)
	pieces = append(pieces, fmt.Sprintf("%d", r.GetProveSinceTx()))
	fullKey := strings.Join(pieces, "\n")
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fullKey))), nil
}

func (r *VCNSearchRequest) GetKey(prefix ...string) (string, error) {
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
	fullKey := strings.Join(pieces, "\n")
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fullKey))), nil
}

func (r *VCNLabelsGetRequest) GetKey(prefix ...string) (string, error) {
	pieces := make([]string, len(r.Request))
	for _, v := range r.Request {
		pieces = append(pieces, v.Hash)
	}
	pieces = append(pieces, prefix...)
	fullKey := strings.Join(pieces, "\n")
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fullKey))), nil
}

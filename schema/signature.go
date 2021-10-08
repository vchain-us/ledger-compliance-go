package schema

// VCNArtifactSignaturesKeySuffix is the suffix for the key of the artifact signatures entry.
const VCNArtifactSignaturesKeySuffix = ".signatures"

// VCNArtifactSignatures holds the persisted artifact signatures.
type VCNArtifactSignatures struct {
	Artifact    []byte
	Attachments map[string][]byte
	SignedAt    []byte
}

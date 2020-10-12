package grpcclient

import immuschema "github.com/codenotary/immudb/pkg/api/schema"

func (c *LcClient) addLedgerPrefix(key []byte) []byte {

	var prefixed = make([]byte, len(c.signerId)+1+len(key))
	copy(prefixed[0:], c.signerId+".")
	copy(prefixed[len(c.signerId)+1:], key)

	return prefixed
}

func (c *LcClient) stripLedgerPrefix(key []byte) []byte {
	kLen := len(key)
	sigIdLen := len(c.signerId) + 1 // remove also .
	hashLen := kLen - (sigIdLen)
	var prefixed = make([]byte, hashLen)
	copy(prefixed[:], key[sigIdLen:])
	return prefixed
}

func (c *LcClient) stripLedgerPrefixList(list *immuschema.StructuredItemList) *immuschema.StructuredItemList {
	for _, v := range list.Items {
		v.Key = c.stripLedgerPrefix(v.Key)
	}
	return list
}

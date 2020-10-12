package grpcclient

import immuschema "github.com/codenotary/immudb/pkg/api/schema"

func (c *LcClient) addLedgerPrefix(key []byte) []byte {
	lenPrefixed := len(c.signerId) + 1 + len(key)
	if c.PluginPrefix != "" {
		lenPrefixed += len(c.PluginPrefix) + 1
	}
	var prefixed = make([]byte, lenPrefixed)
	var pluginPrefixSpace = 0
	if c.PluginPrefix != "" {
		copy(prefixed[0:], c.PluginPrefix+".")
		pluginPrefixSpace = len(c.PluginPrefix) + 1
	}

	copy(prefixed[pluginPrefixSpace:], c.signerId+".")
	copy(prefixed[len(c.signerId)+1+pluginPrefixSpace:], key)

	return prefixed
}

func (c *LcClient) stripLedgerPrefix(key []byte) []byte {
	kLen := len(key)
	sigIdLen := len(c.signerId) + 1 // remove also .
	if c.PluginPrefix != "" {
		sigIdLen += len(c.PluginPrefix) + 1
	}
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

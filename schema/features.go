package schema

func (x *Features) Map() map[string]bool {
	elementMap := make(map[string]bool)
	for _, v := range x.Feat {
		elementMap[v] = true
	}
	return elementMap
}

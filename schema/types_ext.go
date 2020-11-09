package schema

import (
	immuschema "github.com/codenotary/immudb/pkg/api/schema"
	"github.com/codenotary/immudb/pkg/client"
	"github.com/golang/protobuf/ptypes/timestamp"
)

type StructuredItemExtList struct {
	Items []*StructuredItemExt `json:"items,omitempty"`
}

type StructuredItemExt struct {
	Item      *immuschema.StructuredItem `json:"item,omitempty"`
	Timestamp *timestamp.Timestamp       `json:"timestamp,omitempty"`
}

type ZStructuredItemExtList struct {
	Items []*ZStructuredItemExt `json:"items,omitempty"`
}

type ZStructuredItemExt struct {
	Item      *immuschema.ZStructuredItem `json:"item,omitempty"`
	Timestamp *timestamp.Timestamp        `json:"timestamp,omitempty"`
}

type VerifiedItemExt struct {
	Item      *client.VerifiedItem `json:"item,omitempty"`
	Timestamp *timestamp.Timestamp `json:"timestamp,omitempty"`
}

func (list *ItemExtList) ToSItemExtList() (*StructuredItemExtList, error) {
	slist := &StructuredItemExtList{}
	for _, item := range list.Items {
		i, err := item.Item.ToSItem()
		if err != nil {
			return nil, err
		}
		itemExt := &StructuredItemExt{
			Item:      i,
			Timestamp: item.Timestamp,
		}
		slist.Items = append(slist.Items, itemExt)
	}
	return slist, nil
}

func (zlist *ZItemExtList) ToZSItemExtList() (*ZStructuredItemExtList, error) {
	slist := &ZStructuredItemExtList{}
	for _, zitem := range zlist.Items {
		i, err := zitem.Item.Item.ToSItem()
		if err != nil {
			return nil, err
		}
		itemExt := &ZStructuredItemExt{
			Item: &immuschema.ZStructuredItem{
				Item:          i,
				Score:         zitem.Item.Score,
				CurrentOffset: zitem.Item.CurrentOffset,
				Index:         zitem.Item.Index,
			},
			Timestamp: zitem.Timestamp,
		}
		slist.Items = append(slist.Items, itemExt)
	}
	return slist, nil
}

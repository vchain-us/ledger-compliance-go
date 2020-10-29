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

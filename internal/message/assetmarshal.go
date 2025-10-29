package message

import (
	"encoding/json"
	"fmt"
	"log"

	"perun.network/go-perun/channel"
)

// Helper function to marshal an Asset field into JSON.
func marshalAsset(asset Asset) ([]byte, error) {
	assetJSON, err := asset.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Asset json.RawMessage `json:"asset"`
	}{
		Asset: assetJSON,
	})
}

// Helper function to unmarshal an Asset field from JSON, determining the asset type dynamically.
func unmarshalAsset(data []byte) (Asset, error) {
	var temp struct {
		Asset json.RawMessage `json:"asset"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return nil, err
	}

	var assetType struct {
		AssetType string `json:"assetType"`
	}
	if err := json.Unmarshal(temp.Asset, &assetType); err != nil {
		return nil, err
	}

	var asset Asset
	switch assetType.AssetType {
	case "Ethereum":
		var ethAsset EthereumAsset
		if err := json.Unmarshal(temp.Asset, &ethAsset); err != nil {
			return nil, err
		}
		asset = &ethAsset
	case "Solana":
		var solanaAsset SolanaAsset
		if err := json.Unmarshal(temp.Asset, &solanaAsset); err != nil {
			return nil, err
		}
		asset = &solanaAsset
	default:
		return nil, fmt.Errorf("unsupported asset type: %s", assetType.AssetType)
	}

	return asset, nil
}

// MarshalJSON marshals GetDecimals into JSON.
func (g GetDecimals) MarshalJSON() ([]byte, error) {
	asset, err := marshalAsset(g.Asset)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Asset json.RawMessage `json:"asset"`
	}{
		Asset: asset,
	})
}

// UnmarshalJSON unmarshals GetDecimals from JSON.
func (g *GetDecimals) UnmarshalJSON(data []byte) error {
	var temp struct {
		Asset json.RawMessage `json:"asset"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	asset, err := unmarshalAsset(temp.Asset)
	if err != nil {
		return err
	}
	g.Asset = asset
	return nil
}

// MarshalJSON marshals GetAddress into JSON.
func (c ChannelState) MarshalJSON() ([]byte, error) {
	var assetJSONs []json.RawMessage
	for _, asset := range c.Assets {
		assetJSON, err := marshalAsset(asset)
		if err != nil {
			return nil, err
		}
		assetJSONs = append(assetJSONs, assetJSON)
	}

	return json.Marshal(struct {
		Assets      []json.RawMessage `json:"assets"`
		Backends    []int             `json:"backends"`
		Balance     []Balance         `json:"balance"`
		PeerBalance []Balance         `json:"peerBalance"`
		IsFinal     bool              `json:"isFinal"`
	}{
		Assets:      assetJSONs,
		Backends:    c.Backends,
		Balance:     c.Balance,
		PeerBalance: c.PeerBalance,
		IsFinal:     c.IsFinal,
	})
}

// UnmarshalJSON unmarshals ChannelState from JSON.
func (c *ChannelState) UnmarshalJSON(data []byte) error {
	var temp struct {
		Balance     []Balance         `json:"balance"`
		PeerBalance []Balance         `json:"peerBalance"`
		Assets      []json.RawMessage `json:"assets"`
		Backends    []int             `json:"backends"`
		IsFinal     bool              `json:"isFinal"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	var assets []Asset
	for _, rawAsset := range temp.Assets {
		asset, err := unmarshalAsset(rawAsset)
		if err != nil {
			return err
		}
		assets = append(assets, asset)
	}

	c.Assets = assets
	c.Backends = temp.Backends
	c.Balance = temp.Balance
	c.PeerBalance = temp.PeerBalance
	c.IsFinal = temp.IsFinal
	return nil
}

func (c Order) MarshalJSON() ([]byte, error) {
	var baseJSON json.RawMessage
	var quoteJSON json.RawMessage
	baseJSON, err := marshalAsset(c.Base)
	if err != nil {
		return nil, err
	}

	quoteJSON, err = marshalAsset(c.Quote)
	if err != nil {
		return nil, err
	}

	return json.Marshal(struct {
		ID        OrderID         `json:"id"`
		ChannelID channel.ID      `json:"channelID"`
		MakerIdx  channel.Index   `json:"makerIdx"`  // who created it, 0/1
		Side      OrderSide       `json:"side"`      // "bid" or "ask"
		Base      json.RawMessage `json:"base"`      // asset being bought/sold
		Quote     json.RawMessage `json:"quote"`     // pricing asset
		Price     string          `json:"price"`     // decimal string
		Amount    string          `json:"amount"`    // base units
		Status    OrderStatus     `json:"status"`    // lifecycle status
		CreatedAt int64           `json:"createdAt"` // unix seconds
		ExpiresAt *int64          `json:"expiresAt,omitempty"`
		ClientTag string          `json:"clientTag,omitempty"` // optional client tag
	}{
		ID:        c.ID,
		ChannelID: c.ChannelID,
		MakerIdx:  c.MakerIdx,
		Side:      c.Side,
		Base:      baseJSON,
		Quote:     quoteJSON,
		Price:     c.Price,
		Amount:    c.Amount,
		Status:    c.Status,
		CreatedAt: c.CreatedAt,
		ExpiresAt: c.ExpiresAt,
		ClientTag: c.ClientTag,
	})
}

func (c *Order) UnmarshalJSON(data []byte) error {
	var temp struct {
		ID        OrderID         `json:"id"`
		ChannelID channel.ID      `json:"channelID"`
		MakerIdx  channel.Index   `json:"makerIdx"`  // who created it, 0/1
		Side      OrderSide       `json:"side"`      // "bid" or "ask"
		Base      json.RawMessage `json:"base"`      // asset being bought/sold
		Quote     json.RawMessage `json:"quote"`     // pricing asset
		Price     string          `json:"price"`     // decimal string
		Amount    string          `json:"amount"`    // base units
		Status    OrderStatus     `json:"status"`    // lifecycle status
		CreatedAt int64           `json:"createdAt"` // unix seconds
		ExpiresAt *int64          `json:"expiresAt,omitempty"`
		ClientTag string          `json:"clientTag,omitempty"` // optional client tag
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	baseAsset, err := unmarshalAsset(temp.Base)
	if err != nil {
		return err
	}
	quoteAsset, err := unmarshalAsset(temp.Quote)
	if err != nil {
		return err
	}

	c.ID = temp.ID
	c.ChannelID = temp.ChannelID
	c.MakerIdx = temp.MakerIdx
	c.Side = temp.Side
	c.Base = baseAsset
	c.Quote = quoteAsset
	c.Price = temp.Price
	c.Amount = temp.Amount
	c.Status = temp.Status
	c.CreatedAt = temp.CreatedAt
	c.ExpiresAt = temp.ExpiresAt
	c.ClientTag = temp.ClientTag
	return nil
}

// MarshalJSON marshals GetAddress into JSON.
func (c GetQuote) MarshalJSON() ([]byte, error) {
	fromAssetJSON, err := marshalAsset(c.FromAsset)
	if err != nil {
		return nil, err
	}
	toAssetJSON, err := marshalAsset(c.ToAsset)
	if err != nil {
		return nil, err
	}

	serialized, err := json.Marshal(struct {
		FromAsset json.RawMessage `json:"fromAsset"`
		ToAsset   json.RawMessage `json:"toAsset"`
		Hub       string          `json:"hub"`
	}{
		FromAsset: fromAssetJSON,
		ToAsset:   toAssetJSON,
		Hub:       c.Hub,
	})
	if err != nil {
		log.Println("GetQuote MarshalJSON3: Error marshaling final struct: ", err)
		return nil, err
	}
	return serialized, nil
}

// UnmarshalJSON unmarshals GetQuote from JSON.
func (c *GetQuote) UnmarshalJSON(data []byte) error {
	var temp struct {
		FromAsset json.RawMessage `json:"fromAsset"`
		ToAsset   json.RawMessage `json:"toAsset"`
		Hub       string          `json:"hub"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	fromAsset, err := unmarshalAsset(temp.FromAsset)
	if err != nil {
		return err
	}
	toAsset, err := unmarshalAsset(temp.ToAsset)
	if err != nil {
		return err
	}
	c.FromAsset = fromAsset
	c.ToAsset = toAsset
	c.Hub = temp.Hub
	return nil
}

// MarshalJSON marshals GetAddress into JSON.
func (c GetHubBalance) MarshalJSON() ([]byte, error) {
	assetJSON, err := marshalAsset(c.Asset)
	if err != nil {
		return nil, err
	}

	serialized, err := json.Marshal(struct {
		Asset json.RawMessage `json:"asset"`
		Hub   string          `json:"hub"`
	}{
		Asset: assetJSON,
		Hub:   c.Hub,
	})
	if err != nil {
		log.Println("GetQuote MarshalJSON3: Error marshaling final struct: ", err)
		return nil, err
	}
	return serialized, nil
}

// UnmarshalJSON unmarshals GetHubBalance from JSON.
func (c *GetHubBalance) UnmarshalJSON(data []byte) error {
	var temp struct {
		Asset json.RawMessage `json:"asset"`
		Hub   string          `json:"hub"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	asset, err := unmarshalAsset(temp.Asset)
	if err != nil {
		return err
	}
	c.Asset = asset
	c.Hub = temp.Hub
	return nil
}

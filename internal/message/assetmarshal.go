package message

import (
	"encoding/json"
	"fmt"
	"log"
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
	case "Stellar":
		var stellarAsset StellarAsset
		if err := json.Unmarshal(temp.Asset, &stellarAsset); err != nil {
			return nil, err
		}
		asset = &stellarAsset
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
		Assets      []json.RawMessage `json:"assets"`
		Backends    []int             `json:"backends"`
		Balance     []Balance         `json:"balance"`
		PeerBalance []Balance         `json:"peerBalance"`
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

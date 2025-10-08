package message

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"reflect"

	ethchannel "github.com/perun-network/perun-eth-backend/channel"
	ethwallet "github.com/perun-network/perun-eth-backend/wallet"

	"github.com/pkg/errors"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/wallet"
)

// IDLen the length of a channelID.
const IDLen = 32

// Balance is a JSON encodable representation of a channel balance.
type Balance = BigInt

// MakeBalance creates a balance from a channel.Bal.
func MakeBalance(b channel.Bal) Balance {
	return MakeBigInt(b)
}

// ChannelState is a JSON encodable representation of a channel state.

// SignedState is the response to a GetSignedState request.
// SignedState channel.SignedState
type SignedState struct {
	Params *channel.Params `json:"params"`
	State  *channel.State  `json:"state"`
	Sigs   []wallet.Sig    `json:"sigs"`
}

// MarshalJSON marshals the signed state into a JSON message.
func (s SignedState) MarshalJSON() ([]byte, error) {
	// Marshal Params
	var binaryAddresses []json.RawMessage
	for _, partMap := range s.Params.Parts {
		marshalledPart := make(map[wallet.BackendID]string)
		for key, address := range partMap {
			binaryData, err := address.MarshalBinary()
			if err != nil {
				return nil, err
			}

			marshalledPart[key] = base64.StdEncoding.EncodeToString(binaryData)
		}
		jsonData, err := json.Marshal(marshalledPart)
		if err != nil {
			return nil, err
		}

		binaryAddresses = append(binaryAddresses, jsonData)
	}

	appBinary, paramsAppType, err := serialiseApp(s.Params.App)
	if err != nil {
		return nil, err
	}

	paramsByte, err := json.Marshal(struct {
		ID                [IDLen]byte       `json:"id"`
		ChallengeDuration uint64            `json:"challengeDuration"`
		Parts             []json.RawMessage `json:"parts"`
		App               []byte            `json:"app"`
		AppType           string            `json:"appType"`
		Nonce             string            `json:"nonce"`
		LedgerChannel     bool              `json:"ledgerChannel"`
		VirtualChannel    bool              `json:"virtualChannel"`
	}{
		s.Params.ID(),
		s.Params.ChallengeDuration,
		binaryAddresses,
		appBinary,
		paramsAppType,
		s.Params.Nonce.String(),
		s.Params.LedgerChannel,
		s.Params.VirtualChannel,
	})
	if err != nil {
		return nil, err
	}

	// Marshal State
	binaryAssets := make([]string, len(s.State.Allocation.Assets))
	for i, v := range s.State.Allocation.Assets {
		binaryAsset, err := v.MarshalBinary()
		if err != nil {
			return nil, err
		}
		binaryAssets[i] = base64.StdEncoding.EncodeToString(binaryAsset)
	}

	binaryData, err := s.State.Data.MarshalBinary()
	if err != nil {
		return nil, err
	}
	dataByte, err := json.Marshal(binaryData)
	if err != nil {
		return nil, err
	}

	balances := [][]string{}
	for _, v := range s.State.Allocation.Balances {
		balance := []string{}
		for _, bal := range v {
			balance = append(balance, bal.String())
		}
		balances = append(balances, balance)
	}

	backends := make([]int, len(s.State.Allocation.Backends))
	for i, v := range s.State.Allocation.Backends {
		backends[i] = int(v)
	}

	var locked []struct {
		ASId     channel.ID      `json:"asId"`
		Bals     []string        `json:"bals"`
		IndexMap []channel.Index `json:"indexMap"`
	}
	for _, v := range s.State.Allocation.Locked {
		lockedBal := []string{}
		for _, v := range v.Bals {
			lockedBal = append(lockedBal, v.String())
		}
		locked = append(locked, struct {
			ASId     channel.ID      `json:"asId"`
			Bals     []string        `json:"bals"`
			IndexMap []channel.Index `json:"indexMap"`
		}{v.ID, lockedBal, v.IndexMap})
	}

	stateAppBinary, stateAppType, err := serialiseApp(s.State.App)
	if err != nil {
		return nil, err
	}
	stateByte, err := json.Marshal(struct {
		ID         channel.ID `json:"id"`
		Version    uint64     `json:"version"`
		App        []byte     `json:"app"`
		AppType    string     `json:"appType"`
		Allocation struct {
			Assets   []string   `json:"assets"`
			Backends []int      `json:"backends"`
			Balances [][]string `json:"balances"`
			Locked   []struct {
				ASId     channel.ID      `json:"asId"`
				Bals     []string        `json:"bals"`
				IndexMap []channel.Index `json:"indexMap"`
			} `json:"locked"`
		} `json:"allocation"`
		Data    json.RawMessage `json:"data"`
		IsFinal bool            `json:"isFinal"`
	}{
		s.State.ID,
		s.State.Version,
		stateAppBinary,
		stateAppType,
		struct {
			Assets   []string   `json:"assets"`
			Backends []int      `json:"backends"`
			Balances [][]string `json:"balances"`
			Locked   []struct {
				ASId     channel.ID      `json:"asId"`
				Bals     []string        `json:"bals"`
				IndexMap []channel.Index `json:"indexMap"`
			} `json:"locked"`
		}{
			binaryAssets,
			backends,
			balances,
			locked,
		},
		dataByte,
		s.State.IsFinal,
	})
	if err != nil {
		return nil, err
	}

	tempStruct := struct {
		Params json.RawMessage `json:"params"`
		State  json.RawMessage `json:"state"`
		Sigs   []wallet.Sig    `json:"sigs"`
	}{
		Params: paramsByte,
		State:  stateByte,
		Sigs:   s.Sigs,
	}

	return json.Marshal(tempStruct)
}

// UnmarshalJSON unmarshals a JSON message back into a signed state if possible.
func (s *SignedState) UnmarshalJSON(data []byte) error {
	type TemptStruct struct {
		Params json.RawMessage `json:"params"`
		State  json.RawMessage `json:"state"`
		Sigs   []wallet.Sig    `json:"sigs"`
	}

	var tempStruct TemptStruct
	if err := json.Unmarshal(data, &tempStruct); err != nil {
		return err
	}

	// Unmarshal Params
	var paramsData struct {
		ID                [IDLen]byte       `json:"id"`
		ChallengeDuration uint64            `json:"challengeDuration"`
		Parts             []json.RawMessage `json:"parts"`
		App               []byte            `json:"app"`
		AppType           string            `json:"appType"`
		Nonce             string            `json:"nonce"`
		LedgerChannel     bool              `json:"ledgerChannel"`
		VirtualChannel    bool              `json:"virtualChannel"`
	}
	if err := json.Unmarshal(tempStruct.Params, &paramsData); err != nil {
		return err
	}

	paramsParts := make([]map[wallet.BackendID]wallet.Address, len(paramsData.Parts))
	for i, part := range paramsData.Parts {
		var marshalledPart map[wallet.BackendID]string
		if err := json.Unmarshal(part, &marshalledPart); err != nil {
			return err
		}

		// Decode the base64 strings back into wallet.Address instances
		addressMap := make(map[wallet.BackendID]wallet.Address)
		for key, base64Data := range marshalledPart {
			binaryData, err := base64.StdEncoding.DecodeString(base64Data)
			if err != nil {
				return err
			}

			address := wallet.NewAddress(key)
			if err := address.UnmarshalBinary(binaryData); err != nil {
				return err
			}
			addressMap[key] = address
		}

		paramsParts[i] = addressMap
	}
	paramsApp, err := deserialiseApp(paramsData.App, paramsData.AppType)
	if err != nil {
		return err
	}
	nonce := new(big.Int)
	_, err = fmt.Sscan(paramsData.Nonce, nonce)
	if err != nil {
		return err
	}
	params, err := channel.NewParams(
		paramsData.ChallengeDuration,
		paramsParts,
		paramsApp,
		nonce,
		paramsData.LedgerChannel,
		paramsData.VirtualChannel)
	if err != nil {
		return err
	}

	if params.ID() != paramsData.ID {
		return errors.New("ID is not unique")
	}

	// Unmarshal State
	var stateData struct {
		ID         channel.ID `json:"id"`
		Version    uint64     `json:"version"`
		App        []byte     `json:"app"`
		AppType    string     `json:"appType"`
		Allocation struct {
			Assets   []string   `json:"assets"`
			Backends []int      `json:"backends"`
			Balances [][]string `json:"balances"`
			Locked   []struct {
				ASId     channel.ID      `json:"asId"`
				Bals     []string        `json:"bals"`
				IndexMap []channel.Index `json:"indexMap"`
			} `json:"locked"`
		} `json:"allocation"`
		Data    json.RawMessage `json:"data"`
		IsFinal bool            `json:"isFinal"`
	}
	if err := json.Unmarshal(tempStruct.State, &stateData); err != nil {
		return err
	}

	stateApp, err := deserialiseApp(stateData.App, stateData.AppType)
	if err != nil {
		return err
	}

	var dataBinary []byte
	err = json.Unmarshal(stateData.Data, &dataBinary)
	if err != nil {
		return err
	}
	dataVal := stateApp.NewData()
	err = dataVal.UnmarshalBinary(dataBinary)
	if err != nil {
		return err
	}

	stateBackends := make([]wallet.BackendID, len(stateData.Allocation.Assets))
	for i, backend := range stateData.Allocation.Backends {
		stateBackends[i] = wallet.BackendID(backend)
	}

	stateAssets := make([]channel.Asset, len(stateData.Allocation.Assets))
	for i, v := range stateData.Allocation.Assets {
		binaryAsset, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return err
		}
		asset := channel.NewAsset(stateBackends[i])
		if err := asset.UnmarshalBinary(binaryAsset); err != nil {
			return err
		}
		stateAssets[i] = asset
	}

	stateBalances := [][]channel.Bal{}
	for _, v := range stateData.Allocation.Balances {
		balance := []channel.Bal{}
		for _, balString := range v {
			bal := new(big.Int)
			_, err = fmt.Sscan(balString, bal)
			if err != nil {
				return err
			}
			balance = append(balance, bal)
		}
		stateBalances = append(stateBalances, balance)
	}

	stateLocked := []channel.SubAlloc{}
	for _, v := range stateData.Allocation.Locked {
		lockedBals := []channel.Bal{}
		for _, v := range v.Bals {
			lockedBal := new(big.Int)
			_, err = fmt.Sscan(v, lockedBal)
			if err != nil {
				return err
			}
			lockedBals = append(lockedBals, lockedBal)
		}
		subAlloc := channel.SubAlloc{
			ID:       v.ASId,
			Bals:     lockedBals,
			IndexMap: v.IndexMap,
		}
		stateLocked = append(stateLocked, subAlloc)
	}

	s.Params = params
	s.State = &channel.State{
		ID:      stateData.ID,
		Version: stateData.Version,
		App:     stateApp,
		Allocation: channel.Allocation{
			Assets:   stateAssets,
			Backends: stateBackends,
			Balances: stateBalances,
			Locked:   stateLocked,
		},
		Data:    dataVal,
		IsFinal: stateData.IsFinal,
	}
	s.Sigs = tempStruct.Sigs

	return nil
}

// ChannelState is a JSON encodable representation of a channel state.
type ChannelState struct {
	Assets      []Asset   `json:"assets"`
	Backends    []int     `json:"backends"`
	Balance     []Balance `json:"balance"`
	PeerBalance []Balance `json:"peerBalance"`
	IsFinal     bool      `json:"isFinal"`
}

// NewChannelState creates a new channel state from the given parameters.
func NewChannelState(
	assets []channel.Asset, backends []wallet.BackendID, balance, peerBalance []Balance, isFinal bool,
) ChannelState {
	bs := make([]int, len(backends))
	for i, v := range backends {
		bs[i] = int(v)
	}
	return ChannelState{
		Assets:      MakeAssetsGPAsAssets(assets),
		Backends:    bs,
		Balance:     balance,
		PeerBalance: peerBalance,
		IsFinal:     isFinal,
	}
}

// MakeBals creates two balances slices from channel.Balances.
func MakeBals(
	balances channel.Balances, myIdx, peerIdx channel.Index,
) (myBals, peerBals []Balance) {
	myBals = make([]Balance, len(balances))
	peerBals = make([]Balance, len(balances))

	for i := 0; i < len(balances); i++ {
		myBals[i] = MakeBalance(balances[i][myIdx])
		peerBals[i] = MakeBalance(balances[i][peerIdx])
	}
	return
}

// MakePerunBals converts two slices of balances to channel.Balances.
func MakePerunBals(
	myBals, peerBals []Balance, myIdx, peerIdx channel.Index,
) (balances channel.Balances, err error) {
	log.Println("My Bals: ", myBals)
	log.Println("Peer Bals: ", peerBals)
	if len(myBals) != len(peerBals) {
		return nil, errors.New("balances have different lengths")
	}

	if myIdx+peerIdx > 1 {
		return nil, errors.New("invalid 2 party channel participant indices")
	}

	balances = make(channel.Balances, len(myBals))
	for i := 0; i < len(balances); i++ {
		balances[i] = make([]channel.Bal, 2)
		balances[i][myIdx] = myBals[i].Int
		balances[i][peerIdx] = peerBals[i].Int
	}
	return
}

// serialiseApp serialises the app to binary data and gives its type.
func serialiseApp(a channel.App) ([]byte, string, error) {
	if channel.IsNoApp(a) {
		json, err := json.Marshal(a)
		if err != nil {
			return nil, "", err
		}
		return json, "NoApp", nil
	}
	if reflect.TypeOf(a) == reflect.TypeOf(channel.NewMockApp(nil)) {
		appBinary, err := a.Def().MarshalBinary()
		if err != nil {
			return nil, "", err
		}
		return appBinary, "MockApp", nil
	}
	return nil, "", errors.New("App is not supported at this implementation")
}

// deserialiseApp will deserialises the data back to an app based on its type.
func deserialiseApp(def []byte, appType string) (channel.App, error) {
	if appType == "NoApp" {
		return channel.NoApp(), nil
	} else if appType == "MockApp" {
		appDef := wallet.NewAddress(1)
		err := appDef.UnmarshalBinary(def)
		if err != nil {
			return nil, err
		}
		appAddrBackend := appDef.(*ethwallet.Address)
		appID := &ethchannel.AppID{Address: appAddrBackend}
		app := channel.NewMockApp(appID)
		return app, nil
	} else {
		return nil, errors.New("This app's type is not supported.")
	}
}

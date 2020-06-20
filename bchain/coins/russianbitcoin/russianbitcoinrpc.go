package russianbitcoin

import (
	"encoding/json"

	"github.com/golang/glog"
	"github.com/nbcorg/blockbook/bchain"
	"github.com/nbcorg/blockbook/bchain/coins/btc"
)

// RussianBitcoinRPC is an interface to JSON-RPC bitcoind service.
type RussianBitcoinRPC struct {
	*btc.BitcoinRPC
}

// NewRussianBitcoinRPC returns new RussianBitcoinRPC instance.
func NewRussianBitcoinRPC(config json.RawMessage, pushHandler func(bchain.NotificationType)) (bchain.BlockChain, error) {
	b, err := btc.NewBitcoinRPC(config, pushHandler)
	if err != nil {
		return nil, err
	}

	s := &RussianBitcoinRPC{
		b.(*btc.BitcoinRPC),
	}
	s.RPCMarshaler = btc.JSONMarshalerV2{}
	s.ChainConfig.SupportsEstimateFee = false

	return s, nil
}

// Initialize initializes RussianBitcoinRPC instance.
func (b *RussianBitcoinRPC) Initialize() error {
	ci, err := b.GetChainInfo()
	if err != nil {
		return err
	}
	chainName := ci.Chain

	glog.Info("Chain name ", chainName)
	params := GetChainParams(chainName)

	// always create parser
	b.Parser = NewRussianBitcoinParser(params, b.ChainConfig)

	// parameters for getInfo request
	if params.Net == MainnetMagic {
		b.Testnet = false
		b.Network = "livenet"
	} else {
		b.Testnet = true
		b.Network = "testnet"
	}

	glog.Info("rpc: block chain ", params.Name)

	return nil
}

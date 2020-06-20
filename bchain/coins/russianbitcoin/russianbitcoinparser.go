package russianbitcoin

import (
	"github.com/nbcorg/btcd/wire"
	"github.com/nbcorg/btcutil/chaincfg"
	"github.com/nbcorg/blockbook/bchain/coins/btc"
)

// magic numbers
const (
	MainnetMagic wire.BitcoinNet = 0xd8b4bef8
	TestnetMagic wire.BitcoinNet = 0x0809110c
	RegtestMagic wire.BitcoinNet = 0xdbd5bffb
)

// chain parameters
var (
	MainNetParams chaincfg.Params
	TestNetParams chaincfg.Params
)

func init() {
	MainNetParams = chaincfg.MainNetParams
	MainNetParams.Net = MainnetMagic
	MainNetParams.PubKeyHashAddrID = []byte{102}
	MainNetParams.ScriptHashAddrID = []byte{11}
	MainNetParams.Bech32HRPSegwit = "rubtcm"

	TestNetParams = chaincfg.TestNet3Params
	TestNetParams.Net = TestnetMagic
	TestNetParams.PubKeyHashAddrID = []byte{105}
	TestNetParams.ScriptHashAddrID = []byte{13}
	TestNetParams.Bech32HRPSegwit = "rubtct"
}

// RussianBitcoinParser handle
type RussianBitcoinParser struct {
	*btc.BitcoinParser
}

// NewRussianBitcoinParser returns new RussianBitcoinParser instance
func NewRussianBitcoinParser(params *chaincfg.Params, c *btc.Configuration) *RussianBitcoinParser {
	return &RussianBitcoinParser{BitcoinParser: btc.NewBitcoinParser(params, c)}
}

// GetChainParams contains network parameters for the main RussianBitcoin network,
// and the test RussianBitcoin network
func GetChainParams(chain string) *chaincfg.Params {
	if !chaincfg.IsRegistered(&MainNetParams) {
		err := chaincfg.Register(&MainNetParams)
		if err == nil {
			err = chaincfg.Register(&TestNetParams)
		}
		if err != nil {
			panic(err)
		}
	}
	switch chain {
	case "test":
		return &TestNetParams
	default:
		return &MainNetParams
	}
}

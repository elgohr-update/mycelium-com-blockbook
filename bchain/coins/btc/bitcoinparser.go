package btc

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math/big"
	"strconv"

	vlq "github.com/bsm/go-vlq"
	"github.com/nbcorg/btcd/blockchain"
	"github.com/nbcorg/btcd/wire"
	"github.com/nbcorg/btcutil"
	"github.com/nbcorg/btcutil/chaincfg"
	"github.com/nbcorg/btcutil/txscript"
	"github.com/nbcorg/blockbook/bchain"
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
	RegtestParams chaincfg.Params
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

	RegtestParams = chaincfg.RegressionNetParams
	RegtestParams.Net = RegtestMagic
	RegtestParams.PubKeyHashAddrID = []byte{105}
	RegtestParams.ScriptHashAddrID = []byte{13}
	RegtestParams.Bech32HRPSegwit = "rubtct"
}

// OutputScriptToAddressesFunc converts ScriptPubKey to bitcoin addresses
type OutputScriptToAddressesFunc func(script []byte) ([]string, bool, error)

// BitcoinParser handle
type BitcoinParser struct {
	*bchain.BaseParser
	Params                       *chaincfg.Params
	OutputScriptToAddressesFunc  OutputScriptToAddressesFunc
	minimumCoinbaseConfirmations int
}

// NewBitcoinParser returns new BitcoinParser instance
func NewBitcoinParser(params *chaincfg.Params, c *Configuration) *BitcoinParser {
	p := &BitcoinParser{
		BaseParser: &bchain.BaseParser{
			BlockAddressesToKeep: c.BlockAddressesToKeep,
			AmountDecimalPoint:   8,
		},
		Params:                       params,
		minimumCoinbaseConfirmations: c.MinimumCoinbaseConfirmations,
	}
	p.OutputScriptToAddressesFunc = p.outputScriptToAddresses
	return p
}

// GetChainParams contains network parameters for the main Russian Bitcoin network,
// the regression test Russian Bitcoin network and the test Bitcoin network and
func GetChainParams(chain string) *chaincfg.Params {
	if !chaincfg.IsRegistered(&MainNetParams) {
		err := chaincfg.Register(&MainNetParams)
		if err == nil {
			err = chaincfg.Register(&TestNetParams)
		}
		if err == nil {
			err = chaincfg.Register(&RegtestParams)
		}
		if err != nil {
			panic(err)
		}
	}
	switch chain {
	case "test":
		return &TestNetParams
	case "regtest":
		return &RegtestParams
	}
	return &MainNetParams
}

// GetAddrDescFromVout returns internal address representation (descriptor) of given transaction output
func (p *BitcoinParser) GetAddrDescFromVout(output *bchain.Vout) (bchain.AddressDescriptor, error) {
	ad, err := hex.DecodeString(output.ScriptPubKey.Hex)
	if err != nil {
		return ad, err
	}
	// convert possible P2PK script to P2PKH
	// so that all transactions by given public key are indexed together
	return txscript.ConvertP2PKtoP2PKH(p.Params.Base58CksumHasher, ad)
}

// GetAddrDescFromAddress returns internal address representation (descriptor) of given address
func (p *BitcoinParser) GetAddrDescFromAddress(address string) (bchain.AddressDescriptor, error) {
	return p.addressToOutputScript(address)
}

// GetAddressesFromAddrDesc returns addresses for given address descriptor with flag if the addresses are searchable
func (p *BitcoinParser) GetAddressesFromAddrDesc(addrDesc bchain.AddressDescriptor) ([]string, bool, error) {
	return p.OutputScriptToAddressesFunc(addrDesc)
}

// GetScriptFromAddrDesc returns output script for given address descriptor
func (p *BitcoinParser) GetScriptFromAddrDesc(addrDesc bchain.AddressDescriptor) ([]byte, error) {
	return addrDesc, nil
}

// IsAddrDescIndexable returns true if AddressDescriptor should be added to index
// empty or OP_RETURN scripts are not indexed
func (p *BitcoinParser) IsAddrDescIndexable(addrDesc bchain.AddressDescriptor) bool {
	if len(addrDesc) == 0 || addrDesc[0] == txscript.OP_RETURN {
		return false
	}
	return true
}

// addressToOutputScript converts bitcoin address to ScriptPubKey
func (p *BitcoinParser) addressToOutputScript(address string) ([]byte, error) {
	da, err := btcutil.DecodeAddress(address, p.Params)
	if err != nil {
		return nil, err
	}
	script, err := txscript.PayToAddrScript(da)
	if err != nil {
		return nil, err
	}
	return script, nil
}

// TryParseOPReturn tries to process OP_RETURN script and return its string representation
func (p *BitcoinParser) TryParseOPReturn(script []byte) string {
	if len(script) > 1 && script[0] == txscript.OP_RETURN {
		// trying 2 variants of OP_RETURN data
		// 1) OP_RETURN OP_PUSHDATA1 <datalen> <data>
		// 2) OP_RETURN <datalen> <data>
		var data []byte
		var l int
		if script[1] == txscript.OP_PUSHDATA1 && len(script) > 2 {
			l = int(script[2])
			data = script[3:]
			if l != len(data) {
				l = int(script[1])
				data = script[2:]
			}
		} else {
			l = int(script[1])
			data = script[2:]
		}
		if l == len(data) {
			var ed string

			ed = p.tryParseOmni(data)
			if ed != "" {
				return ed
			}

			isASCII := true
			for _, c := range data {
				if c < 32 || c > 127 {
					isASCII = false
					break
				}
			}
			if isASCII {
				ed = "(" + string(data) + ")"
			} else {
				ed = hex.EncodeToString(data)
			}
			return "OP_RETURN " + ed
		}
	}
	return ""
}

var omniCurrencyMap = map[uint32]string{
	1:  "Omni",
	2:  "Test Omni",
	31: "TetherUS",
}

// tryParseOmni tries to extract Omni simple send transaction from script
func (p *BitcoinParser) tryParseOmni(data []byte) string {

	// currently only simple send transaction version 0 is supported, see
	// https://github.com/OmniLayer/spec#transfer-coins-simple-send
	if len(data) != 20 || data[0] != 'o' {
		return ""
	}
	// omni (4) <tx_version> (2) <tx_type> (2)
	omniHeader := []byte{'o', 'm', 'n', 'i', 0, 0, 0, 0}
	if bytes.Compare(data[0:8], omniHeader) != 0 {
		return ""
	}

	currencyID := binary.BigEndian.Uint32(data[8:12])
	currency, ok := omniCurrencyMap[currencyID]
	if !ok {
		return ""
	}
	amount := new(big.Int)
	amount.SetBytes(data[12:])
	amountStr := p.AmountToDecimalString(amount)

	ed := "OMNI Simple Send: " + amountStr + " " + currency + " (#" + strconv.Itoa(int(currencyID)) + ")"
	return ed
}

// outputScriptToAddresses converts ScriptPubKey to addresses with a flag that the addresses are searchable
func (p *BitcoinParser) outputScriptToAddresses(script []byte) ([]string, bool, error) {
	sc, addresses, _, err := txscript.ExtractPkScriptAddrs(script, p.Params)
	if err != nil {
		return nil, false, err
	}
	rv := make([]string, len(addresses))
	for i, a := range addresses {
		rv[i] = a.EncodeAddress()
	}
	var s bool
	if sc == txscript.PubKeyHashTy || sc == txscript.WitnessV0PubKeyHashTy || sc == txscript.ScriptHashTy || sc == txscript.WitnessV0ScriptHashTy {
		s = true
	} else if len(rv) == 0 {
		or := p.TryParseOPReturn(script)
		if or != "" {
			rv = []string{or}
		}
	}
	return rv, s, nil
}

// TxFromMsgTx converts bitcoin wire Tx to bchain.Tx
func (p *BitcoinParser) TxFromMsgTx(t *wire.MsgTx, parseAddresses bool) bchain.Tx {
	vin := make([]bchain.Vin, len(t.TxIn))
	for i, in := range t.TxIn {
		if blockchain.IsCoinBaseTx(t) {
			vin[i] = bchain.Vin{
				Coinbase: hex.EncodeToString(in.SignatureScript),
				Sequence: in.Sequence,
			}
			break
		}
		s := bchain.ScriptSig{
			Hex: hex.EncodeToString(in.SignatureScript),
			// missing: Asm,
		}
		vin[i] = bchain.Vin{
			Txid:      in.PreviousOutPoint.Hash.String(),
			Vout:      in.PreviousOutPoint.Index,
			Sequence:  in.Sequence,
			ScriptSig: s,
		}
	}
	vout := make([]bchain.Vout, len(t.TxOut))
	for i, out := range t.TxOut {
		addrs := []string{}
		if parseAddresses {
			addrs, _, _ = p.OutputScriptToAddressesFunc(out.PkScript)
		}
		s := bchain.ScriptPubKey{
			Hex:       hex.EncodeToString(out.PkScript),
			Addresses: addrs,
			// missing: Asm,
			// missing: Type,
		}
		var vs big.Int
		vs.SetInt64(out.Value)
		vout[i] = bchain.Vout{
			ValueSat:     vs,
			N:            uint32(i),
			ScriptPubKey: s,
		}
	}
	tx := bchain.Tx{
		Txid:     t.TxHash().String(),
		Version:  t.Version,
		LockTime: t.LockTime,
		Vin:      vin,
		Vout:     vout,
		// skip: BlockHash,
		// skip: Confirmations,
		// skip: Time,
		// skip: Blocktime,
	}
	return tx
}

// ParseTx parses byte array containing transaction and returns Tx struct
func (p *BitcoinParser) ParseTx(b []byte) (*bchain.Tx, error) {
	t := wire.MsgTx{}
	r := bytes.NewReader(b)
	if err := t.Deserialize(r); err != nil {
		return nil, err
	}
	tx := p.TxFromMsgTx(&t, true)
	tx.Hex = hex.EncodeToString(b)
	return &tx, nil
}

// ParseBlock parses raw block to our Block struct
func (p *BitcoinParser) ParseBlock(b []byte) (*bchain.Block, error) {
	w := wire.MsgBlock{}
	r := bytes.NewReader(b)

	if err := w.Deserialize(r); err != nil {
		return nil, err
	}

	txs := make([]bchain.Tx, len(w.Transactions))
	for ti, t := range w.Transactions {
		txs[ti] = p.TxFromMsgTx(t, false)
	}

	return &bchain.Block{
		BlockHeader: bchain.BlockHeader{
			Size: len(b),
			Time: w.Header.Timestamp.Unix(),
		},
		Txs: txs,
	}, nil
}

// PackTx packs transaction to byte array
func (p *BitcoinParser) PackTx(tx *bchain.Tx, height uint32, blockTime int64) ([]byte, error) {
	buf := make([]byte, 4+vlq.MaxLen64+len(tx.Hex)/2)
	binary.BigEndian.PutUint32(buf[0:4], height)
	vl := vlq.PutInt(buf[4:4+vlq.MaxLen64], blockTime)
	hl, err := hex.Decode(buf[4+vl:], []byte(tx.Hex))
	return buf[0 : 4+vl+hl], err
}

// UnpackTx unpacks transaction from byte array
func (p *BitcoinParser) UnpackTx(buf []byte) (*bchain.Tx, uint32, error) {
	height := binary.BigEndian.Uint32(buf)
	bt, l := vlq.Int(buf[4:])
	tx, err := p.ParseTx(buf[4+l:])
	if err != nil {
		return nil, 0, err
	}
	tx.Blocktime = bt

	return tx, height, nil
}

// MinimumCoinbaseConfirmations returns minimum number of confirmations a coinbase transaction must have before it can be spent
func (p *BitcoinParser) MinimumCoinbaseConfirmations() int {
	return p.minimumCoinbaseConfirmations
}

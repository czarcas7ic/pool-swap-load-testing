package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	cometrpc "github.com/cometbft/cometbft/rpc/client/http"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/osmosis-labs/osmosis/osmomath"
	poolmanagertypes "github.com/osmosis-labs/osmosis/v25/x/poolmanager/types"
	"github.com/tidwall/gjson"
)

var client = &http.Client{
	Timeout: 10 * time.Second, // Adjusted timeout to 10 seconds
	Transport: &http.Transport{
		MaxIdleConns:        100,              // Increased maximum idle connections
		MaxIdleConnsPerHost: 10,               // Increased maximum idle connections per host
		IdleConnTimeout:     90 * time.Second, // Increased idle connection timeout
		TLSHandshakeTimeout: 10 * time.Second, // Increased TLS handshake timeout
	},
}

// Memo represents the structure of the memo field in the transaction
type Memo struct {
	Forward Forward `json:"forward"`
}

// Forward contains details about the forwarding information
type Forward struct {
	Receiver string   `json:"receiver"`
	Port     string   `json:"port"`
	Channel  string   `json:"channel"`
	Timeout  string   `json:"timeout"`
	Retries  int      `json:"retries"`
	Next     *Forward `json:"next,omitempty"`
}

// ToJSON converts the Memo struct to a JSON string
func (m *Memo) ToJSON() (string, error) {
	bytes, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// NewMemo creates a new Memo struct with default values
func NewMemo(config Config) *Memo {
	return &Memo{
		Forward: Forward{
			Receiver: strings.Repeat(config.IBCMemo, config.IBCMemoRepeat), // Note: This is an invalid bech32 address
			Port:     "transfer",
			Channel:  "channel-569",
			Timeout:  "12h",
			Retries:  10,
		},
	}
}

var cdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

func init() {
	types.RegisterInterfaces(cdc.InterfaceRegistry())
}

func poolManagerSwapInViaRPC(rpcEndpoint string, chainID string, sequence, accnum uint64, privKey cryptotypes.PrivKey, pubKey cryptotypes.PubKey, address string, poolId uint64) (response *coretypes.ResultBroadcastTx, txbody string, err error) {
	encodingConfig := moduletestutil.MakeTestEncodingConfig()
	poolmanagertypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	// Create a new TxBuilder.
	txBuilder := encodingConfig.TxConfig.NewTxBuilder()

	nonOsmoDenom := getNonOsmoAssetFromPool(poolId)

	msg := &poolmanagertypes.MsgSwapExactAmountIn{
		Sender: address,
		Routes: []poolmanagertypes.SwapAmountInRoute{
			{
				PoolId:        poolId,
				TokenOutDenom: nonOsmoDenom,
			},
		},
		TokenIn:           sdk.NewInt64Coin("uosmo", 100000),
		TokenOutMinAmount: osmomath.OneInt(),
	}

	// set messages
	err = txBuilder.SetMsgs(msg)
	if err != nil {
		return nil, "", err
	}

	// Estimate gas limit based on transaction size
	txSize := msg.Size()
	gasLimit := uint64((txSize * GasPerByte) + BaseGas)
	txBuilder.SetGasLimit(gasLimit)

	// Calculate fee based on gas limit and a fixed gas price

	gasPrice := sdk.NewDecCoinFromDec(Denom, osmomath.NewDecWithPrec(GasLow, Precision)) // 0.00051 token per gas unit
	//	gasPrice := getGasPrice(config.Gas.Low, config.Denom)
	feeAmount := gasPrice.Amount.MulInt64(int64(gasLimit)).RoundInt()
	feecoin := sdk.NewCoin(Denom, feeAmount)
	txBuilder.SetFeeAmount(sdk.NewCoins(feecoin))
	txBuilder.SetTimeoutHeight(0)

	// First round: we gather all the signer infos. We use the "set empty
	// signature" hack to do that.
	sigV2 := signing.SignatureV2{
		PubKey: pubKey,
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode(encodingConfig.TxConfig.SignModeHandler().DefaultMode()),
			Signature: nil,
		},
		Sequence: sequence,
	}

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		fmt.Println("error setting signatures")
		return nil, "", err
	}

	signerData := authsigning.SignerData{
		ChainID:       chainID,
		AccountNumber: accnum,
		Sequence:      sequence,
	}

	signed, err := tx.SignWithPrivKey(
		context.Background(),
		signing.SignMode(encodingConfig.TxConfig.SignModeHandler().DefaultMode()), signerData,
		txBuilder, privKey, encodingConfig.TxConfig, sequence)
	if err != nil {
		fmt.Println("couldn't sign")
		return nil, "", err
	}

	err = txBuilder.SetSignatures(signed)
	if err != nil {
		return nil, "", err
	}

	// Generate a JSON string.
	txJSONBytes, err := encodingConfig.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		fmt.Println(err)
		return nil, "", err
	}

	resp, err := BroadcastTransaction(txJSONBytes, rpcEndpoint)
	if err != nil {
		return nil, "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	return resp, string(txJSONBytes), nil
}

func BroadcastTransaction(txBytes []byte, rpcEndpoint string) (*coretypes.ResultBroadcastTx, error) {
	cmtCli, err := cometrpc.New(rpcEndpoint, "/websocket")
	if err != nil {
		log.Fatal(err)
	}

	t := tmtypes.Tx(txBytes)

	ctx := context.Background()
	res, err := cmtCli.BroadcastTxSync(ctx, t)
	if err != nil {
		fmt.Println(err)
		fmt.Println("error at broadcast")
		return nil, err
	}

	// fmt.Println("other: ", res.Data)
	// fmt.Println("log: ", res.Log)
	// fmt.Println("code: ", res.Code)
	// fmt.Println("code: ", res.Codespace)
	// fmt.Println("txid: ", res.Hash)

	return res, nil
}

func getNonOsmoAssetFromPool(poolID uint64) string {
	url := fmt.Sprintf("%s/osmosis/poolmanager/v1beta1/pools/%d/total_pool_liquidity", LCDURL, poolID)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Failed to get pool info: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	nonOsmoDenom := gjson.Get(string(body), `liquidity.#(denom!="uosmo").denom`).String()
	return nonOsmoDenom
}

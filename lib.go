package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func getInitialSequence(address string, config Config) (int64, int64) {
	resp, err := httpGet(config.LcdUrl + "/cosmos/auth/v1beta1/accounts/" + address)
	if err != nil {
		log.Printf("Failed to get initial sequence: %v", err)
		return 0, 0
	}

	var accountRes AccountResult
	err = json.Unmarshal(resp, &accountRes)
	if err != nil {
		log.Printf("Failed to unmarshal account result: %v", err)
		return 0, 0
	}

	seqint, err := strconv.ParseInt(accountRes.Account.Sequence, 10, 64)
	if err != nil {
		log.Printf("Failed to convert sequence to int: %v", err)
		return 0, 0
	}

	accnum, err := strconv.ParseInt(accountRes.Account.AccountNumber, 10, 64)
	if err != nil {
		log.Printf("Failed to convert account number to int: %v", err)
		return 0, 0
	}

	return seqint, accnum
}

func getChainID(nodeURL string) (string, error) {
	resp, err := httpGet(nodeURL + "/status")
	if err != nil {
		log.Printf("Failed to get node status: %v", err)
		return "", err
	}

	var statusRes NodeStatusResponse
	err = json.Unmarshal(resp, &statusRes)
	if err != nil {
		log.Printf("Failed to unmarshal node status result: %v", err)
		return "", err
	}

	return statusRes.Result.NodeInfo.Network, nil
}

func httpGet(url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		netErr, ok := err.(net.Error)
		if ok && netErr.Timeout() {
			log.Printf("Request to %s timed out, continuing...", url)
			return nil, nil
		}
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func getPrivKey(mnemonic string) (cryptotypes.PrivKey, cryptotypes.PubKey, string) {
	// Generate a Bip32 HD wallet for the mnemonic and a user supplied password
	// create master key and derive first key for keyring
	algo := hd.Secp256k1

	// Derive the first key for keyring
	// NOTE: this function had a bug, it was set to 118, then to 330.
	// it is now configurable in the config file, to prevent this problem
	derivedPriv, err := algo.Derive()(mnemonic, "", fmt.Sprintf("m/44'/%d'/0'/0/0", 118))
	if err != nil {
		panic(err)
	}

	privKey := algo.Generate()(derivedPriv)

	// Create master private key from

	pubKey := privKey.PubKey()

	// Convert the public key to Bech32 with custom HRP
	// bech32PubKey, err := bech32ifyPubKeyWithCustomHRP("celestia", pubKey)
	// if err != nil {
	//	panic(err)
	// }

	addressbytes := sdk.AccAddress(pubKey.Address().Bytes())
	address, err := sdk.Bech32ifyAddressBytes("osmo", addressbytes)
	if err != nil {
		panic(err)
	}

	fmt.Println("Seed provided translates to the following address: ", address)

	return privKey, pubKey, address
}

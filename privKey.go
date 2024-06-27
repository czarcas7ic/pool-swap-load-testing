package main

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

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

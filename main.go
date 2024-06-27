package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/tidwall/gjson"
)

var (
	AllPoolIds = []int{}
)

func main() {
	// tracking vars
	var successfulTxns int
	var failedTxns int
	var allTxHashes []string

	// Declare a map to hold response codes and their counts
	responseCodes := make(map[uint32]int)

	// Read in config file
	config := readInConfig()

	// keyring
	// read seed phrase
	privkey, pubKey, acctaddress := getPrivKey(config.Mnemonic)

	fmt.Printf("Using rpc URL: %s\n", config.RpcUrl)
	fmt.Printf("Using lcd URL: %s\n", config.LcdUrl)
	fmt.Println()

	// get correct chain-id
	chainID, err := getChainID(config.RpcUrl)
	if err != nil {
		log.Fatalf("Failed to get chain ID: %v", err)
	}

	// Compile the regex outside the loop
	reMismatch := regexp.MustCompile("account sequence mismatch")
	reExpected := regexp.MustCompile(`expected (\d+)`)

	// Get the account number (accNum) once
	seqNum, accNum := getInitialSequence(acctaddress, config)

	swapOnPool := func(poolID int) string {
		for {
			resp, _, err := poolManagerSwapInViaRPC(config.RpcUrl, chainID, uint64(seqNum), uint64(accNum), privkey, pubKey, acctaddress, uint64(poolID), config)
			if err != nil {
				failedTxns++
				fmt.Printf("%s Node: %s, Error: %v\n", time.Now().Format("15:04:05"), config.RpcUrl, err)
				return ""
			} else {
				successfulTxns++
				if resp != nil {
					// Increment the count for this response code
					responseCodes[resp.Code]++
				}

				match := reMismatch.MatchString(resp.Log)
				if match {
					matches := reExpected.FindStringSubmatch(resp.Log)
					if len(matches) > 1 {
						newSequence, err := strconv.ParseInt(matches[1], 10, 64)
						if err != nil {
							log.Fatalf("Failed to convert sequence to integer: %v", err)
						}
						// Update the per-node sequence to the expected value
						seqNum = newSequence
					}
				} else {
					// Increment the per-node sequence number if there was no mismatch
					seqNum++
				}
				return resp.Hash.String()
			}
		}
	}

	// Iterate over AllPoolIds and send transactions in rounds
	for i := 0; i < len(AllPoolIds); i++ {
		waitForNextBlock(acctaddress, config)
		var roundTxHashes []string // To store tx hashes for the current round
		for j := 0; j <= i; j++ {
			txHash := swapOnPool(AllPoolIds[j])
			if txHash != "" {
				roundTxHashes = append(roundTxHashes, txHash)
				allTxHashes = append(allTxHashes, txHash)
			}
		}
		// Report block height and tx hashes for the current round
		currentHeight := retrieveStatus(config)
		fmt.Printf("Round %d completed at block height %d\n", i+1, currentHeight)
		fmt.Printf("Successful transaction submissions for this round (%d)\n", len(roundTxHashes))
	}

	fmt.Println()
	fmt.Println("Total code 0 transactions at submission time: ", successfulTxns)
	fmt.Println("Total non code 0 transactions at submission time: ", failedTxns)
	fmt.Println()
	totalTxns := successfulTxns + failedTxns
	fmt.Println("Response code breakdown:")
	for code, count := range responseCodes {
		percentage := float64(count) / float64(totalTxns) * 100
		fmt.Printf("Code %d: %d (%.2f%%)\n", code, count, percentage)
	}
	fmt.Println()

	// Query each transaction hash at the end
	var failedTxHashes []string
	for _, hash := range allTxHashes {
		url := fmt.Sprintf("%s/tx?hash=0x%s", config.RpcUrl, hash)
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Failed to query tx hash %s: %v", hash, err)
			continue
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read response body for tx hash %s: %v", hash, err)
			continue
		}

		code := gjson.Get(string(body), "result.tx_result.code").Int()
		if code != 0 {
			failedTxHashes = append(failedTxHashes, hash)
		}
	}

	// Report failed transaction hashes
	fmt.Printf("After querying all tx hashes POST submission, the following txs actually failed (%d):\n", len(failedTxHashes))
	for _, hash := range failedTxHashes {
		fmt.Println(hash)
	}
}

func setSequence(acctaddress string, config Config) int64 {
	url := fmt.Sprintf("%s/cosmos/auth/v1beta1/account_info/%s", config.LcdUrl, acctaddress)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Failed to get account info: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	sequence := gjson.Get(string(body), "account.value.sequence").Int()
	return sequence
}

func retrieveStatus(config Config) int64 {
	url := fmt.Sprintf("%s/status", config.RpcUrl)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Failed to get status: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	latestBlockHeight := gjson.Get(string(body), "result.sync_info.latest_block_height").Int()
	return latestBlockHeight
}

func waitForNextBlock(acctaddress string, config Config) {
	initialHeight := int64(0)
	for initialHeight == 0 {
		initialHeight = retrieveStatus(config)
		time.Sleep(50 * time.Millisecond)
	}

	targetHeight := initialHeight + 1
	currentHeight := initialHeight
	for currentHeight < targetHeight {
		currentHeight = retrieveStatus(config)
		time.Sleep(50 * time.Millisecond)
	}

	setSequence(acctaddress, config)
}

func readInConfig() Config {
	// Default values
	config := Config{
		OsmoGammPoolIds: []int{1, 712, 704, 812, 678, 681, 796, 1057, 3, 9, 725, 832, 806, 840, 1241, 1687, 1632, 722, 584, 560, 586, 5, 604, 497, 992, 799, 1244, 744, 1075, 1225},                                // 30 pools
		OsmoClPoolIds:   []int{1252, 1135, 1093, 1134, 1090, 1133, 1248, 1323, 1094, 1095, 1263, 1590, 1096, 1265, 1098, 1097, 1092, 1464, 1400, 1388, 1104, 1325, 1281, 1114, 1066, 1215, 1449, 1077, 1399, 1770}, // 30 pools
		OsmoCwPoolIds:   []int{1463, 1575, 1584, 1642, 1643},
		Mnemonic:        "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius",
		RpcUrl:          "http://localhost:26657",
		LcdUrl:          "http://localhost:1317",
		GasPerByte:      20,
		BaseGas:         710000,
		Denom:           "uosmo",
		GasLow:          25,
		Precision:       4,
	}

	// Read config file
	configFile, err := os.Open("config.json")
	if err != nil {
		log.Printf("Failed to open config file, using default values: %v", err)
	} else {
		defer configFile.Close()
		byteValue, _ := ioutil.ReadAll(configFile)
		json.Unmarshal(byteValue, &config)
	}

	// Combine all pool IDs
	AllPoolIds = append(config.OsmoGammPoolIds, append(config.OsmoClPoolIds, config.OsmoCwPoolIds...)...)

	fmt.Println("Using the following configuration:")
	fmt.Println(config)

	return config
}

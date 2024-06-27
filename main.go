package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/tidwall/gjson"
)

const (
	BatchSize  = 100
	MaxWorkers = 10000
)

var (
	OsmoGammPoolIds = []int{1, 712, 704, 812, 678, 681, 796, 1057, 3, 9, 725, 832, 806, 840, 1241, 1687, 1632, 722, 584, 560, 586, 5, 604, 497, 992, 799, 1244, 744, 1075, 1225}                                // 30 pools
	OsmoClPoolIds   = []int{1252, 1135, 1093, 1134, 1090, 1133, 1248, 1323, 1094, 1095, 1263, 1590, 1096, 1265, 1098, 1097, 1092, 1464, 1400, 1388, 1104, 1325, 1281, 1114, 1066, 1215, 1449, 1077, 1399, 1770} // 30 pools
	OsmoCwPoolIds   = []int{1463, 1575, 1584, 1642, 1643}
	// OsmoGammPoolIds = []int{1, 712, 704, 812, 678}
	// OsmoClPoolIds   = []int{1252} // 30 pools
	// OsmoCwPoolIds   = []int{1463} // 5 pools
	AllPoolIds = append(OsmoGammPoolIds, append(OsmoClPoolIds, OsmoCwPoolIds...)...)
	Mnemonic   = []byte("notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius") // lo-test2
	RPCURL     = "http://localhost:26657"
	LCDURL     = "http://localhost:1317"
	GasPerByte = 20
	BaseGas    = 300000
	Denom      = "uosmo"
	GasLow     = int64(25)
	Precision  = int64(4)
)

func main() {
	// tracking vars
	var successfulTxns int
	var failedTxns int
	var mu sync.Mutex
	// Declare a map to hold response codes and their counts
	responseCodes := make(map[uint32]int)
	var allTxHashes []string

	// keyring
	// read seed phrase
	// mnemonic, _ := os.ReadFile("seedphrase")
	privkey, pubKey, acctaddress := getPrivKey(Mnemonic)
	// Create an in-memory keyring

	fmt.Printf("Using rpc URL: %s\n", RPCURL)
	fmt.Println("Using lcd URL: ", LCDURL)

	// get correct chain-id
	chainID, err := getChainID(RPCURL)
	if err != nil {
		log.Fatalf("Failed to get chain ID: %v", err)
	}

	// Compile the regex outside the loop
	reMismatch := regexp.MustCompile("account sequence mismatch")
	reExpected := regexp.MustCompile(`expected (\d+)`)

	// Get the account number (accNum) once
	seqNum, accNum := getInitialSequence(acctaddress)

	swapOnPool := func(poolID int) string {
		for {
			resp, _, err := poolManagerSwapInViaRPC(RPCURL, chainID, uint64(seqNum), uint64(accNum), privkey, pubKey, acctaddress, uint64(poolID))
			if err != nil {
				mu.Lock()
				failedTxns++
				mu.Unlock()
				fmt.Printf("%s Node: %s, Error: %v\n", time.Now().Format("15:04:05"), RPCURL, err)
				return ""
			} else {
				mu.Lock()
				successfulTxns++
				mu.Unlock()
				if resp != nil {
					// Increment the count for this response code
					mu.Lock()
					responseCodes[resp.Code]++
					mu.Unlock()
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
						fmt.Printf("%s Node: %s, we had an account sequence mismatch, adjusting to %d\n", time.Now().Format("15:04:05"), RPCURL, newSequence)
						// Retry the transaction with the new sequence number
						continue
					}
				} else {
					// Increment the per-node sequence number if there was no mismatch
					seqNum++
					//fmt.Printf("%s Node: %s, sequence: %d\n", time.Now().Format("15:04:05"), RPCURL, seqNum)
				}
				return resp.Hash.String()
			}
		}
	}

	// Iterate over AllPoolIds and send transactions in rounds
	for i := 0; i < len(AllPoolIds); i++ {
		waitForNextBlock(acctaddress)
		var roundTxHashes []string // To store tx hashes for the current round
		for j := 0; j <= i; j++ {
			txHash := swapOnPool(AllPoolIds[j])
			if txHash != "" {
				roundTxHashes = append(roundTxHashes, txHash)
				allTxHashes = append(allTxHashes, txHash)
			}
		}
		// Report block height and tx hashes for the current round
		currentHeight := retrieveStatus()
		fmt.Printf("Round %d completed at block height %d\n", i+1, currentHeight)
		fmt.Printf("Successful transaction submissions for this round (%d):", len(roundTxHashes))
		// for _, hash := range roundTxHashes {
		// 	fmt.Println(hash)
		// }
		// fmt.Println()
	}

	// Query each transaction hash at the end
	var failedTxHashes []string
	for _, hash := range allTxHashes {
		url := fmt.Sprintf("%s/tx?hash=0x%s", RPCURL, hash)
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

	fmt.Println("successful transactions: ", successfulTxns)
	fmt.Println("failed transactions: ", failedTxns)
	totalTxns := successfulTxns + failedTxns
	fmt.Println("Response code breakdown:")
	for code, count := range responseCodes {
		percentage := float64(count) / float64(totalTxns) * 100
		fmt.Printf("Code %d: %d (%.2f%%)\n", code, count, percentage)
	}

	// Report failed transaction hashes
	fmt.Printf("Failed transaction hashes (%d):\n", len(failedTxHashes))
	for _, hash := range failedTxHashes {
		fmt.Println(hash)
	}
}

func setSequence(acctaddress string) int64 {
	url := fmt.Sprintf("%s/cosmos/auth/v1beta1/account_info/%s", LCDURL, acctaddress)
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

func retrieveStatus() int64 {
	url := fmt.Sprintf("%s/status", RPCURL)
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

func waitForNextBlock(acctaddress string) {
	initialHeight := int64(0)
	for initialHeight == 0 {
		initialHeight = retrieveStatus()
		time.Sleep(50 * time.Millisecond)
	}

	targetHeight := initialHeight + 1
	currentHeight := initialHeight
	for currentHeight < targetHeight {
		currentHeight = retrieveStatus()
		time.Sleep(50 * time.Millisecond)
	}

	setSequence(acctaddress)
}

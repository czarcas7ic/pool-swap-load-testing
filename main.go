package main

import (
	"encoding/json"
	"fmt"
	"io"
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
	var (
		successfulTxns int
		failedTxns     int
		allTxHashes    []string
		responseCodes  = make(map[uint32]int)
	)

	config := readInConfig()
	privkey, pubKey, acctaddress := getPrivKey(config.Mnemonic)
	fmt.Println()

	chainID, err := getChainID(config.RpcUrl)
	if err != nil {
		log.Fatalf("Failed to get chain ID: %v", err)
	}

	reMismatch := regexp.MustCompile("account sequence mismatch")
	reExpected := regexp.MustCompile(`expected (\d+)`)

	seqNum, accNum := getInitialSequence(acctaddress, config)

	swapOnPool := func(poolID int) string {

		resp, _, err := poolManagerSwapInViaRPC(config.RpcUrl, chainID, uint64(seqNum), uint64(accNum), privkey, pubKey, acctaddress, uint64(poolID), config)
		if err != nil {
			failedTxns++
			fmt.Printf("%s Node: %s, Error: %v\n", time.Now().Format("15:04:05"), config.RpcUrl, err)
			return ""
		}
		successfulTxns++
		if resp != nil {
			responseCodes[resp.Code]++
		}

		if reMismatch.MatchString(resp.Log) {
			matches := reExpected.FindStringSubmatch(resp.Log)
			if len(matches) > 1 {
				newSequence, err := strconv.ParseInt(matches[1], 10, 64)
				if err != nil {
					log.Fatalf("Failed to convert sequence to integer: %v", err)
				}
				seqNum = newSequence
			}
		} else {
			seqNum++
		}
		return resp.Hash.String()
	}

	for i := 0; i < len(AllPoolIds); i++ {
		waitForNextBlock(acctaddress, config)
		var roundTxHashes []string
		for j := 0; j <= i; j++ {
			txHash := swapOnPool(AllPoolIds[j])
			if txHash != "" {
				roundTxHashes = append(roundTxHashes, txHash)
				allTxHashes = append(allTxHashes, txHash)
			}
		}
		currentHeight := retrieveStatus(config)
		fmt.Printf("Round %d completed at block height %d\n", i+1, currentHeight)
		fmt.Printf("Successful transaction submissions for this round (%d)\n", len(roundTxHashes))
	}

	printSummary(successfulTxns, failedTxns, responseCodes, allTxHashes, config)
}

func printSummary(successfulTxns, failedTxns int, responseCodes map[uint32]int, allTxHashes []string, config Config) {
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

	var failedTxHashes []string
	for _, hash := range allTxHashes {
		url := fmt.Sprintf("%s/tx?hash=0x%s", config.RpcUrl, hash)
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Failed to query tx hash %s: %v", hash, err)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read response body for tx hash %s: %v", hash, err)
			continue
		}

		code := gjson.Get(string(body), "result.tx_result.code").Int()
		if code != 0 {
			failedTxHashes = append(failedTxHashes, hash)
		}
	}

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

	body, err := io.ReadAll(resp.Body)
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

	body, err := io.ReadAll(resp.Body)
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
	config := Config{
		OsmoGammPoolIds: []int{1, 712, 704, 812, 678, 681, 796, 1057, 3, 9,
			725, 832, 806, 840, 1241, 1687, 1632, 722, 584, 560,
			586, 5, 604, 497, 992, 799, 1244, 744, 1075, 1225,
			2, 1020, 789, 816, 674, 608, 1036, 1226, 899, 907,
			605, 1738, 1827, 571, 626, 1320, 1046, 602, 481, 42,
			15, 800, 777, 7, 924, 648, 1173, 900, 597, 1408,
			627, 1249, 773, 601, 625, 651, 573, 641, 577, 644}, // 70 pools
		OsmoClPoolIds: []int{1252, 1135, 1093, 1134, 1090, 1133, 1248, 1323, 1094, 1095,
			1263, 1590, 1096, 1265, 1098, 1097, 1092, 1464, 1400, 1388,
			1104, 1325, 1281, 1114, 1066, 1215, 1449, 1077, 1399, 1770,
			1110, 1750, 1111, 1361, 1670, 1221, 1623, 1101, 1088, 1245,
			1105, 1779, 1434, 1477, 1483, 1620, 1100, 1091, 1108, 1109}, // 50 pools
		OsmoCwPoolIds: []int{1616, 1635, 1461, 1514, 1643, 1642, 1463, 1584}, // 9 pools
		// OsmoCwPoolIds: []int{1463, 1575, 1584, 1642, 1643},
		Mnemonic:   "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius",
		RpcUrl:     "http://localhost:26657",
		LcdUrl:     "http://localhost:1317",
		GasPerByte: 20,
		BaseGas:    710000,
		Denom:      "uosmo",
		GasLow:     25,
		Precision:  4,
	}

	configFile, err := os.Open("config.json")
	if err != nil {
		log.Printf("Failed to open config file, using default values: %v", err)
	} else {
		defer configFile.Close()
		byteValue, _ := io.ReadAll(configFile)
		json.Unmarshal(byteValue, &config)
	}

	AllPoolIds = append(config.OsmoGammPoolIds, append(config.OsmoClPoolIds, config.OsmoCwPoolIds...)...)

	fmt.Println("Using the following configuration (if value wasn't provided, defaults are used):")
	fmt.Println("OsmoGammPoolIds:", config.OsmoGammPoolIds)
	fmt.Println("OsmoClPoolIds:", config.OsmoClPoolIds)
	fmt.Println("OsmoCwPoolIds:", config.OsmoCwPoolIds)
	fmt.Println("Mnemonic:", config.Mnemonic)
	fmt.Println("RpcUrl:", config.RpcUrl)
	fmt.Println("LcdUrl:", config.LcdUrl)
	fmt.Println("GasPerByte:", config.GasPerByte)
	fmt.Println("BaseGas:", config.BaseGas)
	fmt.Println("Denom:", config.Denom)
	fmt.Println("GasLow:", config.GasLow)
	fmt.Println("Precision:", config.Precision)

	return config
}

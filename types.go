package main

type Header struct {
	Height string `json:"height"`
}

type Data struct {
	Txs []string `json:"txs"`
}

type Block struct {
	Header Header `json:"header"`
	Data   Data   `json:"data"`
}

type ResultBlock struct {
	Block Block `json:"block"`
}

type BlockResult struct {
	Result ResultBlock `json:"result"`
}

type MempoolResult struct {
	Result Result `json:"result"`
}

type BroadcastRequest struct {
	Jsonrpc                string `json:"jsonrpc"`
	ID                     string `json:"id"`
	Method                 string `json:"method"`
	BroadcastRequestParams `json:"params"`
}

type BroadcastRequestParams struct {
	Tx string `json:"tx"`
}

type BroadcastResponse struct {
	Jsonrpc         string `json:"jsonrpc"`
	ID              string `json:"id"`
	BroadcastResult `json:"result"`
}

type BroadcastResult struct {
	Code      int    `json:"code"`
	Data      string `json:"data"`
	Log       string `json:"log"`
	Codespace string `json:"codespace"`
	Hash      string `json:"hash"`
}

type Result struct {
	NTxs       string      `json:"n_txs"`
	Total      string      `json:"total"`
	TotalBytes string      `json:"total_bytes"`
	Txs        interface{} `json:"txs"` // Assuming txs can be null or an array, interface{} will accommodate both
}

type AccountInfo struct {
	Sequence      string `json:"sequence"`
	AccountNumber string `json:"account_number"`
}

type AccountResult struct {
	Account AccountInfo `json:"account"`
}

type Transaction struct {
	Body       Body     `json:"body"`
	AuthInfo   AuthInfo `json:"auth_info"`
	Signatures []string `json:"signatures"`
}

type Body struct {
	Messages                    []Message `json:"messages"`
	Memo                        string    `json:"memo"`
	TimeoutHeight               string    `json:"timeout_height"`
	ExtensionOptions            []string  `json:"extension_options"`
	NonCriticalExtensionOptions []string  `json:"non_critical_extension_options"`
}

type Message struct {
	Type             string        `json:"@type"`
	SourcePort       string        `json:"source_port"`
	SourceChannel    string        `json:"source_channel"`
	Token            Token         `json:"token"`
	Sender           string        `json:"sender"`
	Receiver         string        `json:"receiver"`
	TimeoutHeight    TimeoutHeight `json:"timeout_height"`
	TimeoutTimestamp string        `json:"timeout_timestamp"`
	Memo             string        `json:"memo"`
}

type Token struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

type TimeoutHeight struct {
	RevisionNumber string `json:"revision_number"`
	RevisionHeight string `json:"revision_height"`
}

type AuthInfo struct {
	SignerInfos []interface{} `json:"signer_infos"`
	Fee         Fee           `json:"fee"`
}

type Fee struct {
	Amount   []Token `json:"amount"`
	GasLimit string  `json:"gas_limit"`
	Payer    string  `json:"payer"`
	Granter  string  `json:"granter"`
}

type Config struct {
	Chain          string `toml:"chain"`
	Channel        string `toml:"channel"`
	Prefix         string `toml:"prefix"`
	Bytes          int    `toml:"gas_per_byte"`
	RevisionNumber int    `toml:"revision_number"`
	TimeoutHeight  int    `toml:"timeout_height"`
	RandMin        int    `toml:"rand_min"`
	RandMax        int    `toml:"rand_max"`
	Memo           string `toml:"memo"`
	IBCMemo        string `toml:"ibc_memo"`
	IBCMemoRepeat  int    `toml:"ibc_memo_repeat"`
	IBCChannel     string `toml:"ibc_channel"`
	BaseGas        int    `toml:"base_gas"`
	Denom          string `toml:"denom"`
	Slip44         int    `toml:"slip44"`
	Gas            struct {
		Low       int64 `toml:"low"`
		Precision int64 `toml:"precision"`
	} `toml:"gas"`
	Nodes struct {
		RPC []string `toml:"rpc"`
		API string   `toml:"api"`
	} `toml:"nodes"`
}

type NodeInfo struct {
	Network string `json:"network"`
}

type NodeStatusResult struct {
	NodeInfo NodeInfo `json:"node_info"`
}

type NodeStatusResponse struct {
	Result NodeStatusResult `json:"result"`
}

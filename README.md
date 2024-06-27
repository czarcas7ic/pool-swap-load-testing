# Pool Swap Load Tester

Forked from the hardhat repo by somatic-labs, the pool swap load tester is used to simulate pool swaps on a varying number of pools each block. This tool was specifically created to test the ingress of pool changes for SQS, but can be used for any feature that could be effected by a large number of pools changing state each block.

The way this tool works is as follows:

Within the config.json (shown later), a list of pool IDs that contain OSMO as one of the assets in the pool are provided.
The idea is, each block, we swap over one more pool, until we have swapped over all pools.
Ex. If we have 2 GAMM pools, 2 CL pools, and 1 CW pool:
Block 1: Swap over GAMM pool 1
Block 2: Swap over GAMM pool 1, GAMM pool 2
Block 3: Swap over GAMM pool 1, GAMM pool 2, CL pool 1
Block 4: Swap over GAMM pool 1, GAMM pool 2, CL pool 1, CL pool 2
Block 5: Swap over GAMM pool 1, GAMM pool 2, CL pool 1, CL pool 2, CW pool 1

## How To Use

1. Download a mainnet snapshot
2. Run the in-place-testnet command (at the time of this writing, use the trigger upgrade flag as "v26" since this script is meant for a sdk v50 chain)
3. Hit the upgrade height and upgrade
4. Change your tx indexer setting in config.toml from "null" to "kv" and restart the node
5. While the node is running, cd into the pool-swap-load-tester directory
6. Run `go install ./...`
7. Change any desired settings in the config.json file
8. Run with the `poolswaps` command

## Configuration

An example config.json

```json
{
    "OsmoGammPoolIds": [
        1, 712, 704, 812, 678, 681, 796, 1057, 3, 9,
        725, 832, 806, 840, 1241, 1687, 1632, 722, 584, 560,
        586, 5, 604, 497, 992, 799, 1244, 744, 1075, 1225,
        2, 1020, 789, 816, 674, 608, 1036, 1226, 899, 907,
        605, 1738, 1827, 571, 626, 1320, 1046, 602, 481, 42,
        15, 800, 777, 7, 924, 648, 1173, 900, 597, 1408,
        627, 1249, 773, 601, 625, 651, 573, 641, 577, 644
    ],
    "OsmoClPoolIds": [
        1252, 1135, 1093, 1134, 1090, 1133, 1248, 1323, 1094, 1095,
        1263, 1590, 1096, 1265, 1098, 1097, 1092, 1464, 1400, 1388,
        1104, 1325, 1281, 1114, 1066, 1215, 1449, 1077, 1399, 1770,
        1110, 1750, 1111, 1361, 1670, 1221, 1623, 1101, 1088, 1245,
        1105, 1779, 1434, 1477, 1483, 1620, 1100, 1091, 1108, 1109
    ],
    "OsmoCwPoolIds": [
        1616, 1635, 1461, 1643, 1642, 1463, 1584
    ],
    "Mnemonic": "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius",
    "RpcUrl": "http://localhost:26657",
    "LcdUrl": "http://localhost:1317",
    "GasPerByte": 20,
    "BaseGas": 710000,
    "Denom": "uosmo",
    "FeeAmount": 100000
}
```

The first section are the pool ID definitions:

OsmoGammPoolIds: A list of gamm pool IDs that contain OSMO
OsmoClPoolIds: A list of cl pool IDs that contain OSMO
OsmoCwPoolIds: A list of cw pool IDs that contain OSMO

No special logic is done differently between these pool types, its just separated for convenience to note which pool types are being swapped against.

Mnemonic: The mnemonic of the account that will be used to sign the transactions. The provided mnemonic is the lo-test2 account from `make localnet-keys`. This account is seeded with lots of OSMO from in-place-testnet. There is no need to add this key to your keyring, it is used in memory.

RpcUrl: The RPC URL of the chain you are testing against.
LcdUrl: The LCD URL of the chain you are testing against.

GasPerByte: The gas price per byte for the transactions.
BaseGas: The base gas amount for the transactions.
Denom: The denomination of the fee amount.
FeeAmount: The fee amount for the transactions.

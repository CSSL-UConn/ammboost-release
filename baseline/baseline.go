package main

import (
	"ammcore"
	"context"
	"crypto/ecdsa"
	"encoding/csv"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/chainBoostScale/ChainBoost/MainAndSideChain/blockchain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rocketlaunchr/dataframe-go"
	"github.com/rocketlaunchr/dataframe-go/imports"
)

var positionMap map[string]*big.Int
var mapMutex sync.Mutex
var csvMutex sync.Mutex
var csvOutputFile *os.File
var csvWriter *csv.Writer
var users, _ = blockchain.GetUsers()
var upperGasLimit *big.Int
var sleepDuration int

func RunBaseline(batchSize int) {
	file, err := os.OpenFile("./data.csv", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}

	reader := csv.NewReader(file)
	existingData, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}
	amount := new(big.Int).Mul(big.NewInt(10), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	initMintRow := []string{"0x77b0d87f1276b5bacc762de11846d68358e38310", "mint", amount.String(), amount.String(), "1"}
	newData := append([][]string{initMintRow}, existingData...)
	file.Seek(0, 0)
	writer := csv.NewWriter(file)
	err = writer.WriteAll(newData)
	if err != nil {
		panic(err)
	}
	headerRow := []string{"sidechainADDR", "txType", "arg1", "arg2", "round"}
	newData = append([][]string{headerRow}, newData...)
	file.Seek(0, 0)
	err = writer.WriteAll(newData)
	if err != nil {
		panic(err)
	}
	writer.Flush()
	file.Close()

	file, err = os.Open("./data.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	df, err := imports.LoadFromCSV(context.Background(), file)
	if err != nil {
		fmt.Println(err)
	}

	iterator := df.ValuesIterator(dataframe.ValuesOptions{0, 1, true})
	var batchCounter = 0
	var batch []map[interface{}]interface{}
	for {
		if batchCounter >= batchSize {
			var wg sync.WaitGroup
			wg.Add(batchSize)
			for _, row := range batch {
				go func(row map[interface{}]interface{}) {
					defer wg.Done()
					//fmt.Println(row["sidechainADDR"], row["txType"], row["arg1"], row["arg2"], row["round"])
					CallFuncs(row)
				}(row)
			}
			wg.Wait()
			time.Sleep(1 * time.Second)

			batchCounter = 0
			batch = make([]map[interface{}]interface{}, 0)
		}
		row, vals, _ := iterator()
		if row == nil {
			break
		}
		batch = append(batch, vals)
		batchCounter += 1
	}

}

func CallFuncs(row map[interface{}]interface{}) {
	var privateKey *ecdsa.PrivateKey
	var address common.Address
	for _, userinfo := range users {
		if userinfo.AmmAddress == row["sidechainADDR"].(string) {
			privateKey = userinfo.PrivKey
			address = crypto.PubkeyToAddress(*userinfo.Pubkey)
		}
	}
	if row["txType"] == "mint" {
		fmt.Println("Checking gas price")
		ammcore.CheckCurrentGasPrice(upperGasLimit, sleepDuration)
		fmt.Println("Minting")
		amount0, _ := new(big.Int).SetString(row["arg1"].(string), 10)
		amount1, _ := new(big.Int).SetString(row["arg2"].(string), 10)
		var err error
		startTime := time.Now()
		nonce := ammcore.GetNonceFromSyncMap(address)
		reciept0, err := ammcore.CallApproveABTX(nonce, privateKey, amount0)
		if err != nil {
			data := []string{err.Error()}
			WriteToCsv(data)
			return
		}
		nonce = ammcore.GetNonceFromSyncMap(address)
		reciept1, err := ammcore.CallApproveABTY(nonce, privateKey, amount1)
		if err != nil {
			data := []string{err.Error()}
			WriteToCsv(data)
			return
		}
		// add conditional to increase liq / call mint based on if member of mapping
		var positions []interface{}
		var reciept2 []*types.Receipt
		if !checkMapMembership(address.Hex()) {
			nonce = ammcore.GetNonceFromSyncMap(address)
			positions, reciept2, err = ammcore.CallMint(nonce, privateKey, amount0, amount1)
			if err != nil {
				data := []string{err.Error()}
				WriteToCsv(data)
				return
			}
		} else {
			nonce = ammcore.GetNonceFromSyncMap(address)
			reciept2, err = ammcore.CallIncreaseLiquidity(nonce, privateKey, amount0, amount1, ReadFromMap(address.Hex()))
			if err != nil {
				data := []string{err.Error()}
				WriteToCsv(data)
				return
			}
		}
		endTime := time.Now()
		elapsedTime := endTime.Sub(startTime)
		var totalGas = 0
		hashes := []string{}
		reciepts := append(reciept2, reciept1...)
		reciepts = append(reciepts, reciept0...)
		for _, reciept := range reciepts {
			totalGas += int(reciept.GasUsed)
			hashes = append(hashes, reciept.TxHash.String())
		}
		block := reciepts[len(reciepts)-1].BlockNumber
		data := []string{elapsedTime.String(), block.String(), strconv.Itoa(totalGas)}
		data = append(data, hashes...)
		WriteToCsv(data)
		for i, element := range positions {
			var position common.Address
			var tokenID *big.Int
			if !(i%2 == 0) {
				position = element.(common.Address)
			} else {
				tokenID = element.(*big.Int)
			}
			string_position := position.Hex()
			WriteToMap(string_position, tokenID)
		}
	} else if row["txType"] == "burn" {
		fmt.Println("Checking gas price")
		ammcore.CheckCurrentGasPrice(upperGasLimit, sleepDuration)
		fmt.Println("Burning")
		amount0, _ := new(big.Int).SetString(row["arg1"].(string), 10)
		amount1, _ := new(big.Int).SetString(row["arg2"].(string), 10)
		tokenID := ReadFromMap(address.Hex())
		startTime := time.Now()
		nonce := ammcore.GetNonceFromSyncMap(address)
		reciepts, err := ammcore.CallDecreaseLiquidity(nonce, privateKey, amount0, amount1, tokenID)
		if err != nil {
			data := []string{err.Error()}
			WriteToCsv(data)
			return
		}
		endTime := time.Now()
		elapsedTime := endTime.Sub(startTime)
		var totalGas = 0
		hashes := []string{}
		for _, reciept := range reciepts {
			totalGas += int(reciept.GasUsed)
			hashes = append(hashes, reciept.TxHash.String())
		}
		block := reciepts[len(reciepts)-1].BlockNumber
		effectiveGasPrice := reciepts[len(reciepts)-1].EffectiveGasPrice
		data := []string{elapsedTime.String(), block.String(), strconv.Itoa(totalGas), effectiveGasPrice.String()}
		data = append(data, hashes...)
		WriteToCsv(data)
	} else if row["txType"] == "collect" {
		fmt.Println("Checking gas price")
		ammcore.CheckCurrentGasPrice(upperGasLimit, sleepDuration)
		fmt.Println("Collecting")
		tokenID := ReadFromMap(address.Hex())
		startTime := time.Now()
		nonce := ammcore.GetNonceFromSyncMap(address)
		reciepts, err := ammcore.CallCollect(nonce, privateKey, tokenID)
		if err != nil {
			data := []string{err.Error()}
			WriteToCsv(data)
			return
		}
		endTime := time.Now()
		elapsedTime := endTime.Sub(startTime)
		var totalGas = 0
		hashes := []string{}
		for _, reciept := range reciepts {
			totalGas += int(reciept.GasUsed)
			hashes = append(hashes, reciept.TxHash.String())
		}
		block := reciepts[len(reciepts)-1].BlockNumber
		data := []string{elapsedTime.String(), block.String(), strconv.Itoa(totalGas)}
		data = append(data, hashes...)
		WriteToCsv(data)
	} else {
		fmt.Println("Checking gas price")
		ammcore.CheckCurrentGasPrice(upperGasLimit, sleepDuration)
		fmt.Println("Swapping")
		amount, _ := new(big.Int).SetString(row["arg2"].(string), 10)
		zeroForOne, _ := strconv.ParseBool(row["arg1"].(string))
		startTime := time.Now()
		var reciept0 []*types.Receipt
		var err error
		if zeroForOne {
			nonce := ammcore.GetNonceFromSyncMap(address)
			reciept0, err = ammcore.CallApproveABTX(nonce, privateKey, amount)
			if err != nil {
				data := []string{err.Error()}
				WriteToCsv(data)
				return
			}
		} else {
			nonce := ammcore.GetNonceFromSyncMap(address)
			reciept0, err = ammcore.CallApproveABTY(nonce, privateKey, amount)
			if err != nil {
				data := []string{err.Error()}
				WriteToCsv(data)
				return
			}
		}
		nonce := ammcore.GetNonceFromSyncMap(address)
		reciepts, err := ammcore.CallSwap(nonce, privateKey, amount, zeroForOne)
		if err != nil {
			data := []string{err.Error()}
			WriteToCsv(data)
			return
		}
		endTime := time.Now()
		reciepts = append(reciepts, reciept0...)
		elapsedTime := endTime.Sub(startTime)
		var totalGas = 0
		hashes := []string{}
		for _, reciept := range reciepts {
			totalGas += int(reciept.GasUsed)
			hashes = append(hashes, reciept.TxHash.String())
		}
		block := reciepts[len(reciepts)-1].BlockNumber
		data := []string{elapsedTime.String(), block.String(), strconv.Itoa(totalGas)}
		data = append(data, hashes...)
		WriteToCsv(data)
	}
}

func WriteToCsv(data []string) {
	csvMutex.Lock()
	defer csvMutex.Unlock()
	if err := csvWriter.Write(data); err != nil {
		panic(err)
	}
	csvWriter.Flush()
}

func ReadFromMap(key string) *big.Int {
	mapMutex.Lock()
	defer mapMutex.Unlock()
	return positionMap[key]
}

func WriteToMap(key string, value *big.Int) {
	mapMutex.Lock()
	defer mapMutex.Unlock()
	positionMap[key] = value
}

func checkMapMembership(key string) bool {
	mapMutex.Lock()
	defer mapMutex.Unlock()
	_, ok := positionMap[key]
	if ok {
		return true
	} else {
		return false
	}
}

func main() {
	csvOutputFile, _ := os.OpenFile("./metrics.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer csvOutputFile.Close()
	csvWriter = csv.NewWriter(csvOutputFile)
	positionMap = make(map[string]*big.Int)

	// set upper gas limit before starting
	upperGasLimit = big.NewInt(0)
	// time in minutes before rerunning a check on gas (if gasprice > upperGasLimit)
	sleepDuration = 5

	//gasPrice := ammcore.GetCurrentGasPrice()
	//fmt.Println(gasPrice)

	//batchSize := 1
	//RunBaseline(batchSize)
}

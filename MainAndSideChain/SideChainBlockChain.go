package MainAndSideChain

import (
	"ammcore/lib"
	"encoding/csv"
	"math"
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chainBoostScale/ChainBoost/MainAndSideChain/blockchain"
	"github.com/chainBoostScale/ChainBoost/onet/log"
	"golang.org/x/exp/maps"
)

var ztk map[string]bool = make(map[string]bool)

/*
	 ----------------------------------------------------------------------
		updateSideChainBC:  when a side chain leader submit a meta block, the side chain blockchain is
		updated by the root node to reflelct an added meta-block
	 ----------------------------------------------------------------------
*/

func (bz *ChainBoost) updateSideChainBCRound(LeaderName string, blocksize int) {
	//var epochNumber = int(math.Floor(float64(bz.MCRoundNumber) / float64(bz.MCRoundPerEpoch)))
	var err error
	// var rows *excelize.Rows
	// var row []string
	takenTime := time.Now()

	_, info, err := blockchain.SideChainRoundTableGetLastRow()
	if err != nil {
		panic(err)
	}
	var RoundIntervalSec int
	if info == nil {
		RoundIntervalSec = int(time.Now().Unix())
		log.Lvl3("Final result SC: round number: ", 0, "took ", RoundIntervalSec, " seconds in total")
	} else {
		RoundIntervalSec = int(time.Since(info.StartTime).Seconds())
		log.Lvl3("Final result SC: round number: ", info.RoundNumber, "took ", RoundIntervalSec, " seconds in total")
	}

	err = blockchain.InsertIntoSideChainRoundTable(int(bz.SCRoundNumber.Load()), blocksize, LeaderName, 0, time.Now(), 0, 0, 0, RoundIntervalSec, int(bz.MCRoundNumber.Load()))
	if err != nil {
		panic(err)
	}
	log.Lvl1("updateSideChainBCRound took:", time.Since(takenTime).String(), "for sc round number", bz.SCRoundNumber)
}

// /Generates transactions for the system
func (bz *ChainBoost) txgen() {

	numberofTxToGen := int(math.Ceil(float64(bz.DailyVolume) / (3600 * 24.0) * float64(bz.BlockTime)))
	MintTxSize, BurnTxSize, CollectTxSize, SwapTxSize := blockchain.TransactionMeasurement(0, 0)
	scFirstQueueTxs := make([]blockchain.SideChainFirstQueueEntry, 0)
	epoch := int(bz.SCRoundNumber.Load() / int64(bz.SCRoundPerEpoch))
	for i := 0; i < numberofTxToGen; i++ {
		r := rand.Float64()
		var tx blockchain.SideChainFirstQueueEntry
		if r < bz.SwapRate {
			tx = blockchain.SideChainFirstQueueEntry{Name: "TxSwap", Size: int(SwapTxSize), Time: time.Now(), IssuedScRoundNumber: int(bz.SCRoundNumber.Load()), ServAgrId: i + 1, Epoch: epoch}
		} else if r < bz.SwapRate+bz.BurnRate {
			tx = blockchain.SideChainFirstQueueEntry{Name: "TxBurn", Size: int(BurnTxSize), Time: time.Now(), IssuedScRoundNumber: int(bz.SCRoundNumber.Load()), ServAgrId: i + 1, Epoch: epoch}
		} else if r < bz.SwapRate+bz.BurnRate+bz.CollectRate {
			tx = blockchain.SideChainFirstQueueEntry{Name: "TxCollect", Size: int(CollectTxSize), Time: time.Now(), IssuedScRoundNumber: int(bz.SCRoundNumber.Load()), ServAgrId: i + 1, Epoch: epoch}
		} else {
			tx = blockchain.SideChainFirstQueueEntry{Name: "TxMint", Size: int(MintTxSize), Time: time.Now(), IssuedScRoundNumber: int(bz.SCRoundNumber.Load()), ServAgrId: i + 1, Epoch: epoch}
		}

		scFirstQueueTxs = append(scFirstQueueTxs, tx)
	}
	for i := 0; i < len(scFirstQueueTxs); i += 2000 {

		limit := math.Min(float64(len(scFirstQueueTxs)), float64(i+2000))
		err := blockchain.BulkInsertIntoSideChainFirstQueue(scFirstQueueTxs[i:int(limit)])
		if err != nil {
			panic(err)
		}
	}
}

// executes and packs transactions into a block.
func (bz *ChainBoost) updateSideChainBCTransactionQueueTake() int {
	var err error
	takenTime := time.Now()
	//var epochNumber = int(math.Floor(float64(bz.MCRoundNumber) / float64(bz.MCRoundPerEpoch)))
	// --- reset
	bz.SideChainQueueWait = 0

	var accumulatedTxSize int
	blockIsFull := false
	_, MetaBlockSizeMinusTransactions := blockchain.SCBlockMeasurement()
	// --------------- adding bls signature size  -----------------
	log.Lvl4("Size of bls signature:", len(bz.SCSig))
	MetaBlockSizeMinusTransactions = MetaBlockSizeMinusTransactions + len(bz.SCSig)
	// ------------------------------------------------------------
	//var TakeTime time.Time

	/* -----------------------------------------------------------------------------
		 -- take por transactions from sheet: FirstQueue
	----------------------------------------------------------------------------- */
	// --------------------------------------------------------------------
	// looking for last round's number in the round table sheet in the sidechainbc file
	// --------------------------------------------------------------------
	// finding the last row in side chain bc file, in round table sheet
	blockIsFull = false
	accumulatedTxSize = 0

	numberOfPoRTx := 0

	rows, err := blockchain.SideChainGetFirstQueue()
	if err != nil {
		panic(err)
	}
	index := 0
	lastRowId := 0
	// open CSV file
	file, err := os.OpenFile("./data.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	data := []string{"sidechainADDR", "txType", "arg1", "arg2", "round"}
	writer.Write(data)
	// len(rows) gives number of rows - the length of the "external" array
	for index = 0; index <= len(rows)-1 && !blockIsFull; index++ {
		row := rows[index]
		/* each transaction has the following column stored on the transaction queue sheet:
		0) name
		1) size
		2) time
		3) issuedMCRoundNumber
		4) ServAgrId */
		if accumulatedTxSize+row.Size <= bz.SideChainBlockSize-MetaBlockSizeMinusTransactions {
			accumulatedTxSize = accumulatedTxSize + row.Size
			log.Lvl4("a por tx added to block number", bz.MCRoundNumber, " from the queue")
			numberOfPoRTx++
			bz.SideChainQueueWait = bz.SideChainQueueWait + int(bz.SCRoundNumber.Load()-int64(row.IssuedScRoundNumber))
			lastRowId = row.RowId
			epoch := int(bz.SCRoundNumber.Load()) / bz.SCRoundPerEpoch
			if row.Name == "TxSwap" {
				_key := bz.SelectRandomDepositor(epoch)
				zeroForOne, ok := ztk[_key]
				if !ok {
					ztk[_key] = true
				} else {
					ztk[_key] = !ztk[_key]
				}
				var amount *big.Int
				var price *big.Int
				if zeroForOne {
					price = lib.EncodeSqrtRatioX96(big.NewInt(1000000000000000000), big.NewInt(1000000000000000001))
					amount = lib.Exponentiate(1, 18)
				} else {
					price = lib.EncodeSqrtRatioX96(big.NewInt(1000000000000000001), big.NewInt(1000000000000000000))
					amount = lib.Exponentiate(1, 18)
				}

				_, a1, a2 := Pool.Swap(_key, zeroForOne, amount, price)
				Users[_key].Balance.TokenA.Add(Users[_key].Balance.TokenA, a1)
				Users[_key].Balance.TokenB.Add(Users[_key].Balance.TokenB, a2)
				data := []string{_key, "swap", strconv.FormatBool(zeroForOne), amount.String(), bz.SCRoundNumber.String()}
				writer.Write(data)
			} else if row.Name == "TxCollect" {
				keys := maps.Keys(Pool.Positions)
				index := rand.Intn(len(keys))
				collector := keys[index]
				idx0x := strings.Index(collector, "0x")
				user := collector[idx0x : idx0x+42]
				tokenA := lib.Exponentiate(1, 18)
				tokenB := lib.Exponentiate(1, 18)
				a1, a2 := Pool.Collect(user, int32(lib.GetMinTick(lib.TickSpacings[lib.FeeMedium])), int32(lib.GetMaxTick(lib.TickSpacings[lib.FeeMedium])), tokenA, tokenB)
				Users[user].Balance.TokenA.Add(Users[user].Balance.TokenA, a1)
				Users[user].Balance.TokenB.Add(Users[user].Balance.TokenB, a2)
				data := []string{user, "collect", tokenA.String(), tokenB.String(), bz.SCRoundNumber.String()}
				writer.Write(data)
			} else if row.Name == "TxBurn" {
				keys := maps.Keys(Pool.Positions)
				index := rand.Intn(len(keys))
				collector := keys[index]
				idx0x := strings.Index(collector, "0x")
				user := collector[idx0x : idx0x+42]
				tokenA := lib.Exponentiate(1, 18)
				tokenB := lib.Exponentiate(1, 18)
				a1, a2 := Pool.BurnInterface(user, int32(lib.GetMinTick(lib.TickSpacings[lib.FeeMedium])), int32(lib.GetMaxTick(lib.TickSpacings[lib.FeeMedium])), tokenA, tokenB)
				_, _ = a1, a2
				Users[user].Balance.TokenA.Add(Users[user].Balance.TokenA, a1)
				Users[user].Balance.TokenB.Add(Users[user].Balance.TokenB, a2)
				data := []string{user, "burn", tokenA.String(), tokenB.String(), bz.SCRoundNumber.String()}
				writer.Write(data)
			} else {
				_key := bz.SelectRandomDepositor(epoch)
				tokenA := lib.Exponentiate(1, 18)
				tokenB := lib.Exponentiate(1, 18)

				a1, a2 := Pool.MintInterface(_key, int32(lib.GetMinTick(lib.TickSpacings[lib.FeeMedium])), int32(lib.GetMaxTick(lib.TickSpacings[lib.FeeMedium])), tokenA, tokenB)
				Users[_key].Balance.TokenA.Sub(Users[_key].Balance.TokenA, a1)
				Users[_key].Balance.TokenB.Sub(Users[_key].Balance.TokenB, a2)
				data := []string{_key, "mint", tokenA.String(), tokenB.String(), bz.SCRoundNumber.String()}
				writer.Write(data)
			}

			bz.SummPoRTxs[row.ServAgrId] = bz.SummPoRTxs[row.ServAgrId] + 1
		} else {
			blockIsFull = true
			log.Lvl1("final result SC:\n side chain block is full! ")
			err = blockchain.SideChainRoundTableSetBlockSpaceIsFull(int(bz.SCRoundNumber.Load()))
			if err != nil {
				panic(err)
			}
			break
		}
	}

	err = blockchain.SideChainDeleteFromFirstQueue(lastRowId)
	if err != nil {
		panic(err)
	}
	var avgWait float64 = 0
	if numberOfPoRTx != 0 {
		avgWait = float64(bz.SideChainQueueWait) / float64(numberOfPoRTx)
	}
	err = blockchain.SideChainRoundTableSetFinalRoundInfo(accumulatedTxSize+MetaBlockSizeMinusTransactions,
		numberOfPoRTx,
		avgWait,
		int(bz.SCRoundNumber.Load()),
		len(rows)-numberOfPoRTx)
	if err != nil {
		panic(err)
	}
	log.LLvl1("final result SC:\n In total in sc round number ", bz.SCRoundNumber,
		"\n number of published PoR transactions is", numberOfPoRTx)

	log.Lvl1("final result SC:\n", " this round's block size: ", accumulatedTxSize+MetaBlockSizeMinusTransactions)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	// fill OverallEvaluation Sheet
	updateSideChainBCOverallEvaluation(int(bz.SCRoundNumber.Load()))
	blocksize := accumulatedTxSize + MetaBlockSizeMinusTransactions
	log.Lvl1("updateSideChainBCTransactionQueueTake took:", time.Since(takenTime).String())
	return blocksize
}

// --------------------------------------------------------------------------------
// ----------------------- OverallEvaluation Sheet --------------------
// --------------------------------------------------------------------------------
func updateSideChainBCOverallEvaluation(SCRoundNumber int) {
	var err error
	takenTime := time.Now()

	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	sumBC, err := blockchain.GetSumSideChain("RoundTable", "BCSize")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	sumPoRTx, err := blockchain.GetSumSideChain("RoundTable", "PoRTx")
	if err != nil {
		log.LLvl1(err)
	}

	avgWaitTx, err := blockchain.GetAvgSideChain("RoundTable", "AveWait")
	if err != nil {
		log.LLvl1(err)
	}
	/*
		FormulaString = "=SUM(RoundTable!H2:H" + CurrentRow + ")"
		err = f.SetCellFormula("OverallEvaluation", axisOverallBlockSpaceFull, FormulaString)
		if err != nil {
			log.LLvl1(err)
		}*/

	err = blockchain.InsertIntoSideChainOverallEvaluation(SCRoundNumber, sumBC, sumPoRTx, avgWaitTx, 0)
	if err != nil {
		panic(err)
	}
	log.Lvl1("updateSideChainBCOverallEvaluation took:", time.Since(takenTime).String())
}

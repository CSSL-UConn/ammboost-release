package blockchain

import (
	"crypto/sha256"
	"math/rand"
	"time"

	//"github.com/chainBoostScale/ChainBoost/vrf"
	// ToDoRaha: later that I brought everything from blscosi package to ChainBoost package, I shoudl add another pacckage with
	// some definitions in it to be imported/used in blockchain(here) and simulation package (instead of using blscosi/protocol)

	"github.com/chainBoostScale/ChainBoost/onet/log"
)

/* -------------------------------------------------------------------- */
//  ----------------  Block and Transactions size measurements -----
/* -------------------------------------------------------------------- */
// BlockMeasurement compute the size of meta data and every thing other than the transactions inside the block
func BlockMeasurement() (BlockSizeMinusTransactions int) {
	// -- Hash Sample ----
	sha := sha256.New()
	if _, err := sha.Write([]byte("a sample seed")); err != nil {
		log.Error("Couldn't hash header:", err)
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	hash := sha.Sum(nil)
	var hashSample [32]uint8
	copy(hashSample[:], hash[:])
	// --
	var Version [4]byte
	var cnt [2]byte
	var feeSample, BlockSizeSample [3]byte
	var MCRoundNumberSample [3]byte
	// ---------------- block sample ----------------
	var TxPayArraySample []*MC_TxPay
	var TxPorArraySample []*SC_TxPoR
	var TxServAgrProposeArraySample []*SC_TxServAgrPropose
	var TxServAgrCommitSample []*SC_TxServAgrCommit
	var TxStoragePaySample []*MC_TxStoragePay

	x9 := &TransactionList{
		//---
		TxPays:   TxPayArraySample,
		TxPayCnt: cnt,
		//---
		TxPoRs:   TxPorArraySample,
		TxPoRCnt: cnt,
		//---
		TxServAgrProposes:   TxServAgrProposeArraySample,
		TxServAgrProposeCnt: cnt,
		//---
		TxServAgrCommits:   TxServAgrCommitSample,
		TxServAgrCommitCnt: cnt,
		//---
		TxStoragePay:    TxStoragePaySample,
		TxStoragePayCnt: cnt,

		Fees: feeSample,
	}
	// real! TransactionListSize = size.Of(x9) + sum of size of included transactions
	// --- VRF
	//: ToDoRaha: temp comment
	// t := []byte("first round's seed")
	// VrfPubkey, VrfPrivkey := vrf.VrfKeygen()
	// proof, _ := VrfPrivkey.ProveBytes(t)
	// _, vrfOutput := VrfPubkey.VerifyBytes(proof, t)
	// var nextroundseed [64]byte =  // vrfOutput
	// var VrfProof [80]byte = proof
	// --- time
	ti := []byte(time.Now().String())
	var timeSample [4]byte
	copy(timeSample[:], ti[:])
	// ---

	x10 := &BlockHeader{
		MCRoundNumber: MCRoundNumberSample,
		//RoundSeed:         nextroundseed,
		//LeadershipProof:   VrfProof,
		PreviousBlockHash: hashSample,
		Timestamp:         timeSample,
		MerkleRootHash:    hashSample,
		Version:           Version,
	}
	x11 := &Block{
		BlockSize:       BlockSizeSample,
		BlockHeader:     x10,
		TransactionList: x9,
	}

	log.Lvl5(x11)

	BlockSizeMinusTransactions = len(BlockSizeSample) + //x11
		len(MCRoundNumberSample) + /*ToDoRaha: temp comment: len(nextroundseed) + len(VrfProof) + */ len(hashSample) + len(timeSample) + len(hashSample) + len(Version) + //x10
		5*len(cnt) + len(feeSample) //x9
	// ---
	log.Lvl4("Block Size Minus Transactions is: ", BlockSizeMinusTransactions)

	return BlockSizeMinusTransactions
}

// TransactionMeasurement computes the size of 5 types of transactions we currently have in the system:
// Por, ServAgrPropose, Pay, StoragePay, ServAgrCommit
func TransactionMeasurement(SectorNumber, SimulationSeed int) (MintTxSize uint32, BurnTxSize uint32, CollectTxSize uint32, SwapTxSize uint32) {
	// -- Hash Sample ----
	MintTxSize = 814
	BurnTxSize = 907
	CollectTxSize = 922
	SwapTxSize = 1008

	return MintTxSize, BurnTxSize, CollectTxSize, SwapTxSize
}

/* -------------------------------------------------------------------------------------------
    ------------- measuring side chain's sync transaction, summary and meta blocks ------
-------------------------------------------------------------------------------------------- */

// MetaBlockMeasurement compute the size of meta data and every thing other than the transactions inside the meta block
func SCBlockMeasurement() (SummaryBlockSizeMinusTransactions int, MetaBlockSizeMinusTransactions int) {
	// ----- block header sample -----
	// -- Hash Sample ----
	sha := sha256.New()
	if _, err := sha.Write([]byte("a sample seed")); err != nil {
		log.Error("Couldn't hash header:", err)
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	hash := sha.Sum(nil)
	var hashSample [32]uint8
	copy(hashSample[:], hash[:])
	// --
	var Version [4]byte
	var cnt [2]byte
	var feeSample, BlockSizeSample [3]byte
	var SCRoundNumberSample [3]byte
	var samplePublicKey [33]byte
	//var samplePublicKey kyber.Point
	// --- VRF
	// t := []byte("first round's seed")
	// VrfPubkey, VrfPrivkey := vrf.VrfKeygen()
	// proof, _ := VrfPrivkey.ProveBytes(t)
	// _, vrfOutput := VrfPubkey.VerifyBytes(proof, t)
	// var nextroundseed [64]byte = vrfOutput
	// var VrfProof [80]byte = proof
	// --- time
	ti := []byte(time.Now().String())
	var timeSample [4]byte
	copy(timeSample[:], ti[:])
	// ---
	x10 := &SCBlockHeader{
		SCRoundNumber: SCRoundNumberSample,
		//RoundSeed:         nextroundseed,
		//LeadershipProof:   VrfProof,
		PreviousBlockHash: hashSample,
		Timestamp:         timeSample,
		MerkleRootHash:    hashSample,
		Version:           Version,
		LeaderPublicKey:   samplePublicKey,
		//BlsSignature:      sampleBlsSig, // this will be added back in the protocol
	}
	// ---------------- meta block sample ----------------
	var TxPorArraySample []*SC_TxPoR

	x9 := &SCMetaBlockTransactionList{
		TxPoRs:   TxPorArraySample,
		TxPoRCnt: cnt,
		Fees:     feeSample,
	}
	// real! TransactionListSize = size.Of(x9) + sum of size of included transactions

	x11 := &SCMetaBlock{
		BlockSize:                  BlockSizeSample,
		SCBlockHeader:              x10,
		SCMetaBlockTransactionList: x9,
	}

	log.Lvl5(x11)

	MetaBlockSizeMinusTransactions = len(BlockSizeSample) + //x11: SCMetaBlock
		len(SCRoundNumberSample) + /* len(nextroundseed) + len(VrfProof) +*/ len(hashSample) + len(timeSample) +
		len(hashSample) + len(Version) + len(samplePublicKey) + //x10: SCBlockHeader
		len(cnt) + len(feeSample) //x9: SCMetaBlockTransactionList
	// ---
	log.Lvl4("Meta Block Size Minus Transactions is: ", MetaBlockSizeMinusTransactions)

	//------------------------------------- Summary block -----------------------------
	// ---------------- summary block sample ----------------
	var TxSummaryArraySample []*TxSummary

	x12 := &SCSummaryBlockTransactionList{
		//---
		TxSummary:    TxSummaryArraySample,
		TxSummaryCnt: cnt,
		Fees:         feeSample,
	}
	x13 := &SCSummaryBlock{
		BlockSize:                     BlockSizeSample,
		SCBlockHeader:                 x10,
		SCSummaryBlockTransactionList: x12,
	}
	log.Lvl5(x13)
	SummaryBlockSizeMinusTransactions = len(BlockSizeSample) + //x13: SCSummaryBlock
		len(SCRoundNumberSample) + /*len(nextroundseed) + len(VrfProof) +*/ len(hashSample) + len(timeSample) + len(hashSample) +
		len(Version) + len(samplePublicKey) + //x10: SCBlockHeader
		len(cnt) + len(feeSample) //x12: SCSummaryBlockTransactionList
	log.Lvl4("Summary Block Size Minus Transactions is: ", SummaryBlockSizeMinusTransactions)

	return SummaryBlockSizeMinusTransactions, MetaBlockSizeMinusTransactions
}

// SyncTransactionMeasurement computes the size of sync transaction
func SyncTransactionMeasurement() (SyncTxSize int) {
	return
}
func SCSummaryTxMeasurement(SummTxNum int) (SummTxsSizeInSummBlock int) {
	r := rand.New(rand.NewSource(int64(0)))
	var a []uint64
	for i := 0; i < SummTxNum; i++ {
		a = append(a, r.Uint64())
	}
	return 2 * len(a)
}

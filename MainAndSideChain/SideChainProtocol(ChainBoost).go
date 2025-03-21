package MainAndSideChain

import (
	"ammcore"
	"ammcore/TokenBank"
	"ammcore/lib"
	"bgls"
	"bytes"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"encoding/binary"
	"encoding/csv"
	"encoding/gob"

	"github.com/chainBoostScale/ChainBoost/MainAndSideChain/BLSCoSi"
	"github.com/chainBoostScale/ChainBoost/MainAndSideChain/blockchain"
	"github.com/chainBoostScale/ChainBoost/onet"
	"github.com/chainBoostScale/ChainBoost/onet/log"
	"github.com/chainBoostScale/ChainBoost/onet/network"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"golang.org/x/exp/maps"
	"golang.org/x/xerrors"
)

/* ----------------------------------------- TYPES -------------------------------------------------
------------------------------------------------------------------------------------------------  */

// channel used by side chain's leader in each side chain's round
type RtLSideChainNewRound struct {
	SCRoundNumber            int
	CommitteeNodesTreeNodeID []onet.TreeNodeID
	blocksize                int
}
type RtLSideChainNewRoundChan struct {
	*onet.TreeNode
	RtLSideChainNewRound
}
type LtRSideChainNewRound struct { //fix this
	NewRound      bool
	SCRoundNumber int
	SCSig         BLSCoSi.BlsSignature
}
type LtRSideChainNewRoundChan struct {
	*onet.TreeNode
	LtRSideChainNewRound
}

/* ----------------------------------- FUNCTIONS -------------------------------------------------
------------------------------------------------------------------------------------------------  */

/*
	 ----------------------------------------------------------------------
		DispatchProtocol listen on the different channels in side chain protocol

------------------------------------------------------------------------
*/
func (bz *ChainBoost) DispatchProtocol() error {

	running := true
	var err error

	for running {
		select {

		case msg := <-bz.ChainBoostDone:
			bz.simulationDone = msg.IsSimulationDone
			os.Exit(0)
			return nil

		// --------------------------------------------------------
		// message recieved from BLSCoSi (SideChain):
		// ******* just the current side chain's "LEADER" recieves this msg
		// note that other messages communicated in BlsCosi protocol are handled by
		// func (p *SubBlsCosi) Dispatch() which is called when the startSubProtocol in
		// Blscosi.go, create subprotocols => hence calls func (p *SubBlsCosi) Dispatch()
		// --------------------------------------------------------
		case sig := <-bz.BlsCosi.Load().FinalSignature:
			if bz.simulationDone {
				return nil
			}

			if err := BLSCoSi.BdnSignature(sig).Verify(bz.BlsCosi.Load().Suite, bz.BlsCosi.Load().Msg, bz.BlsCosi.Load().SubTrees[0].Roster.Publics()); err == nil {
				log.Lvl1("final result SC:", bz.Name(), " : ", bz.BlsCosi.Load().BlockType, "with side chain's round number", bz.SCRoundNumber, "Confirmed in Side Chain")
				err := bz.SendTo(bz.Root(), &LtRSideChainNewRound{
					NewRound:      true,
					SCRoundNumber: int(bz.SCRoundNumber.Load()),
					SCSig:         sig,
				})
				if err != nil {
					return xerrors.New("can't send new round msg to root" + err.Error())
				}
			} else {
				return xerrors.New("error in running this round of blscosi:  " + err.Error())
			}
		}
	}
	return err
}

// makes deposits into the AMM
func (bz *ChainBoost) MakeRandomDeposits() {
	for !bz.Done {
		//tm := rand.Intn(bz.BlockTime)

		for _, user := range Users {
			current_epoch := bz.SCRoundNumber.Load() / int64(bz.SCRoundPerEpoch)
			bz.makeDeposit(int(current_epoch+1), user)
		}
		//duration, _ := time.ParseDuration(fmt.Sprintf("%ds", tm))
		// time.Sleep(duration)
	}
}

// / decodes Big Ints from Hex String
func MustDecodeBigFromHex(hexstring string) *big.Int {
	s := hexstring[2:]
	s = strings.TrimLeft(s, "0")
	s = fmt.Sprintf("0x%s", s)
	return hexutil.MustDecodeBig(s)
}

// SideChainLeaderPreNewRound is run by the side chain's leader (does consensus)
func (bz *ChainBoost) SideChainLeaderPreNewRound(msg RtLSideChainNewRoundChan) error {
	var err error
	//bz.txgen()
	bz.SCRoundNumber.Store(int64(msg.SCRoundNumber))
	blsCosi := bz.BlsCosi.Load()
	blsCosi.Msg = []byte{0xFF}
	bz.BlsCosi.Store(blsCosi)
	takenTime := time.Now()
	// -----------------------------------------------
	// --- updating the next side chain's leader
	// -----------------------------------------------

	var CommitteeNodesServerIdentity []*network.ServerIdentity
	if bz.SCRoundNumber.Load()%int64(bz.SCRoundPerEpoch) == 1 {

		bz.CommitteeNodesTreeNodeID = msg.CommitteeNodesTreeNodeID
		//todo: a out of range bug happens sometimes!
		//log.LLvl1(": debug:", bz.CommitteeWindow-1)
		log.Lvl2(": debug:", bz.CommitteeWindow)
		log.Lvl1("log :bz.CommitteeWindow:", bz.CommitteeWindow)
		for _, a := range bz.CommitteeNodesTreeNodeID[0 : bz.CommitteeWindow-1] {
			//for _, a := range bz.CommitteeNodesTreeNodeID[0:bz.CommitteeWindow] {
			CommitteeNodesServerIdentity = append(CommitteeNodesServerIdentity, bz.Tree().Search(a).ServerIdentity)
		}
		log.Lvl1("final result SC: ", bz.Name(), " is running next side chain's epoch with new committee")
		for i, a := range bz.CommitteeNodesTreeNodeID {
			log.Lvl3("final result SC: BlsCosi: next side chain's epoch committee number ", i, ":", bz.Tree().Search(a).Name())
		}
	} else {
		//log.LLvl1(len(bz.CommitteeNodesTreeNodeID))
		//log.LLvl1(bz.CommitteeNodesTreeNodeID[len(bz.CommitteeNodesTreeNodeID)-(bz.CommitteeWindow):])
		// todo: check that the case with changing committee after choosing next sc leader (me!) doesnt happen and if it does , it doesnt affect my committee!
		for _, a := range bz.CommitteeNodesTreeNodeID[0 : bz.CommitteeWindow-1] {
			CommitteeNodesServerIdentity = append(CommitteeNodesServerIdentity, bz.Tree().Search(a).ServerIdentity)
		}
		log.Lvl1("final result SC: ", bz.Name(), " is running next side chain's epoch with the same committee members:")
		for i, a := range CommitteeNodesServerIdentity {
			log.Lvl3("final result SC: BlsCosi: next side chain's epoch committee number ", i, ":", a.Address)
		}
	}
	if bz.SCRoundNumber.Load() == 0 {
		blsCosi := bz.BlsCosi.Load()
		blsCosi.BlockType = "Summary Block"
		bz.BlsCosi.Store(blsCosi)
	} else {
		blsCosi := bz.BlsCosi.Load()
		blsCosi.BlockType = "Meta Block"
		bz.BlsCosi.Store(blsCosi)
	}
	committeeRoster := onet.NewRoster(CommitteeNodesServerIdentity)
	// --- root should have root index of 0 (this is based on what happens in gen_tree.go)
	var x = *bz.TreeNode()
	x.RosterIndex = 0
	// ---
	blsCosi = bz.BlsCosi.Load()
	blsCosi.SubTrees, err = BLSCoSi.NewBlsProtocolTree(onet.NewTree(committeeRoster, &x), bz.NbrSubTrees)
	bz.BlsCosi.Store(blsCosi)
	if err == nil {
		if bz.SCRoundNumber.Load() == 1 {
			log.Lvl3("final result SC: Next bls cosi tree is: ", bz.BlsCosi.Load().SubTrees[0].Roster.List,
				" with ", bz.Name(), " as Root \n running BlsCosi sc round number", bz.SCRoundNumber)
		}
	} else {
		return xerrors.New("Problem in cosi protocol run:   " + err.Error())
	}
	// ---
	// from bc: update msg size with next block size on side chain
	s := make([]byte, msg.blocksize)
	blsCosi = bz.BlsCosi.Load()
	blsCosi.Msg = append(blsCosi.Msg, s...) // Msg is the meta block
	bz.BlsCosi.Store(blsCosi)
	// ----
	//go func() error {
	bz.BlsCosi.Load().Start()
	bz.consensusTimeStart = time.Now()
	log.Lvl1("SideChainLeaderPreNewRound took:", time.Since(takenTime).String(), "for sc round number", bz.SCRoundNumber)
	//	if err != nil {
	//		return xerrors.New("Problem in cosi protocol run:   " + err.Error())
	//	}
	return nil
	//}()
	//return xerrors.New("Problem in cosi protocol run: should not get here")
}

func getRealSizeOf(v interface{}) (int, error) {
	b := new(bytes.Buffer)
	if err := gob.NewEncoder(b).Encode(v); err != nil {
		return 0, err
	}
	return b.Len(), nil
}

// returns the length of a structure when packed in binary
func pack(syncRecord TokenBank.TokenBanksyncStruct, pos bool) int {
	retval := make([]byte, 0)
	if syncRecord.TxTypeId {
		retval = append(retval, uint8(1))
	} else {
		retval = append(retval, uint8(1))
	}
	a := uint256.MustFromBig(syncRecord.AmountTokenA).Bytes32()
	retval = append(retval, a[:]...)
	a = uint256.MustFromBig(syncRecord.AmountTokenB).Bytes32()
	retval = append(retval, a[:]...)
	a = uint256.MustFromBig(syncRecord.SidechainAddr).Bytes32()
	retval = append(retval, a[:]...)

	if pos {
		var integer []byte = make([]byte, 4)
		binary.BigEndian.PutUint32(integer[:], uint32(len(syncRecord.PositionId)))
		retval = append(retval, integer...)
		retval = append(retval, []byte(syncRecord.PositionId)...)
		b := uint256.MustFromBig(syncRecord.FeesEarnedA).Bytes32()
		retval = append(retval, b[:]...)
		b = uint256.MustFromBig(syncRecord.FeesEarnedB).Bytes32()
		retval = append(retval, b[:]...)

		binary.BigEndian.PutUint32(integer, uint32(syncRecord.UpperBound))
		retval = append(retval, integer...)
		binary.BigEndian.PutUint32(integer, uint32(syncRecord.LowerBound))
		retval = append(retval, integer...)
	}
	return len(retval)
}

// Executes after conseneus and processes the information that happened during that round
func (bz *ChainBoost) SideChainRootPostNewRound(msg LtRSideChainNewRoundChan) error {
	var err error
	bz.SCRoundNumber.Store(int64(msg.SCRoundNumber))
	bz.SCSig = msg.SCSig
	var blocksize int
	bz.txgen()

	// --------------------------------------------------------------------
	if int(bz.SCRoundNumber.Load())%bz.SCRoundPerEpoch == 1 {

		bz.getDeposit(int(bz.SCRoundNumber.Load() / int64(bz.SCRoundPerEpoch)))
	}

	if bz.SCRoundNumber.Load()%int64(bz.SCRoundPerEpoch) == 0 {
		blsCosi := bz.BlsCosi.Load()
		blsCosi.BlockType = "Summary Block"
		bz.BlsCosi.Store(blsCosi) // just to know!
		// ---
		bz.BCLock.Lock()

		syncRecords := make([]TokenBank.TokenBanksyncStruct, 0)
		size := 0
		///FIXME: add summary block logic here.
		for key, element := range Users {
			if element.Balance.TokenA.Cmp(big.NewInt(0)) > 0 && element.Balance.TokenB.Cmp(big.NewInt(0)) > 0 {
				var syncRecord TokenBank.TokenBanksyncStruct
				syncRecord.TxTypeId = true                                                                                    //1
				syncRecord.AmountTokenA = big.NewInt(0).Sub(element.Balance.TokenA, lib.Exponentiate(rand.Int63n(30000), 18)) //32
				syncRecord.AmountTokenB = big.NewInt(0).Sub(element.Balance.TokenB, lib.Exponentiate(rand.Int63n(30000), 18)) //32
				syncRecord.SidechainAddr = MustDecodeBigFromHex(key)                                                          //32
				syncRecord.PositionId = ""                                                                                    //0
				syncRecord.FeesEarnedA = big.NewInt(0)                                                                        //32
				syncRecord.FeesEarnedB = big.NewInt(0)                                                                        //32
				syncRecord.UpperBound = 0                                                                                     // 4
				syncRecord.LowerBound = 0                                                                                     //4
				syncRecords = append(syncRecords, syncRecord)
				size += pack(syncRecord, false)
				element.Balance.TokenA = big.NewInt(0)
				element.Balance.TokenB = big.NewInt(0)
			}
		}

		for _, element := range Pool.PositionSummaries {
			var syncRecord TokenBank.TokenBanksyncStruct
			syncRecord.TxTypeId = false                  //1
			syncRecord.AmountTokenA = element.DeltaT0    //32
			syncRecord.AmountTokenB = element.DeltaT1    //32
			syncRecord.PositionId = element.ID           //42
			syncRecord.SidechainAddr = big.NewInt(0)     //32
			syncRecord.FeesEarnedA = element.FeeChangeT0 // 32
			syncRecord.FeesEarnedB = element.FeeChangeT1 //32
			syncRecord.UpperBound = element.UpperBound   //4
			syncRecord.LowerBound = element.LowerBound   //4
			syncRecords = append(syncRecords, syncRecord)
			size += pack(syncRecord, false)
		}

		blocksize, _ = blockchain.SCBlockMeasurement()
		blocksize += size + len(bz.SCSig)
		keys := maps.Keys(Users)
		key := keys[rand.Intn(len(keys))]

		data, err := ammcore.AbiEncode(syncRecords)
		if err != nil {
			panic(err)
		}

		sig := bgls.Sign(bz.curve, bz.privKey, bz.pubKey, data)

		bi_sig := sig.ToAffineCoords()

		bi_pubkey := bz.pubKey.ToAffineCoords()
		var mu sync.Mutex
		go func() {
			epoch := int(bz.SCRoundNumber.Load()) / bz.SCRoundPerEpoch
			start := time.Now()
			rc := ammcore.Synchronize(Users[key].ethInfo.privKey, syncRecords, [4]*big.Int(bi_pubkey), [2]*big.Int(bi_sig), big.NewInt(int64(epoch)), Pool.Liquidity, Pool.Price)
			end := time.Now()
			mu.Lock()
			defer mu.Unlock()

			file, err := os.OpenFile("sync.csv", os.O_APPEND|os.O_WRONLY, os.ModeAppend|0777)
			if err != nil {
				panic(err)
			}
			data := []string{end.Sub(start).String(), fmt.Sprint(rc.GasUsed), rc.BlockNumber.String(), rc.TxHash.Hex()}
			writer := csv.NewWriter(file)
			writer.Write(data)
			writer.Flush()
			err = writer.Error()
			if err != nil {
				panic(err)
			}
			file.Close()
		}()

		Pool.CleanSummaries()

		bz.updateSideChainBCRound(msg.Name(), blocksize)
		// ---
		bz.BCLock.Unlock()
		// ---
		// reset side chain round number
		bz.SCRoundNumber.Inc()

		//Run New Election.
		bz.UpdateSideChainCommittee()

		log.Lvl1("Final result SC: BlsCosi: next side chain's epoch leader is: ", bz.Tree().Search(bz.NextSideChainLeader).Name())
		for i, a := range bz.CommitteeNodesTreeNodeID {
			log.Lvl3("Final result SC: BlsCosi: next side chain's epoch committee number ", i, ":", bz.Tree().Search(a).Name())
		}

	} else {
		// ---
		bz.BCLock.Lock()
		// ---
		// next meta block on side chain blockchian is added by the root node
		bz.updateSideChainBCRound(msg.Name(), 0) // we dont use blocksize param bcz when we are generating meta block
		// the block size is measured and added in the func: updateSideChainBCTransactionQueueTake
		blocksize = bz.updateSideChainBCTransactionQueueTake()
		blsCosi := bz.BlsCosi.Load()
		blsCosi.BlockType = "Meta Block"
		bz.BlsCosi.Store(blsCosi) // just to know!
		// ---
		bz.BCLock.Unlock()
		// ---
		// increase side chain round number
		bz.SCRoundNumber.Inc()
		log.Lvl1("Raha Debug: wgSCRound.Done")
		//bz.wgSCRound.Done()
	}

	if bz.SCRoundNumber.Load()-1 == int64(bz.SimulationRounds) {

		log.LLvl1("ChainBoost simulation has passed the number of simulation rounds:", bz.SimulationRounds, "\n returning back to RunSimul")
		bz.DoneRootNode <- true
		return nil
	}

	if time.Since(bz.TimeStart) >= time.Duration(bz.BlockTime) {
		d, _ := time.ParseDuration(fmt.Sprintf("%ds", bz.BlockTime))
		time.Sleep(time.Until(bz.TimeStart.Add(d)))
	}

	// --------------------------------------------------------------------
	//triggering next side chain round leader to run next round of blscosi
	// --------------------------------------------------------------------
	if bz.DepositMode == 1 && bz.SCRoundNumber.Load()%int64(bz.SCRoundPerEpoch) == 1 {
		bz.StopDeposit.Store(true)
		bz.StopDepositWg.Wait()
	}
	err = bz.SendTo(bz.Tree().Search(bz.NextSideChainLeader), &RtLSideChainNewRound{
		SCRoundNumber:            int(bz.SCRoundNumber.Load()),
		CommitteeNodesTreeNodeID: bz.CommitteeNodesTreeNodeID,
		blocksize:                blocksize,
	})
	if err != nil {
		log.LLvl1(bz.Name(), "can't send new side chain round msg to", bz.Tree().Search(bz.NextSideChainLeader).Name())
		return xerrors.New("can't send new side chain round msg to next leader" + err.Error())
	}
	bz.TimeStart = time.Now()
	return nil
}

// ----------------------------------------------------------------------------------------------
// ---------------- Checks that Users and Keys exists, initializes committee and starts running the protcol --------
// ----------------------------------------------------------------------------------------------
func (bz *ChainBoost) StartSideChainProtocol() {
	var err error

	//bz.BCLock.Lock()
	//bz.updateSideChainBCTransactionQueueCollect()
	//bz.BCLock.Unlock()

	// -----------------------------------------------
	// --- initializing side chain's msg and side chain's committee roster index for the second run and the next runs
	// -----------------------------------------------
	rand := rand.New(rand.NewSource(int64(bz.SimulationSeed)))
	var r []int
	var d int
	for i := 0; i < bz.CommitteeWindow; i++ {
		d = int(math.Abs(float64(int(rand.Uint64())))) % len(bz.Tree().List())
		if !contains(r, d) {
			r = append(r, d)
		} else {
			i--
		}
	}

	for i := 0; i < bz.CommitteeWindow; i++ {
		d := r[i]
		bz.CommitteeNodesTreeNodeID = append(bz.CommitteeNodesTreeNodeID, bz.Tree().List()[d].ID)
	}
	for i, a := range bz.CommitteeNodesTreeNodeID {
		log.Lvl1("final result SC: Initial BlsCosi committee queue: ", i, ":", bz.Tree().Search(a).Name())
	}
	bz.NextSideChainLeader = bz.CommitteeNodesTreeNodeID[0]
	// -----------------------------------------------
	// --- initializing next side chain's leader
	// -----------------------------------------------
	//bz.CommitteeNodesTreeNodeID = append([]onet.TreeNodeID{bz.Tree().List()[bz.CommitteeWindow+1].ID}, bz.CommitteeNodesTreeNodeID[:bz.CommitteeWindow-1]...)
	//bz.NextSideChainLeader = bz.Tree().List()[bz.CommitteeWindow+1].ID
	bz.curve = bgls.CurveSystem(bgls.Altbn128)
	blockchain.KeyDBMustExist()

	bz.privKey, bz.pubKey, _ = blockchain.UnpickleKeys()

	Users = make(map[string]UserInfo, 0)
	blockchain.UserDBMustExist()
	users, _ := blockchain.GetUsers()
	for _, user := range users {
		address := user.AmmAddress
		var userInfo UserInfo
		userInfo.ethInfo.privKey = user.PrivKey
		userInfo.ethInfo.pubkey = user.Pubkey
		userInfo.Balance = &Balance{TokenA: big.NewInt(0), TokenB: big.NewInt(0)}
		userInfo.ammId = address
		Users[address] = userInfo
	}

	price := lib.EncodeSqrtRatioX96(big.NewInt(1), big.NewInt(1))
	Pool = ammcore.PoolBuilder(price)
	keys := maps.Keys(Users)
	Pool.Mint(keys[0], int32(lib.GetMinTick(lib.TickSpacings[lib.FeeMedium])), int32(lib.GetMaxTick(lib.TickSpacings[lib.FeeMedium])), lib.Exponentiate(9000, 18)) // Initial Mint

	bz.DepositArray = make([]sync.Map, bz.SimulationRounds)
	ammcore.Deposit(Users[keys[0]].ethInfo.privKey, MustDecodeBigFromHex(keys[0]), big.NewInt(30000), big.NewInt(30000), big.NewInt(0))
	bz.DepositArray[0].Store(keys[0], true) // Initial Deposit

	bz.printDeposits(0)
	//if bz.DepositMode == 0 {
	//	go bz.MakeRandomDeposits()
	//}

	createFile("deposit.csv")
	createFile("sync.csv")
	fmt.Printf("Making Deposits")
	epochs := bz.SimulationRounds / bz.SCRoundPerEpoch
	for i := 0; i < epochs; i++ {
		wg := new(sync.WaitGroup)
		wg.Add(10)
		for j := 0; j < 10; j++ {
			go func(j int) {
				fmt.Println("Epoch: ", i, " j ", j, " Depositor", j*10+(i%10))
				bz.makeDeposit(i, Users[keys[j*10+(i%10)]])
				wg.Done()
			}(j)
		}
		wg.Wait()
	}

	// --------------------------------------------------------------------
	// triggering next side chain round leader to run next round of blscosi
	// --------------------------------------------------------------------

	blsCosi := bz.BlsCosi.Load()
	blsCosi.Msg = []byte{0xFF}
	bz.BlsCosi.Store(blsCosi)

	err = bz.SendTo(bz.Tree().Search(bz.NextSideChainLeader), &RtLSideChainNewRound{
		SCRoundNumber:            int(bz.SCRoundNumber.Load()),
		CommitteeNodesTreeNodeID: bz.CommitteeNodesTreeNodeID,
	})
	if err != nil {
		log.LLvl1(bz.Name(), "can't send new side chain round msg to", bz.Tree().Search(bz.NextSideChainLeader).Name())
		panic("can't send new side chain round msg to the first leader")
	}
	bz.TimeStart = time.Now()

}

func createFile(filepath string) {
	file, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		panic(err)
	}
	data := []string{"time taken", "gas fees", "block", "txId"}
	writer := csv.NewWriter(file)
	writer.Flush()
	writer.Write(data)
	err = writer.Error()
	if err != nil {
		panic(err)
	}
	file.Close()
}

func getUsersForBatch(start, batchSize int) []UserInfo {
	userList := make([]UserInfo, 0)
	i := 0
	for k, user := range Users {
		if i >= start && i < start+batchSize {
			user.ammId = k
			userList = append(userList, user)
		}
		i++
	}
	return userList
}

// maintain array of keys (sidechain key, string)
func (bz *ChainBoost) makeDeposit(epoch int, user UserInfo) {
	var mu sync.Mutex

	//bz.printDeposits()

	startTime := time.Now()
	rcs := ammcore.Deposit(user.ethInfo.privKey, MustDecodeBigFromHex(user.ammId), big.NewInt(30000), big.NewInt(30000), big.NewInt(int64(epoch)))
	var GasFees uint64 = 0
	hashes := make([]string, 0)
	blocks := make([]string, 0)
	for _, rc := range rcs {
		GasFees += rc.GasUsed
		hashes = append(hashes, rc.TxHash.Hex())
		blocks = append(blocks, rc.BlockNumber.String())
	}

	endTime := time.Now()
	bz.DepositArray[epoch].Store(user.ammId, true)
	mu.Lock()
	defer mu.Unlock()
	file, err := os.OpenFile("deposit.csv", os.O_APPEND|os.O_WRONLY, os.ModeAppend|0777)
	if err != nil {
		panic(err)
	}
	data := []string{endTime.Sub(startTime).String(), fmt.Sprint(GasFees)}
	data = append(data, blocks...)
	data = append(data, hashes...)
	writer := csv.NewWriter(file)
	writer.Write(data)
	writer.Flush()
	err = writer.Error()
	if err != nil {
		panic(err)
	}
	file.Close()
}

func (bz *ChainBoost) printDeposits(epoch int) {
	var keys []string
	bz.DepositArray[epoch].Range(func(key, value any) bool {
		keys = append(keys, key.(string))
		return true
	})
	fmt.Println("Epoch: ", epoch, " Depositors: ", keys)
}

func (bz *ChainBoost) getDeposit(epoch int) {
	//get deposit.
	//deposits := make(map[string]tokenPair)
	bz.printDeposits(epoch)
	for k, _ := range Users {
		_, member := bz.DepositArray[epoch].Load(k)
		if member {
			Users[k].Balance.TokenA, Users[k].Balance.TokenB = ammcore.GetDepositBalance(MustDecodeBigFromHex(k), big.NewInt(int64(epoch)))
			//Users[k].Balance.TokenA, Users[k].Balance.TokenB = lib.Exponentiate(30000, 18), lib.Exponentiate(30000, 18)
		}
	}
}

/*
	-----------------------------------------------

dynamically change the side chain's committee with last main chain's leader
the committee nodes is shifted by one and the new leader is added to be used for next epoch's side chain's committee
Note that: for now we are considering the last w distinct leaders in the committee which means
if a leader is selected multiple times during an epoch, he will not be added multiple times,
-----------------------------------------------
*/
func (bz *ChainBoost) UpdateSideChainCommittee() {
	bz.SimulationSeed++ // change the seed for the RNG
	rand := rand.New(rand.NewSource(int64(bz.SimulationSeed)))
	var r []int
	var d int
	for i := 0; i < bz.CommitteeWindow; i++ {
		d = int(math.Abs(float64(int(rand.Uint64())))) % len(bz.Tree().List())
		if !contains(r, d) {
			r = append(r, d)
		} else {
			i--
		}
	}
	bz.CommitteeNodesTreeNodeID = make([]onet.TreeNodeID, 0)
	for i := 0; i < bz.CommitteeWindow; i++ {
		d := r[i]
		bz.CommitteeNodesTreeNodeID = append(bz.CommitteeNodesTreeNodeID, bz.Tree().List()[d].ID)
	}
	for i, a := range bz.CommitteeNodesTreeNodeID {
		log.Lvl1("final result SC: Initial BlsCosi committee queue: ", i, ":", bz.Tree().Search(a).Name())
	}
	bz.NextSideChainLeader = bz.CommitteeNodesTreeNodeID[0]
}

// utility
func contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

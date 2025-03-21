package main

import (
	"ammcore"
	"ammcore/lib"
	"bgls"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/chainBoostScale/ChainBoost/MainAndSideChain/blockchain"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func generateAddress() (string, error) {
	bytes := make([]byte, 20)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("0x%s", hex.EncodeToString(bytes)), nil
}

var hardhatkeys []string = []string{"0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
	"0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d",
	"0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a",
	"0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6",
	"0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a",
	"0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba",
	"0x92db14e403b83dfe3df233f83dfa3a0d7096f21ca9b0d6d6b8d88b2b4ec1564e",
	"0x4bbbf85ce3377467afe5d46f804f221813b2bb87f24d81f60f1fcdbf7cbf4356",
	"0xdbda1821b80551c9d65939329250298aa3472ba22feea921c0cf5d620ea67b97",
	"0x2a871D0798f97d79848a013d4936a73bf4cc922c825d33c1cf7073dff6d409c6",
	"0xf214f2b2cd398c806f84e317254e0f0b801d0643303237d97a22a48e01628897",
	"0x701b615bbdfb9de65240bc28bd21bbc0d996645a3dd57e7b12bc2bdf6f192c82",
	"0xa267530f49f8280200edf313ee7af6b827f2a8bce2897751d06a843f644967b1",
	"0x47c99Abed3324a2707c28affff1267e45918ec8c3f20b8aa892e8b065d2942dd",
	"0xc526eE95bf44d8fc405a158bb884d9d1238d99f0612e9f33d006bb0789009aaa",
	"0x8166f546bab6da521a8369cab06c5d2b9e46670292d85c875ee9ec20e84ffb61",
	"0xea6c44ac03bff858b476bba40716402b03e41b8e97e276d1baec7c37d42484a0",
	"0x689af8efa8c651a91ad287602527f3af2fe9f6501a7ac4b061667b5a93e037fd",
	"0xde9be858da4a475276426320d5e9262ecfc3ba460bfac56360bfa6c4c28b4ee0",
	"0xdf57089febbacf7ba0bc227dafbffa9fc08a93fdc68e1e42411a14efcf23656e"}

func getUsersForBatch(batchSize int, isHardHat bool) []blockchain.UserInfoLite {
	Users := make([]blockchain.UserInfoLite, batchSize)

	for i := 0; i < batchSize; i++ {
		if isHardHat {
			ownerKeyBytes, _ := hex.DecodeString(hardhatkeys[i][2:])
			Users[i].PrivKey, _ = crypto.ToECDSA(ownerKeyBytes)
		} else {
			Users[i].PrivKey, _ = crypto.GenerateKey()
		}
		Users[i].Pubkey = &Users[i].PrivKey.PublicKey
		Users[i].AmmAddress, _ = generateAddress()
	}
	return Users
}

func fund(pubkey *ecdsa.PublicKey, totalUsers int) {
	var url string = "https://eth-sepolia.blastapi.io/aaec52d7-8f95-4c33-a82e-04bcc135aaa1"
	client, err := ethclient.Dial(url)
	if err != nil {
		fmt.Println(err)
	}
	hexString := "" // Replace with your ethreum account with tokens
	ownerKeyBytes, err := hex.DecodeString(hexString)
	if err != nil {
		fmt.Println(err)
	}
	ownerKey, err := crypto.ToECDSA(ownerKeyBytes)
	//ownerKey, err := x509.ParseECPrivateKey(ownerKeyBytes)
	if err != nil {
		fmt.Println(err)
	}
	numerator, _ := new(big.Int).SetString("10000000000000000000", 10)
	value := new(big.Int).Div(numerator, big.NewInt(int64(totalUsers)))
	chainID := big.NewInt(11155111) //Sepolia !
	fromAddress := crypto.PubkeyToAddress(ownerKey.PublicKey)

	nonce := ammcore.GetNonceFromSyncMap(fromAddress)

	gasLimit := uint64(21000)
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		panic(err)
	}
	toAddress := crypto.PubkeyToAddress(*pubkey)
	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, nil)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), ownerKey)
	if err != nil {
		panic(err)
	}
	fmt.Print("Sending: " + value.String() + " wei to " + toAddress.Hex() + " ...")
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		panic(err)
	}
	fmt.Println("OK")

}

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		panic("not enough args")
	}
	if args[0] == "create" || args[0] == "hardhat" {
		batchSize := 1
		totalUsers := 1000
		if args[0] == "hardhat" {
			totalUsers = len(hardhatkeys)
			batchSize = len(hardhatkeys)
		}
		var wg sync.WaitGroup

		for i := 0; i < totalUsers; i += batchSize {
			wg.Add(batchSize)
			fmt.Printf("Creating Users Batches From %d to %d", i, i+batchSize-1)
			users := getUsersForBatch(batchSize, args[0] == "hardhat")
			for _, user := range users {
				go func(user blockchain.UserInfoLite) {
					defer wg.Done()
					if !ammcore.MintTokens(user.PrivKey, lib.Exponentiate(50000000, 18)) {
						panic("Noooooo")
					}
				}(user)
			}
			fmt.Println("Mint Issued")
			wg.Wait()
			fmt.Println("Mint Done")
			blockchain.BatchInsertIntoUsersTable(users)
			time.Sleep(2 * time.Second)
		}
		// wait for all batches to finish
		fmt.Println("All users processed")
	} else if args[0] == "fund" {
		users, _ := blockchain.GetUsers()
		for _, user := range users {
			fund(user.Pubkey, len(users))
		}
	} else {
		curve := bgls.CurveSystem(bgls.Altbn128)
		x, _, X, _ := bgls.KeyGen(curve)
		blockchain.PickleKeys(x.Bytes(), X.Marshal())
	}
}

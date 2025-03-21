
# AmmBoost 

AmmBoost is a sidechain based performance enhancer for Automated Market Maker (AMMs)

## Sources Used:

  * Onet: https://github.com/dedis/onet/tree/v3.2.9 for networking and simulation layers
  * BdnCosi/BLSCoSi: https://github.com/dedis/cothority for consensus
  * Kyber: https://github.com/dedis/kyber for Advanced cryptographic primitives
  * VRF: https://github.com/algorand/go-algorand/pull/2310/
  * go-ethereum.



## Getting Started:

This project needs golang version 1.20 to run successfully.

### Config File

Config File "ChainBoost.toml" is located under the following directory: simulation/manage/simulation/ChainBoost.toml

### On Building using Makefiles

this repo contains three Makefiles, one on the root of the repo, one in `simulation/manage/simulation/Makefile` and one in `simulation/platform/csslab_users/Makefile`

The Makefile in the root of the repo builds all the binaries (simul and users) and puts them in the build folder,
it allows the following commands:

* `make build` : builds the binaries
* `make deploy USER=<user>` : builds the binaries and deploys them to gateway using rsync over SSH
* `make clean`: cleans up all the binaries that were built.

The in-folder Makefiles are helper Makefiles for the one in the root of the repo, and are used to build their respective binaries

### On generating and Ethereum Accounts for the experiment:

client creator allows for creating and funding ethereum accounts for the experiment. 

on the client creator folder build the local source code using `go build .` and run the executable with the two following command line arguments:

  * create: creates ethereum accounts in the network
  * fund: funds the ethereum accounts in token.
  * genkey: to generate a pair of BLS Keys.

### On running the experiment locally:

run `make build`:
    find your correct executable inside `build/simul/<os>/<arch>`
    copy the following files in the same folder as your executable: `Chainboost.toml`, `keys.db`, `users.db`

    run `./simul`

### On Running the exeperiment using the orchestrator:

The orchestrator is a production ready way to orchestrate different instances of the simulation executable. in order to run the experiment:
1) edit `ssh.toml` to match your expected files to be uploaded to the VMs and the ones you want to retrieve after the experiments are over.
2) run `./orchestrator ssh.toml`

Note: The logs of every simul instance are written under the folder `<vm-ip-address>/stdout.txt` , it contains stdout and stderr outputs


## Project Layout ##

`ammboost` is split into various subpackages:

  * [submodule] bgls: an implementation for BLS Signatures in Golang and Solidity
  * [submodule] abstract_amm: an implementation of an AMM based on Uniswap v3 in golang
  *             clientcreator: creates ethereum clients that perform AMM deposits (and operations)
  *             MainAndSidechain: contains the core behaviour for the AMM Sidechain (mainchain code is unused)
  *             orchestrator:  contains the orchestrator to run the distributed experiments.
  *             simulation  :  contains the tools to instantiate and execute the simulation.
  *             utilities   :  contains the utilities used for post-processing the data.
  *             vrf, onet   : helper code implementing VRF, ONet, 








# Time Out
Timeouts are parsed according to Go's time.Duration: A duration string
is a possibly signed sequence of decimal numbers, each with optional
fraction and a unit suffix, such as "300ms", "-1.5h" or "2h45m". Valid
time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

This is a list of timeouts that we want to control or may want to keep static but edit according to our network setting:

### Network Level
- In the overlay layer, there is a GlobalProtocolTO of 10 mins (I increased it from 1 min to be sure it is not causing error!), 
- In the Server file, in TryConn a 2 sec listening TO and a 10 sec TO for getting access to IPs
- In the TCP files, a globalTO of 1 minute (increased now!) for connection TO and a dialTO of 1 minute (increased now!) which the later has a function for changing it.
- A 20 seconds TO for SSH

###  A speciall TimeOut
- A joining TO of 20 sec for trying to invite the nodes to join the simulation.

### Prrotocol(s) Level
- BLSCoSi has a TO which is set to 100 secs and is for waiting for response from the subprotocol

In the `ChainBoost.toml` config file:
- A `RunWait` parameter which shows how many seconds to wait for a run (one line of .toml-file) to finish
- A `timeout` shows how many seconds to wait for the while experiment to finish (default: RunWait \* #Runs)

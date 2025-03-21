module github.com/chainBoostScale/ChainBoost

require (
	filippo.io/edwards25519 v1.0.0
	github.com/BurntSushi/toml v1.2.0
	github.com/algorand/go-algorand v0.0.0-20220610190922-ff1a53d3cf8d
	github.com/daviddengcn/go-colortext v1.0.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.5.0
	github.com/stretchr/testify v1.8.4
	go.dedis.ch/kyber/v3 v3.0.13
	go.dedis.ch/protobuf v1.0.11
	go.etcd.io/bbolt v1.3.6
	golang.org/x/sync v0.7.0
	golang.org/x/sys v0.20.0
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	rsc.io/goversion v1.2.0
)

replace ammcore => ./abstract_amm

replace bgls => ./bgls

require golang.org/x/crypto v0.23.0

require (
	github.com/algorand/go-codec/codec v1.1.8 // indirect
	github.com/algorand/go-deadlock v0.2.2 // indirect
	github.com/algorand/msgp v1.1.52 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/petermattis/goid v0.0.0-20180202154549-b0b1615b78e5 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20200410134404-eec4a21b6bb0 // indirect
	go.dedis.ch/fixbuf v1.0.3 // indirect
	go.uber.org/atomic v1.10.0 // direct
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/term v0.20.0 // indirect
	golang.org/x/tools v0.21.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	lukechampine.com/uint128 v1.1.1 // indirect
	modernc.org/cc/v3 v3.38.1 // indirect
	modernc.org/ccgo/v3 v3.16.9 // indirect
	modernc.org/libc v1.19.0 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.4.0 // indirect
	modernc.org/opt v0.1.3 // indirect
	modernc.org/strutil v1.1.3 // indirect
	modernc.org/token v1.0.1 // indirect
)

require (
	ammcore v0.0.0-00010101000000-000000000000
	bgls v0.0.0-00010101000000-000000000000
	github.com/ethereum/go-ethereum v1.14.0
	github.com/rocketlaunchr/dataframe-go v0.0.0-20211025052708-a1030444159b
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842
	modernc.org/sqlite v1.19.1
)

require (
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/apache/thrift v0.0.0-20181112125854-24918abba929 // indirect
	github.com/bits-and-blooms/bitset v1.10.0 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.2.0 // indirect
	github.com/consensys/bavard v0.1.13 // indirect
	github.com/consensys/gnark-crypto v0.12.1 // indirect
	github.com/crate-crypto/go-kzg-4844 v1.0.0 // indirect
	github.com/dchest/blake2b v1.0.0 // indirect
	github.com/deckarep/golang-set/v2 v2.1.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/ethereum/c-kzg-4844 v1.0.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/golang/snappy v0.0.5-0.20220116011046-fa5810519dcb // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/guptarohit/asciigraph v0.7.1 // indirect
	github.com/holiman/uint256 v1.2.4 // indirect
	github.com/juju/clock v0.0.0-20190205081909-9c5c9712527c // indirect
	github.com/juju/errors v0.0.0-20200330140219-3fe23663418f // indirect
	github.com/juju/loggo v0.0.0-20200526014432-9ce3a2e09b5e // indirect
	github.com/juju/utils/v2 v2.0.0-20200923005554-4646bfea2ef1 // indirect
	github.com/klauspost/compress v1.15.15 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rocketlaunchr/mysql-go v1.1.3 // indirect
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/supranational/blst v0.3.11 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/xitongsys/parquet-go v1.5.2 // indirect
	golang.org/x/net v0.25.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
)

require (
	github.com/briandowns/spinner v1.19.0
	github.com/povsister/scp v0.0.0-20210427074412-33febfd9f13e
	github.com/smartystreets/goconvey v1.7.2 // indirect
	github.com/withmandala/go-log v0.1.0
)

go 1.21.6

toolchain go1.21.9

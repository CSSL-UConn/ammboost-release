// Package platform contains interface and implementation to run onet code
// amongst multiple platforms. Such implementations include Localhost (run your
// test locally) and Csslab (similar to emulab).
package platform

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"sync"

	"github.com/BurntSushi/toml"
	"github.com/chainBoostScale/ChainBoost/onet/log"
	"golang.org/x/xerrors"
)

// The Life of a simulation:
//
// 1. Configure
//     * read configuration
//     * compile eventual files
// 2. Build
//     * builds all files
//     * eventually for different platforms
// 3. Cleanup
//     * send killall to applications
// 4. Deploy
//     * make sure the environment is up and running
//     * copy files
// 5. Start
//     * start all logservers
//     * start all nodes
//     * start all clients
// 6. Wait
//     * wait for the applications to finish

// Platform interface that has to be implemented to add another simulation-
// platform.
type Platform interface {
	// Does the initial configuration of all structures needed for the platform
	Configure([]Config)
	// Build builds all necessary binaries
	Build(build string, arg ...string) error
	// Makes sure that there is no part of the application still running
	Cleanup() error
	// Copies the binaries to the appropriate directory/machines, together with
	// the necessary configuration. RunConfig is a simple string that should
	// be copied as 'app.toml' to the directory where the app resides
	Deploy(*RunConfig) error
	// Starts the application and returns - non-blocking!
	Start(args ...string) error
	// Waits for the application to quit
	Wait() error
}

// Raha: passing system wide configuration: updated the structure
// Config is passed to Platform.Config and prepares the platform for
// specific system-wide configurations
type Config struct {
	// string denoting the group used for simulations
	// XXX find ways to remove that "one suite" assumption
	Suite string
	Debug int
	// : adding some other system-wide configurations, these fields are those who are needed in the protocol
	MCRoundDuration                  int
	PercentageTxPay                  int
	MainChainBlockSize               int
	SideChainBlockSize               int
	SectorNumber                     int
	NumberOfPayTXsUpperBound         int
	NumberOfActiveContractsPerServer int
	SimulationRounds                 int
	SimulationSeed                   int
	// bls cosi

	NbrSubTrees              int
	Threshold                int
	SCRoundDuration          int
	CommitteeWindow          int
	MCRoundPerEpoch          int
	SCRoundPerEpoch          int
	SimState                 int
	StoragePaymentEpoch      int
	PayPercentOfTransactions float64

	//ammInfo
	DailyVolume int
	BlockTime   int
	SwapRate    float64
	BurnRate    float64
	CollectRate float64
	MintRate    float64
	UsersNbr    int
	DepositMode int
}

var csslab = "csslab"
var localhost = "localhost"
var mininet = "mininet"
var cssl = "cssl"

// NewPlatform returns the appropriate platform
// [csslab,localhost]
func NewPlatform(t string) Platform {
	var p Platform
	switch t {
	case csslab:
		p = &Csslab{}
	case localhost:
		p = &Localhost{}
	case cssl:
		p = &Cssl{}
	}
	return p
}

// ReadRunFile reads from a configuration-file for a run. The configuration-file has the
// following syntax:
// Name1 = value1
// Name2 = value2
// [empty line]
// n1, n2, n3, n4
// v11, v12, v13, v14
// v21, v22, v23, v24
//
// The Name1...Namen are global configuration-options.
// n1..nn are configuration-options for one run
// Both the global and the run-configuration are copied to both
// the platform and the app-configuration.
func ReadRunFile(p Platform, filename string) []*RunConfig {
	var runconfigs []*RunConfig
	masterConfig := NewRunConfig()
	log.Lvl1("Reading file", filename)

	file, err := os.Open(filename)
	defer func() {
		if err := file.Close(); err != nil {
			log.Error("Couldn' close", file.Name())
		}
	}()
	if err != nil {
		log.Fatal("Couldn't open file", filename, err)
	}

	// Decoding of the first part of the run config file
	// where the config wont change for the whole set of the simulation's tests
	scanner := bufio.NewScanner(file)
	line := 0
	for scanner.Scan() {
		text := scanner.Text()
		line++
		//log.LLvl3("Decoding", text)
		// end of the first part
		if text == "" {
			break
		}
		if text[0] == '#' {
			continue
		}

		// checking if format is good
		vals := strings.Split(text, "=")
		if len(vals) != 2 {
			log.Fatal("Simulation file:", filename, "line", line, "is not properly formatted ( key = value )")
		}
		// fill in the general config
		masterConfig.Put(strings.TrimSpace(vals[0]), strings.TrimSpace(vals[1]))
		// also put it in platform
		if _, err := toml.Decode(text, p); err != nil {
			log.Error("Error decoding", text)
		}
		//log.LLvl3("Platform is now %+v", p)
	}

	for scanner.Scan() {
		if scanner.Text() != "" && scanner.Text()[0] != '#' {
			break
		}
	}
	args := strings.Split(scanner.Text(), ",")
	for scanner.Scan() {
		if len(scanner.Text()) == 0 || scanner.Text()[0] == '#' {
			continue
		}
		rc := masterConfig.Clone()
		// put each individual test configs
		for i, value := range strings.Split(scanner.Text(), ",") {
			rc.Put(strings.TrimSpace(args[i]), strings.TrimSpace(value))
		}
		runconfigs = append(runconfigs, rc)
	}

	return runconfigs
}

// RunConfig is a struct that represent the configuration to apply for one "test"
// Note: a "simulation" is a set of "tests"
type RunConfig struct {
	fields map[string]string
	sync.RWMutex
}

// NewRunConfig returns an initialised config to be used for reading
// in runconfig-files
func NewRunConfig() *RunConfig {
	rc := new(RunConfig)
	rc.fields = make(map[string]string)
	return rc
}

// One problem for now is RunConfig read also the ' " ' char (34 ASCII)
// and thus when doing Get(), also return the value enclosed by ' " '
// One fix is to each time we Get(), automatically delete those chars
var replacer = strings.NewReplacer("\"", "", "'", "")

// Get returns the associated value of the field in the config
func (r *RunConfig) Get(field string) string {
	r.RLock()
	defer r.RUnlock()
	return replacer.Replace(r.fields[strings.ToLower(field)])
}

// Delete a field from the runconfig (delete for example Simulation which we
// dont care in the final csv)
func (r *RunConfig) Delete(field string) {
	r.Lock()
	defer r.Unlock()
	delete(r.fields, field)
}

// ErrorFieldNotPresent signals that a field is not in the RunConfig.
var ErrorFieldNotPresent = xerrors.New("field not present")

// GetInt returns the integer of the field, or error if not defined
func (r *RunConfig) GetInt(field string) (int, error) {
	val := r.Get(field)
	if val == "" {
		return 0, ErrorFieldNotPresent
	}
	ret, err := strconv.Atoi(val)
	return ret, err
}

// GetDuration returns the field parsed as a duration, or error if a parse error occurs.
func (r *RunConfig) GetDuration(field string) (time.Duration, error) {
	val := r.Get(field)
	if val == "" {
		return 0, ErrorFieldNotPresent
	}
	return time.ParseDuration(val)
}

// GetBuckets returns the list of buckets defined in the configuration
// file to split the statistics
func (r *RunConfig) GetBuckets() ([][]string, error) {
	val := r.Get("buckets")
	if val == "" {
		return nil, ErrorFieldNotPresent
	}

	bb := [][]string{}
	for _, b := range strings.Fields(val) {
		bb = append(bb, strings.Split(b, "-"))
	}

	return bb, nil
}

// Put inserts a new field - value relationship
func (r *RunConfig) Put(field, value string) {
	r.Lock()
	defer r.Unlock()
	r.fields[strings.ToLower(field)] = value
}

// Toml returns this config as bytes in a Toml format
func (r *RunConfig) Toml() []byte {
	r.RLock()
	defer r.RUnlock()
	var buf bytes.Buffer
	for k, v := range r.fields {
		fmt.Fprintf(&buf, "%s = %s\n", k, v)
	}
	return buf.Bytes()
}

// Map returns this config as a Map
func (r *RunConfig) Map() map[string]string {
	r.RLock()
	defer r.RUnlock()
	tomap := make(map[string]string)
	for k := range r.fields {
		tomap[k] = r.Get(k)
	}
	return tomap
}

// Clone this runconfig so it has all fields-value relationship already present
func (r *RunConfig) Clone() *RunConfig {
	r.RLock()
	defer r.RUnlock()
	rc := NewRunConfig()
	for k, v := range r.fields {
		rc.fields[k] = v
	}
	return rc
}

func (r *RunConfig) String() string {
	r.RLock()
	defer r.RUnlock()
	return fmt.Sprintf("%-v", r.fields)
}

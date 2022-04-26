# Lumerin

Lumerin Node (aka ProxyRouter)

# Setup with Ganache Test Blockchain
1. Install Go https://go.dev/dl/
2. Install and Run Ganache<br/>
    a) https://trufflesuite.com/ganache/ // GUI<br/>
    b) https://github.com/trufflesuite/ganache // headless<br/>
    c) Click "quickstart" in Ganache to run test blockchain
3. Clone repo
4. Copy `lumerinconfig.example.json` to `lumerinconfig.json`
5. `cd` into `cmd` directory
6. Run `go build -o $GOPATH/bin/lumerin` // builds binary
7. `cd` into `cmd/contractmanager`
8. Run `go test -run Deployment` // will deploy contracts to Ganache
9. Edit `lumerinconfig.json`<br/>
    a) "mnemonic" should be the one generated in Ganache<br/>
    b) "lumerinTokenAddress" will be generated from Deployment test<br/>
    c) "cloneFactoryAddress" will be generated from Deployment test<br/>
    d) "defaultPoolAddr" is the stratum address of the default pool<br/>
    e) "listenIP" and "listenPort" will be the address and port the lumerin node is listening on<br/> 
    f) "passthrough" should be set to true for POC use<br/>
    g) "disable" should be set to true for any subsystems to be ignored for a given run<br/>
10. Edit `run_lumerin.sh` (optional: config file params will take priority over flag params so leave configfile flag to empty if using config flags)
11. Run `./run_lumerin.sh`

# Setup with Ropsten Testnet
1. Install Go https://go.dev/dl/
2. Clone repo
3. Copy `lumerinconfig.example.json` to `lumerinconfig.json`
4. Edit `lumerinconfig.json`<br/>
    a) "mnemonic" and "account index" should be the same account used for creating hashrate contracts in the marketplace<br/>
    b) "lumerinTokenAddress" on Ropsten is: "0xC6a30Bc2e1D7D9e9FFa5b45a21b6bDCBc109aE1B"<br/>
    c) "cloneFactoryAddress" on Ropsten is: "0xe91be01493f4ae28297790277303926aaec604dc"<br/>
    d) "defaultPoolAddr" is the stratum address of the default pool<br/>
    e) "listenIP" and "listenPort" will be the address and port the lumerin node is listening on<br/> 
    f) "passthrough" should be set to true for POC use<br/>
    g) "disable" should be set to true for any subsystems to be ignored for a given run<br/>
5. Edit `run_lumerin.sh` (optional: config file params will take priority over flag params so leave configfile flag to empty if using config flags)
6. Run `./run_lumerin.sh`

License
MIT
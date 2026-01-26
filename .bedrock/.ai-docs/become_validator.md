# proxyrouter Validator 
* Become a Validator: https://github.com/Lumerin-protocol/proxy-router/tree/dev?tab=readme-ov-file#validator-node

## ENVIRONMENT OVERVIEW

### TESTNET/DEV: 
* validator: `bugs.dev.lumerin.io:3333`
* `mssh -t i-0630f430df1485bf5 -u titanio-dev titanadmin@bugs-int.dev.lumerin.io`
* wallet: `0xb8F836C167d60e20e44BAf62d4d46c9E26Fea97F`
* eth_node: 
* contracts: 
	- saLMR token:   `0xC27DafaD85F199FD50dD3FD720654875D6815871`
	- clone factory: `0x15437978300786aDe37f61e02Be1C061e51353D3`
	- validator reg: `0xD81265c55ED9Ca7C71d665EA12A40e7921EA1123`
* Validator Staking: `1000000000000` (10,000 aLMR)`
* Validator Punishment (Per infraction): `100000000000` (1,000 aLMR)`

### MAINNET/LMN 
* validator: `coyote.lumerin.io:3333`
* `mssh -t i- -u titanio-lmn titanadmin@coyote-int.lumerin.io`
* wallet: `0x65bBb982d9B0AfE9AED13E999B79c56dDF9e04fC`
* eth_node: 
* contracts: 
	* aLMR Token:    `0x0FC0c323Cf76E188654D63D62e668caBeC7a525b`
	* clone:factory: `0x05C9F9E9041EeBCD060df2aee107C66516E2b9bA`
	* validator reg: `0xbEB5b2df7B554Fb175e97Eb21eE1e8D7fF2f56B1`
* Validator Staking: `100000000000000` (1,000,000 aLMR)`
* Validator Punishment (Per infraction): `10000000000000` (100,000 aLMR)`

## PROCESS 

1. Setup standard EC2 Instance 
	* dev - bugs
	* stg - roadrunner
	* lmn - coyote 

2. install go 
	```bash 
    wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz
	sudo rm -rf /usr/local/go
	sudo tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
	export PATH=$PATH:/usr/local/go/bin
    echo "export PATH=\$PATH:/usr/local/go/bin" >> ~/.bashrc
	source ~/.bashrc
	go version
    ```

3. Clone Repo: 
    ```bash 
    git clone -b main https://github.com/Lumerin-protocol/proxy-router.git
	cd proxy-router 
    ```

4. Edit .env file (See below)
	* cd ~/proxy-router
	* cp .env.example .env 
	* vi .env 

5. Build ProxyRouter: Should output `~/proxy-router/bin/proxyrouter`
	```bash 
    cd ~/proxy-router 
	go mod tidy 
	./build.sh 
	```

6. Shorten the Public Key ... needed for setup validator on the contract
	* `./bin/proxyrouter pubkey` 

7. Approve aLMR spend from Validator Wallet to Contract 
	* as validator wallet address
	* go to lumerin contract address: 
		- TEST - [0xC27DafaD85F199FD50dD3FD720654875D6815871](https://sepolia.arbiscan.io/address/0xC27DafaD85F199FD50dD3FD720654875D6815871#writeProxyContract)
		- MAIN - [0x0FC0c323Cf76E188654D63D62e668caBeC7a525b](https://arbiscan.io/address/0x0FC0c323Cf76E188654D63D62e668caBeC7a525b#writeProxyContract)
	* approve validator_registry contract:  
		- DEV - `0xD81265c55ED9Ca7C71d665EA12A40e7921EA1123`
		- STG - `0xdA7EB6D64F5B8E38735ce424E3fA81B25d8a2a7e`
        - LMN - `0xbEB5b2df7B554Fb175e97Eb21eE1e8D7fF2f56B1`
    * to spend 100000000000 (1000 saLMR)

8. Register Validator: 
    - DEV [0xD81265c55ED9Ca7C71d665EA12A40e7921EA1123] (https://sepolia.arbiscan.io/address/0xD81265c55ED9Ca7C71d665EA12A40e7921EA1123#writeProxyContract)
    - STG [0xdA7EB6D64F5B8E38735ce424E3fA81B25d8a2a7e] (https://arbiscan.io/address/0xdA7EB6D64F5B8E38735ce424E3fA81B25d8a2a7e#writeProxyContract)
    - LMN [0xbEB5b2df7B554Fb175e97Eb21eE1e8D7fF2f56B1] (https://arbiscan.io/address/0xbEB5b2df7B554Fb175e97Eb21eE1e8D7fF2f56B1#writeProxyContract)
    * `validatorRegiester`:
        * Stake - 1000000000000 (10,000 aLMR ... with 8 zeros)
        * yParty: (parity from short pubkey in above)  
        * pubKeyX: (short pubkey from above)
        * host: host:port (eg: host.dev.domain.com:1234 or 192.98.4.56:5678)

8. ALT - Register with hardhat 
    * From Repo root  
    * ensure root .env file is correct
    `VALIDATOR_PRKEY="privatekey" VALIDATOR_HOST="host:port" VALIDATOR_STAKE="10000000000000" yarn hardhat --config hardhat.config.ts --network default run ./scripts/register-validator.ts` 

9. Setup Service 
	* On EC2 instance (after .env config and proxy-router setup (assume all in /home/titanadmin/proxy-router) )S
	* Create `setup_proxyrouter.sh` in home root
	* chmod +x setup_proxyrouter.sh 
	* sudo su (be root)
	* execute ./setup_proxyrouter.sh

## HELPER FILES
1. [Base Installation Script](/.terragrunt/scripts/01-baseinstall_proxyrouter.sh)
	- `vi ~/baseinstall.sh` 
	- copy contents and save 
	- `chmod +x ~/baseinstall.sh`
	- take note of clone branch (default to main)
	- execute as titanadmin user `./baseinstall.sh`
1. Copy .env file from `/.terragrunt/scripts/` in this repo to `~/proxy-router/.env`
	- `vi ~/proxy-router/.env`
	- update with correct values
	- copy contents and save
1. [Setup Proxy Router Service Script](/.terragrunt/scripts/02-setup_proxyrouter.sh)
	- `vi ~/setup_proxyrouter.sh`
	- copy contents and save
	- `chmod +x ~/setup_proxyrouter.sh`
	- `sudo su`
	- `./setup_proxyrouter.sh` must be exectued as root 
	- Confirm logs look good: `journalctl -u proxyrouter -f`
1. Register the validator per instructions 
 	- Restart the service: `sudo systemctl restart proxyrouter`
	- Confirm logs look good: `journalctl -u proxyrouter -f`

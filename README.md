# Lumerin Node Proxy-Router

Hashrouter node that can route inbound hashrate to default Seller pool destination or, when contracts are purchased, to the Buyer pool destination.

The Proxy-Router can be utilized in three different modes:

- As a Seller of hashrate
- As a Buyer of hashrate
- As a Validator of hashrate from the Web Based Marketplace

## Seller Node
### For full details, start here: [Lumerin Gitbook](https://gitbook.lumerin.io/lumerin-hashpower-marketplace/seller/partner-preview/1.0-prerequisites)
1. Download the latest version of the Proxy-Router from the [Lumerin Github](https://github.com/Lumerin-protocol/proxy-router/releases) for your platform
   1. Rename to `proxy-router` or `proxy-router.exe` for Windows (this is only for simplicity in the instructions)
   1. Linux/Mac - `chmod +x proxy-router`
   1. Mac - `xattr -c proxy-router` 
1. Create an .env file using the `.env.min.example` file as a template in the same directory as the `proxy-router` binary
   1. Be sure to watch the Mainnet/Testnet settings
1. Start the proxy-router 
   1. Linux/Mac - `./proxy-router`
   1. Windows - `proxy-router.exe`
1. The proxy-router will start
1. Redirect your ASICS to the proxy-router's IP address and port (default 3333)
   1. The proxy-router will now route un-contracted hashrate to the default Seller pool destination configured in the .env file
1. To view the proxy-router's status, use the API Endpoints: 
   1. `http://localhost:8080/healthcheck` - For version and uptime 
   1. `http://localhost:8080/config` - For configuration details 
   1. `http://localhost:8080/miners` - To see inbound miner stats
   1. `http://localhost:8080/contracts-v2` - To see contract stats and logs
1. Setup Contracts 
   1. Download the [Lumerin Desktop Wallet](https://github.com/Lumerin-protocol/WalletDesktop/releases/tag/latest) file for your platform
      1. If you want to run on Mainnet - choose `latest` release without a suffix 
      1. If you want to run on Testnet - choose the most recent `*-stg` release
      1. Mac - 
         1. Download the arm64.dmg (for Apple Silicon) or x64.dmg (for Intel) file
         1. Open the .dmg file and drag the Lumerin Wallet to the Applications folder
         1. Open the Lumerin Wallet from the Applications folder
      1. Linux -
         1. Download the .deb file
         1. Run `sudo dpkg -i lumerin-wallet-*.deb`
         1. Run `lumerin-wallet` from the terminal
      1. Windows -
         1. Download the .exe file
         1. Run the .exe file
         1. Defender may complain, so you may need to click on "More Info" and "Run Anyway"
   1. Create a import an existing wallet using the mnemonic or private key of your Seller Node
      1. When asked, make sure to `RUN WITHOUT PROXY` as this will be the control / manager for your Seller Node
   1. Go to the Seller Hub 
   1. Click Create Contract 
   1. Enter Details and Clice Create Contract

## Validator Node

### How to register as a validator node
#### For full details, visit the [Lumerin Gitbook](https://gitbook.lumerin.io/lumerin-hashpower-marketplace/validator/)
1. Download the latest version of the Proxy-Router from the [Lumerin Github](https://github.com/Lumerin-protocol/proxy-router/releases)
1. Enter your validator node wallet private key as WALLET_PRIVATE_KEY environment variable
1. Run the Proxy-Router with the following command to generate compressed public key
   ```bash
   ./proxy-router pubkey
   ```
1. Fill in the rest of configuration in .env file and start the validator node with the following command
   ```bash
   ./proxy-router
   ```
1. Open the [validator registry smart-contract](https://sepolia.arbiscan.io/address/0xD81265c55ED9Ca7C71d665EA12A40e7921EA1123)
1. Click on the "Write as Proxy" tab and click on the "validatorRegister" function
1. Enter your stake, pubKeyYparity, pubKeyX (generated in previous steps) and the host where the validator is running as `hostname:port` (example: `my-validator.com:8080` or `165.122.1.1:3333`)
1. Make sure you have enough ETH to pay tx fees, enough LMR to stake and LMR is approved for the contract for the stake amount
1. Click on "Write" and confirm the transaction


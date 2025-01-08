# Lumerin Node Proxy-Router

Hashrouter node that can route inbound hashrate to default Seller pool destination or, when contracts are purchased, to the Buyer pool destination.

The Proxy-Router can be utilized in three different modes:

- As a Seller of hashrate
- As a Buyer of hashrate
- As a Validator of hashrate from the Web Based Marketplace

## Validator Node

### How to register as a validator node

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

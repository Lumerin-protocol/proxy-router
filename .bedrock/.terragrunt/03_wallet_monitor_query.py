########### WALLET BALANCE MONITOR ###########
# Monitors ETH, USDC, and LMR balances for specified wallets on Arbitrum
# Publishes metrics to CloudWatch with wallet name dimensions for alerting

import urllib.request
import json
import boto3
import os
import time
from datetime import datetime

########### CONFIGURATION ###########
# Get environment variables
eth_chain_id = os.environ.get("ETH_CHAIN", "42161")  # Default to Arbitrum One
eth_api_key = os.environ.get("ETH_API_KEY", "")
cw_namespace = os.environ.get("CW_NAMESPACE", "wallet-monitor")
region_name = os.environ.get("REGION_NAME", "us-east-1")

# Token contract addresses by chain ID
# These are the well-known token addresses for Arbitrum
TOKEN_ADDRESSES = {
    # Arbitrum One (mainnet)
    "42161": {
        "lmr": os.environ.get("LMR_TOKEN_ADDRESS", "0xaf5db6e1cc585ca312e8c8f7c499033590cf5c98"),
        "usdc": os.environ.get("USDC_TOKEN_ADDRESS", "0xaf88d065e77c8cC2239327C5EDb3A432268e5831"),
    },
    # Arbitrum Sepolia (testnet)
    "421614": {
        "lmr": os.environ.get("LMR_TOKEN_ADDRESS", "0x0000000000000000000000000000000000000000"),  # Test token
        "usdc": os.environ.get("USDC_TOKEN_ADDRESS", "0x75faf114eafb1BDbe2F0316DF893fd58CE46AA4d"),  # Circle test USDC
    }
}

# Token decimals
TOKEN_DECIMALS = {
    "eth": 18,
    "lmr": 8,
    "usdc": 6
}


########### ETHERSCAN API V2 FUNCTIONS ###########
def get_eth_balance(wallet_address):
    """Get ETH balance for a wallet using Etherscan API V2"""
    base_url = f"https://api.etherscan.io/v2/api?chainid={eth_chain_id}&module=account&action=balance&address={wallet_address}&tag=latest&apikey={eth_api_key}"
    return _make_api_request(base_url, "ETH balance")


def get_token_balance(wallet_address, token_address, token_name):
    """Get ERC-20 token balance for a wallet using Etherscan API V2"""
    if token_address == "0x0000000000000000000000000000000000000000":
        return "0"
    
    base_url = f"https://api.etherscan.io/v2/api?chainid={eth_chain_id}&module=account&action=tokenbalance&contractaddress={token_address}&address={wallet_address}&tag=latest&apikey={eth_api_key}"
    return _make_api_request(base_url, f"{token_name} balance")


def _make_api_request(url, description):
    """Make API request with retry logic for rate limiting"""
    max_retries = 3
    retry_delay = 0.3
    
    for attempt in range(max_retries):
        try:
            req = urllib.request.Request(url)
            with urllib.request.urlopen(req, timeout=10) as response:
                if response.getcode() == 200:
                    data = json.load(response)
                    
                    # Check for API error
                    if data.get('status') == '0':
                        error_msg = data.get('result', 'Unknown error')
                        print(f"API Error fetching {description}: {error_msg}")
                        if "rate limit" in error_msg.lower() and attempt < max_retries - 1:
                            time.sleep(retry_delay)
                            retry_delay *= 2
                            continue
                        return "0"
                    
                    # Success
                    if 'result' in data and str(data['result']).isdigit():
                        return data['result']
                    return "0"
                else:
                    print(f"HTTP error fetching {description}: {response.getcode()}")
                    return "0"
                    
        except urllib.error.HTTPError as e:
            print(f"HTTPError fetching {description} (attempt {attempt + 1}): {e}")
            if attempt < max_retries - 1:
                time.sleep(retry_delay)
                retry_delay *= 2
        except Exception as e:
            print(f"Error fetching {description}: {e}")
            return "0"
    
    return "0"


def convert_balance(raw_balance, decimals):
    """Convert raw balance string to float with proper decimals"""
    try:
        return round(int(raw_balance or "0") / (10 ** decimals), 8)
    except (ValueError, TypeError):
        return 0.0


########### WALLET DATA FUNCTIONS ###########
def get_wallet_balances(wallet_name, wallet_address):
    """Get all balances for a single wallet"""
    print(f"\n--- Fetching balances for {wallet_name} ({wallet_address[:10]}...) ---")
    
    # Get token addresses for current chain
    tokens = TOKEN_ADDRESSES.get(eth_chain_id, TOKEN_ADDRESSES["42161"])
    
    # Fetch ETH balance
    eth_raw = get_eth_balance(wallet_address)
    eth_balance = convert_balance(eth_raw, TOKEN_DECIMALS["eth"])
    time.sleep(0.25)  # Rate limit protection
    
    # Fetch LMR balance
    lmr_raw = get_token_balance(wallet_address, tokens["lmr"], "LMR")
    lmr_balance = convert_balance(lmr_raw, TOKEN_DECIMALS["lmr"])
    time.sleep(0.25)
    
    # Fetch USDC balance
    usdc_raw = get_token_balance(wallet_address, tokens["usdc"], "USDC")
    usdc_balance = convert_balance(usdc_raw, TOKEN_DECIMALS["usdc"])
    
    return {
        "wallet_name": wallet_name,
        "wallet_address": wallet_address,
        "eth_balance": eth_balance,
        "lmr_balance": lmr_balance,
        "usdc_balance": usdc_balance
    }


########### CLOUDWATCH FUNCTIONS ###########
def publish_wallet_metrics(cloudwatch, wallet_data):
    """Publish metrics for a single wallet to CloudWatch"""
    wallet_name = wallet_data["wallet_name"]
    
    # Create metric data with wallet name dimension
    metric_data = [
        {
            "MetricName": "eth_balance",
            "Value": wallet_data["eth_balance"],
            "Unit": "None",
            "Dimensions": [
                {"Name": "WalletName", "Value": wallet_name}
            ]
        },
        {
            "MetricName": "lmr_balance",
            "Value": wallet_data["lmr_balance"],
            "Unit": "None",
            "Dimensions": [
                {"Name": "WalletName", "Value": wallet_name}
            ]
        },
        {
            "MetricName": "usdc_balance",
            "Value": wallet_data["usdc_balance"],
            "Unit": "None",
            "Dimensions": [
                {"Name": "WalletName", "Value": wallet_name}
            ]
        }
    ]
    
    cloudwatch.put_metric_data(
        Namespace=cw_namespace,
        MetricData=metric_data
    )
    
    print(f"Published metrics for {wallet_name}")


def publish_aggregate_metrics(cloudwatch, all_wallet_data):
    """Publish aggregate metrics across all wallets"""
    total_eth = sum(w["eth_balance"] for w in all_wallet_data)
    total_lmr = sum(w["lmr_balance"] for w in all_wallet_data)
    total_usdc = sum(w["usdc_balance"] for w in all_wallet_data)
    
    metric_data = [
        {
            "MetricName": "total_eth_balance",
            "Value": total_eth,
            "Unit": "None"
        },
        {
            "MetricName": "total_lmr_balance",
            "Value": total_lmr,
            "Unit": "None"
        },
        {
            "MetricName": "total_usdc_balance",
            "Value": total_usdc,
            "Unit": "None"
        },
        {
            "MetricName": "wallets_monitored",
            "Value": len(all_wallet_data),
            "Unit": "Count"
        }
    ]
    
    cloudwatch.put_metric_data(
        Namespace=cw_namespace,
        MetricData=metric_data
    )
    
    print(f"\nPublished aggregate metrics: ETH={total_eth}, LMR={total_lmr}, USDC={total_usdc}")


########### LAMBDA HANDLER ###########
def lambda_handler(event, context):
    """Main Lambda handler - fetches wallet balances and publishes to CloudWatch"""
    current_time = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    print(f"\n{'='*60}")
    print(f"Wallet Monitor Execution: {current_time}")
    print(f"Chain ID: {eth_chain_id}")
    print(f"CloudWatch Namespace: {cw_namespace}")
    print(f"{'='*60}")
    
    # Parse wallets from environment variable (JSON string)
    try:
        wallets_json = os.environ.get("WALLETS_TO_WATCH", "[]")
        wallets = json.loads(wallets_json)
    except json.JSONDecodeError as e:
        print(f"Error parsing WALLETS_TO_WATCH: {e}")
        return {
            "statusCode": 500,
            "body": f"Error parsing wallet configuration: {e}"
        }
    
    if not wallets:
        print("No wallets configured for monitoring")
        return {
            "statusCode": 200,
            "body": "No wallets configured"
        }
    
    print(f"\nMonitoring {len(wallets)} wallet(s):")
    for w in wallets:
        print(f"  - {w.get('walletName', 'Unknown')}: {w.get('walletId', 'N/A')[:16]}...")
    
    # Initialize CloudWatch client
    cloudwatch = boto3.client("cloudwatch", region_name=region_name)
    
    # Fetch and publish metrics for each wallet
    all_wallet_data = []
    
    for wallet in wallets:
        wallet_name = wallet.get("walletName", "Unknown")
        wallet_address = wallet.get("walletId", "")
        
        if not wallet_address or wallet_address == "0x0000000000000000000000000000000000000000":
            print(f"Skipping invalid wallet: {wallet_name}")
            continue
        
        try:
            wallet_data = get_wallet_balances(wallet_name, wallet_address)
            all_wallet_data.append(wallet_data)
            
            # Publish individual wallet metrics
            publish_wallet_metrics(cloudwatch, wallet_data)
            
            # Small delay between wallets to respect rate limits
            time.sleep(0.5)
            
        except Exception as e:
            print(f"Error processing wallet {wallet_name}: {e}")
            continue
    
    # Publish aggregate metrics
    if all_wallet_data:
        publish_aggregate_metrics(cloudwatch, all_wallet_data)
    
    # Print summary
    print(f"\n{'='*60}")
    print("WALLET BALANCE SUMMARY")
    print(f"{'='*60}")
    print(f"{'Wallet':<20} {'ETH':>12} {'LMR':>15} {'USDC':>12}")
    print(f"{'-'*60}")
    
    for w in all_wallet_data:
        print(f"{w['wallet_name']:<20} {w['eth_balance']:>12.6f} {w['lmr_balance']:>15.4f} {w['usdc_balance']:>12.4f}")
    
    print(f"{'-'*60}")
    total_eth = sum(w["eth_balance"] for w in all_wallet_data)
    total_lmr = sum(w["lmr_balance"] for w in all_wallet_data)
    total_usdc = sum(w["usdc_balance"] for w in all_wallet_data)
    print(f"{'TOTAL':<20} {total_eth:>12.6f} {total_lmr:>15.4f} {total_usdc:>12.4f}")
    print(f"{'='*60}\n")
    
    return {
        "statusCode": 200,
        "body": json.dumps({
            "message": "Wallet metrics published to CloudWatch",
            "wallets_processed": len(all_wallet_data),
            "timestamp": current_time
        })
    }


# For local testing
if __name__ == "__main__":
    # Set test environment variables
    os.environ["ETH_CHAIN"] = "42161"
    os.environ["CW_NAMESPACE"] = "wallet-monitor-test"
    os.environ["WALLETS_TO_WATCH"] = json.dumps([
        {"walletName": "TestWallet", "walletId": "0x344C98E25F981976215669E048ECcb21be16aC8e"}
    ])
    
    # Note: ETH_API_KEY must be set in environment for actual API calls
    
    result = lambda_handler({}, {})
    print(result)

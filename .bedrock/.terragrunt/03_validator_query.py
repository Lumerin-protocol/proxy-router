########### IMPORTS ###########
import urllib.request
import json
import boto3
import os
import time
from datetime import datetime

# Get the current date and time
now = datetime.now()
current_time = now.strftime("%Y-%m-%d %H:%M:%S")

######### SET ENVIRONMENT VARIABLES #########
api_url = os.environ["API_URL"]
eth_chain_id = os.environ["ETH_CHAIN"]  # Chain ID: 42161 for Arbitrum One, 421614 for Arbitrum Sepolia
eth_api_key = os.environ["ETH_API_KEY"]
cw_namespace = os.environ["CW_NAMESPACE"]
cw_metric1 = os.environ["CW_METRIC1"]
cw_metric2 = os.environ["CW_METRIC2"]
cw_metric3 = os.environ["CW_METRIC3"]
cw_metric4 = os.environ["CW_METRIC4"]
cw_metric5 = os.environ["CW_METRIC5"]
cw_metric6 = os.environ["CW_METRIC6"]
cw_metric7 = os.environ["CW_METRIC7"]
cw_metric8 = os.environ["CW_METRIC8"]
cw_metric9 = os.environ["CW_METRIC9"]
cw_metric10 = os.environ["CW_METRIC10"]
cw_metric11 = os.environ["CW_METRIC11"]
cw_metric12 = os.environ["CW_METRIC12"]
cw_metric13 = os.environ["CW_METRIC13"]

# Get Healthcheck data from /healthcheck api - returns 3 values (status, uptime, version)    
def get_healthcheck_data():
    api_url = os.environ["API_URL"]
    base_url = f"http://{api_url}/healthcheck"     
    base_req = urllib.request.Request(base_url)  
    try:
        response = urllib.request.urlopen(base_req)
        healthcheck_data = json.loads(response.read().decode())
        if healthcheck_data:  
            healthcheck_status = healthcheck_data["status"]
            healthcheck_uptime = healthcheck_data["uptime"]
            healthcheck_version = healthcheck_data["version"]
            return healthcheck_status, healthcheck_uptime, healthcheck_version
        else:
            print("No contract data found in the response.")
            return None, None, None     

    except urllib.error.HTTPError as e:
            print(f"Error occurred while querying contract data: {e}")
            return None, None, None 

# Get Config data from /config api - returns 6 values (version, commit, wallet, lumerin, usdc, clonefactory)   
def get_config_data():
    api_url = os.environ["API_URL"]
    base_url = f"http://{api_url}/config"     
    base_req = urllib.request.Request(base_url)  
    try:
        response = urllib.request.urlopen(base_req)
        config_data = json.loads(response.read().decode())
        if config_data:  
            config_version = config_data["Version"]
            config_commit = config_data["Commit"]
            config_wallet = config_data["DerivedConfig"]["WalletAddress"]
            config_lumerin = "0x0000000000000000000000000000000000000000" #config_data["DerivedConfig"]["FeeTokenAddress"]     
            config_usdc = "0x0000000000000000000000000000000000000000" #config_data["DerivedConfig"]["PaymentTokenAddress"]    
            config_clonefactory = config_data["Config"]["Marketplace"]["CloneFactoryAddress"]

            return config_version, config_commit, config_wallet, config_lumerin, config_usdc, config_clonefactory
        else:
            print("No contract data found in the response.")
            return None, None, None, None, None, None     

    except urllib.error.HTTPError as e:
            print(f"Error occurred while querying contract data: {e}")
            return None, None, None, None, None, None  
    
# Get Overall Contract data from /contracts-v2 api - returns 8 values (seller_hashrate_offered, seller_hashrate_purchased, seller_contracts_offered, seller_contracts_active, validator_hashrate_purchased, validator_hashrate_actual, validator_contracts_active, buyers_unique)
def get_contractsv2_data():
    api_url = os.environ["API_URL"]
    base_url = f"http://{api_url}/contracts-v2"     
    base_req = urllib.request.Request(base_url)  
    try:
        response = urllib.request.urlopen(base_req)
        contractsv2_data = json.loads(response.read().decode())
        if contractsv2_data:  
            # Get Seller Header data 
            seller_contracts_active = contractsv2_data["SellerTotal"]["RunningNumber"]
            seller_hashrate_purchased = round((contractsv2_data["SellerTotal"]["RunningActualGHS"] / 10**6), 4)
            # Get Contracts Validator Header data
            validator_contracts_active = contractsv2_data["ValidatorTotal"]["Number"]
            validator_hashrate_purchased = round((contractsv2_data["ValidatorTotal"]["HashrateGHS"] / 10**6),4)
            validator_hashrate_actual = round((contractsv2_data ["ValidatorTotal"]["ActualHashrateGHS"] / 10**6),4)

            # Contracts Section and Summary 
            contracts = contractsv2_data["Contracts"]
            if contracts:
                # Intialize contract summary variables  
                seller_hashrate_offered = 0
                seller_contracts_offered = 0
                buyers_unique = 0
                buyers = set()
                # Loop through each contract in the dataset
                for contract in contracts:                
                    seller_hashrate_offered = round((sum(contract["ResourceEstimatesTarget"]["hashrate_ghs"] for contract in contracts if not contract["IsDeleted"]) / 10**6),4)
                    seller_contracts_offered = len([contract for contract in contracts if not contract["IsDeleted"]])
                    buyer = contract["BuyerAddr"]
                    if buyer not in buyers and buyer != "":                
                        buyers.add(buyer)
                    buyers_unique = len(buyers)
            else: 
                seller_hashrate_offered = 0
                seller_contracts_offered = 0
                buyers_unique = 0

            return seller_hashrate_offered, seller_hashrate_purchased, seller_contracts_offered, seller_contracts_active, validator_hashrate_purchased, validator_hashrate_actual, validator_contracts_active, buyers_unique  

        else:
            print("No contract data found in the response.")
            return 0, 0, 0, 0, 0, 0, 0, 0    

    except urllib.error.HTTPError as e:
            print(f"Error occurred while querying contract data: {e}")
            return 0, 0, 0, 0, 0, 0, 0, 0
    
# Get Overall miner data from /miners api - returns 13 values (hashrate_available, hashrate_used, hashrate_free, miners_total, miners_vetting, miners_busy, miners_partial, miners_free, miners_average_difficulty, miners_accepted_shares, miners_accepted_they_rejected, miners_rejected_shares, miners_rejected_they_accepted)
def get_miner_data():
    api_url = os.environ["API_URL"]
    base_url = f"http://{api_url}/miners"     
    base_req = urllib.request.Request(base_url)  
    try:
        response = urllib.request.urlopen(base_req)
        miners_data = json.loads(response.read().decode())
        if miners_data:
            # Get miner Header Data 
            hashrate_available = round(miners_data["TotalHashrateGHS"] / 10**6,4)
            hashrate_used = round(miners_data["UsedHashrateGHS"] / 10**6,4)
            hashrate_free = round(miners_data["AvailableHashrateGHS"] / 10**6,4)
            miners_total = miners_data["TotalMiners"]
            miners_busy = miners_data["BusyMiners"]
            miners_free = miners_data["FreeMiners"]
            miners_vetting = miners_data["VettingMiners"]
            miners_partial = miners_data["PartialBusyMiners"]

            # Miner Section and Summary 
            if miners_data["Miners"]:
                # Initialize miner variables to accumulate total difficulty and stats
                miners_total_difficulty = 0
                miners_accepted_shares = 0
                miners_accepted_they_rejected = 0
                miners_rejected_shares = 0
                miners_rejected_they_accepted = 0
                # Loop through each miner in the dataset
                for miner in miners_data["Miners"]:
                    # Accumulate the current difficulty for each miner
                    miners_total_difficulty += miner["CurrentDifficulty"]
                    # Sum the elements in Stats for each miner
                    stats = miner["Stats"]
                    miners_accepted_shares += stats["we_accepted_shares"]
                    miners_accepted_they_rejected += stats["we_accepted_they_rejected"]
                    miners_rejected_shares += stats["we_rejected_shares"]
                    miners_rejected_they_accepted += stats["we_rejected_they_accepted"]
                # Calculate the average difficulty across all miners
                miners_average_difficulty = miners_total_difficulty / len(miners_data["Miners"]) 
            else: 
                miners_average_difficulty = 0
                miners_accepted_shares = 0
                miners_accepted_they_rejected = 0
                miners_rejected_shares = 0
                miners_rejected_they_accepted = 0               
            
            return hashrate_available, hashrate_used, hashrate_free, miners_total, miners_vetting, miners_busy, miners_partial, miners_free, miners_average_difficulty, miners_accepted_shares, miners_accepted_they_rejected, miners_rejected_shares, miners_rejected_they_accepted
        else:
            print("No miner data found in the response.")
            return 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
    except urllib.error.HTTPError as e:
        print(f"Error occurred while querying miners data: {e}")
        return 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
    
# Get ETH balance for a given address using Etherscan API V2
def get_eth_balance(eth_address):
    chain_id = os.environ["ETH_CHAIN"]
    api_key = os.environ["ETH_API_KEY"]
    # Using unified Etherscan API V2 endpoint with chain ID
    base_url = f"https://api.etherscan.io/v2/api?chainid={chain_id}&module=account&action=balance&address={eth_address}&tag=latest&apikey={api_key}"     
    print(f"ETH Balance V2 URL: {base_url}")
    base_req = urllib.request.Request(base_url)  
    
    max_retries = 3
    retry_delay = 0.2  # API V2 has better rate limits (5 calls/sec free tier)
    
    for attempt in range(max_retries):
        try:
            with urllib.request.urlopen(base_req) as response:
                if response.getcode() == 200:
                    data = json.load(response)
                    
                    # Check for API error
                    if data.get('status') == '0':
                        error_msg = data.get('result', 'Unknown error')
                        print(f"API Error in ETH balance: {error_msg}")
                        if "rate limit" in error_msg.lower():
                            print(f"Rate limit hit, waiting {retry_delay} seconds before retry {attempt + 1}")
                            if attempt < max_retries - 1:
                                time.sleep(retry_delay)
                                retry_delay *= 2
                                continue
                        return "0"
                    
                    # Success case
                    if 'result' in data and data['result'].isdigit():
                        return data['result']
                    else:
                        print("Invalid result format in API response")
                        return "0"
                else:
                    print("Failed to retrieve ETH balance from Etherscan V2")
                    return "0"
        except urllib.error.HTTPError as e:
            print(f"Error occurred while querying ETH balance V2 (attempt {attempt + 1}): {e}")
            if attempt < max_retries - 1:
                time.sleep(retry_delay)
                retry_delay *= 2
            else:
                return "0"
        except Exception as e:
            print(f"Unexpected error while querying ETH balance V2: {e}")
            return "0"
    
    return "0"

# Get LMR token balance for a given address using Etherscan API V2
def get_lmr_token_balance(eth_address, token_address):
    chain_id = os.environ["ETH_CHAIN"]
    api_key = os.environ["ETH_API_KEY"]
    # Using unified Etherscan API V2 endpoint with chain ID
    base_url = f"https://api.etherscan.io/v2/api?chainid={chain_id}&module=account&action=tokenbalance&contractaddress={token_address}&address={eth_address}&tag=latest&apikey={api_key}"     
    print(f"Token Balance V2 URL: {base_url}")
    base_req = urllib.request.Request(base_url)  
    
    max_retries = 3
    retry_delay = 0.2  # API V2 has better rate limits
    
    for attempt in range(max_retries):
        try:
            with urllib.request.urlopen(base_req) as response:
                if response.getcode() == 200:
                    data = json.load(response)
                    
                    # Check for API error
                    if data.get('status') == '0':
                        error_msg = data.get('result', 'Unknown error')
                        print(f"API Error in Token balance: {error_msg}")
                        if "rate limit" in error_msg.lower():
                            print(f"Rate limit hit, waiting {retry_delay} seconds before retry {attempt + 1}")
                            if attempt < max_retries - 1:
                                time.sleep(retry_delay)
                                retry_delay *= 2
                                continue
                        return "0"
                    
                    # Success case
                    if 'result' in data and data['result'].isdigit():
                        return data['result']
                    else:
                        print("Invalid result format in API response")
                        return "0"
                else:
                    print("Failed to retrieve Token balance from Etherscan V2")
                    return "0"
        except urllib.error.HTTPError as e:
            print(f"Error occurred while querying Token balance V2 (attempt {attempt + 1}): {e}")
            if attempt < max_retries - 1:
                time.sleep(retry_delay)
                retry_delay *= 2
            else:
                return "0"
        except Exception as e:
            print(f"Unexpected error while querying Token balance V2: {e}")
            return "0"
    
    return "0"

# Get USDC balance for a given address using Etherscan API V2
def get_usdc_balance(eth_address, token_address):
    chain_id = os.environ["ETH_CHAIN"]
    api_key = os.environ["ETH_API_KEY"]
    # Using unified Etherscan API V2 endpoint with chain ID
    base_url = f"https://api.etherscan.io/v2/api?chainid={chain_id}&module=account&action=tokenbalance&contractaddress={token_address}&address={eth_address}&tag=latest&apikey={api_key}"     
    print(f"USDC Balance V2 URL: {base_url}")
    base_req = urllib.request.Request(base_url)  
    
    max_retries = 3
    retry_delay = 0.2  # API V2 has better rate limits
    
    for attempt in range(max_retries):
        try:
            with urllib.request.urlopen(base_req) as response:
                if response.getcode() == 200:
                    data = json.load(response)
                    
                    # Check for API error
                    if data.get('status') == '0':
                        error_msg = data.get('result', 'Unknown error')
                        print(f"API Error in USDC balance: {error_msg}")
                        if "rate limit" in error_msg.lower():
                            print(f"Rate limit hit, waiting {retry_delay} seconds before retry {attempt + 1}")
                            if attempt < max_retries - 1:
                                time.sleep(retry_delay)
                                retry_delay *= 2
                                continue
                        return "0"
                    
                    # Success case
                    if 'result' in data and data['result'].isdigit():
                        return data['result']
                    else:
                        print("Invalid result format in API response")
                        return "0"
                else:
                    print("Failed to retrieve USDC balance from Etherscan V2")
                    return "0"
        except urllib.error.HTTPError as e:
            print(f"Error occurred while querying USDC balance V2 (attempt {attempt + 1}): {e}")
            if attempt < max_retries - 1:
                time.sleep(retry_delay)
                retry_delay *= 2
            else:
                return "0"
        except Exception as e:
            print(f"Unexpected error while querying USDC balance V2: {e}")
            return "0"
    
    return "0"

########### SEND METRICS TO CLOUDWATCH ###########
def lambda_handler(event, context):
    healthcheck_status, healthcheck_uptime, healthcheck_version = get_healthcheck_data()
    config_version, config_commit, config_wallet, config_lumerin, config_usdc, config_clonefactory = get_config_data()
    seller_hashrate_offered, seller_hashrate_purchased, seller_contracts_offered, seller_contracts_active, validator_hashrate_purchased, validator_hashrate_actual, validator_contracts_active, buyers_unique   = get_contractsv2_data()
    hashrate_available, hashrate_used, hashrate_free, miners_total, miners_vetting, miners_busy, miners_partial, miners_free, miners_average_difficulty, miners_accepted_shares, miners_accepted_they_rejected, miners_rejected_shares, miners_rejected_they_accepted = get_miner_data()
    
    # Get financial data using API V2 (better rate limits, more reliable)
    seller_eth_balance = get_eth_balance(config_wallet)
    # Smaller delay needed with V2 API (5 calls/sec vs 2 calls/sec)
    time.sleep(0.3)
    seller_token_balance = get_lmr_token_balance(config_wallet, config_lumerin)
    time.sleep(0.3)
    seller_usdc_balance = get_usdc_balance(config_wallet, config_usdc)

    # Send metrics to CloudWatch - map returned data from API calls to outbound metrics
    cloudwatch = boto3.client("cloudwatch")
    cloudwatch.put_metric_data(
        Namespace= cw_namespace,
        MetricData=[
            {"MetricName": cw_metric1, "Value": validator_contracts_active, "Unit": "Count" },
            {"MetricName": cw_metric2, "Value": validator_hashrate_purchased, "Unit": "Count" },
            {"MetricName": cw_metric3, "Value": validator_hashrate_actual, "Unit": "Count" },
            {"MetricName": cw_metric4, "Value": buyers_unique, "Unit": "Count" },
            {"MetricName": cw_metric5, "Value": miners_total, "Unit": "Count" },
            {"MetricName": cw_metric6, "Value": round((int(seller_eth_balance or "0")/10**18),8), "Unit": "None" },
            {"MetricName": cw_metric7, "Value": round((int(seller_token_balance or "0")/10**8),4), "Unit": "None" }, 
            {"MetricName": cw_metric8, "Value": round(int(miners_average_difficulty),4), "Unit": "Count" },
            {"MetricName": cw_metric9, "Value": int(miners_accepted_shares), "Unit": "Count" },
            {"MetricName": cw_metric10, "Value": int(miners_accepted_they_rejected), "Unit": "Count" },
            {"MetricName": cw_metric11, "Value": int(miners_rejected_shares), "Unit": "Count" },
            {"MetricName": cw_metric12, "Value": int(miners_rejected_they_accepted), "Unit": "Count" }, 
            {"MetricName": cw_metric13, "Value": round((int(seller_usdc_balance or "0")/10**6),4), "Unit": "None" }
        ]
    )
    
    # Log the metrics in Lambda output or logs
    print(f"{current_time}")
    print(f"Metrics for {api_url}:")
    print(f"Healthcheck:")
    print(f"  -Status: {healthcheck_status}")
    print(f"  -Uptime: {healthcheck_uptime}")   
    print(f"  -Version: {healthcheck_version}")
    print(f"\nConfig:")
    print(f"  -Version: {config_version}")
    print(f"  -Commit: {config_commit}")
    print(f"  -Wallet: {config_wallet}")
    print(f"  -LMR Token: {config_lumerin}")
    print(f"  -USDC Token: {config_usdc}")
    print(f"  -CloneFactory: {config_clonefactory}")
    print(f"\nContracts: {validator_contracts_active}")
    print(f"\nHashrate (PH/s):")
    print(f"  -Purchased: {validator_hashrate_purchased}")
    print(f"  -Actual: {validator_hashrate_actual}")
    print(f"\nBuyers: {buyers_unique}")
    print(f"\nMiners: {miners_total}")
    print(f"\nFinancial:")
    print(f"  -ETH Balance: {round((int(seller_eth_balance or '0')/10**18),8)}")
    print(f"  -LMR Balance: {round((int(seller_token_balance or '0')/10**8),4)}")
    print(f"  -USDC Balance: {round((int(seller_usdc_balance or '0')/10**6),4)}")    
    print(f"\nMiner Stats:")
    print(f"  -Average Difficulty: {round(int(miners_average_difficulty),4)}")
    print(f"  -Accepted Shares: {int(miners_accepted_shares)}")
    print(f"  -Accepted They Rejected: {int(miners_accepted_they_rejected)}")
    print(f"  -Rejected Shares: {int(miners_rejected_shares)}")
    print(f"  -Rejected They Accepted: {int(miners_rejected_they_accepted)}")
    
    # Return any desired response from the Lambda function
    return {
        "statusCode": 200,
        "body": "Metrics sent to CloudWatch"
    }
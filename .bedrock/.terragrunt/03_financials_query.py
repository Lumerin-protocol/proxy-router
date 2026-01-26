import urllib.request
import json
import boto3
import os


########### CRYPTOCURRENCY PRICE AND CALCULATION FUNCTIONS ###########
def get_crypto_price(url):
    try:
        with urllib.request.urlopen(url) as response:
            data = json.loads(response.read().decode())
            return float(data['data']['amount'])
    except Exception as e:
        print(f"Error fetching crypto price from {url}: {e}")
        return 0.0

def get_lumerin_price():
    url = "https://api.coingecko.com/api/v3/simple/price?ids=lumerin&vs_currencies=usd,btc"
    try:
        with urllib.request.urlopen(url) as response:
            data = json.loads(response.read().decode())
            lmr_usd = float(data['lumerin']['usd'])
            lmr_btc = float(data['lumerin']['btc'])
            return lmr_usd, lmr_btc
    except Exception as e:
        print(f"Error fetching Lumerin price: {e}")
        return 0.0, 0.0

def get_btc_difficulty():
    try:
        with urllib.request.urlopen("https://blockchain.info/q/getdifficulty") as response:
            difficulty_exp = response.read().decode()
            difficulty = float(difficulty_exp)
            return difficulty / (10 ** 12)
    except Exception as e:
        print(f"Error fetching Bitcoin difficulty: {e}")
        return 0.0

def calculate_earnings_and_breakeven(your_hash_rate_th_s, mining_hours, block_reward_btc, btc_price, btc_difficulty_t):
    # Calculate the number of blocks mined
    average_block_time_minutes = 10
    blocks_mined = mining_hours * (60 / average_block_time_minutes)

    # Calculate total network hash rate in TH/s
    total_network_hash_rate_th_s = (btc_difficulty_t * (2 ** 32)) / (average_block_time_minutes * 60)

    # Calculate earnings in BTC
    earnings_btc = (your_hash_rate_th_s / total_network_hash_rate_th_s) * block_reward_btc * blocks_mined

    # Convert earnings to USD
    earnings_usd = earnings_btc * btc_price

    # Calculate breakeven in BTC, using Lumerin's BTC price
    lmr_btc = get_lumerin_price()[1]
    breakeven_btc = earnings_btc / lmr_btc if lmr_btc else 0

    return earnings_btc, earnings_usd, breakeven_btc

########### SEND METRICS TO CLOUDWATCH ###########
def lambda_handler(event, context):
    # Define input variables for calculations
    your_hash_rate_th_s = 100
    mining_hours = 24
    block_reward_btc = 6.25

    # Fetch prices and network data
    btc_price = get_crypto_price("https://api.coinbase.com/v2/prices/BTC-USD/spot")
    eth_price = get_crypto_price("https://api.coinbase.com/v2/prices/ETH-USD/spot")
    lmr_usd, lmr_btc = get_lumerin_price()
    btc_difficulty_t = get_btc_difficulty()

    # Calculate earnings and breakeven
    earnings_btc, earnings_usd, breakeven_btc = calculate_earnings_and_breakeven(
        your_hash_rate_th_s, mining_hours, block_reward_btc, btc_price, btc_difficulty_t
    )

    # Send metrics to CloudWatch
    try:
        cloudwatch = boto3.client("cloudwatch")
        cloudwatch.put_metric_data(
            Namespace=os.environ["CW_NAMESPACE"],
            MetricData=[
                {"MetricName": os.environ["CW_METRIC1"], "Value": btc_price, "Unit": "Count"},
                {"MetricName": os.environ["CW_METRIC2"], "Value": eth_price, "Unit": "Count"},
                {"MetricName": os.environ["CW_METRIC3"], "Value": lmr_usd, "Unit": "Count"},
                {"MetricName": os.environ["CW_METRIC4"], "Value": btc_difficulty_t, "Unit": "None"},
                {"MetricName": os.environ["CW_METRIC5"], "Value": earnings_btc, "Unit": "None"},
                {"MetricName": os.environ["CW_METRIC6"], "Value": earnings_usd, "Unit": "None"},
                {"MetricName": os.environ["CW_METRIC7"], "Value": breakeven_btc, "Unit": "Count"}
            ]
        )
        print("Metrics sent to CloudWatch successfully.")
    except Exception as e:
        print(f"Error sending metrics to CloudWatch: {e}")

    # Log outputs for debugging
    print(f"Bitcoin Price: {btc_price}")
    print(f"Ethereum Price: {eth_price}")
    print(f"Lumerin Price (USD): {lmr_usd}")
    print(f"Bitcoin Difficulty: {btc_difficulty_t}")
    print(f"Earnings BTC: {earnings_btc}")
    print(f"Earnings USD: {earnings_usd}")
    print(f"Breakeven BTC: {breakeven_btc}")

    return {
        "statusCode": 200,
        "body": "Metrics sent to CloudWatch"
    }

# For local testing, you can simulate a Lambda invocation:
if __name__ == "__main__":
    result = lambda_handler({}, {})
    print(result)
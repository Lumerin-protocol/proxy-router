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
cw_namespace = os.environ["CW_NAMESPACE"]
cw_metric1 = os.environ["CW_METRIC1"]
cw_metric2 = os.environ["CW_METRIC2"]

# Get Healthcheck data from /healthcheck api - returns 3 values (status, uptime, version)    
def get_healthcheck_data():
    api_url = os.environ["API_URL"]
    base_url = f"https://{api_url}/healthcheck"     
    base_req = urllib.request.Request(base_url)  
    try:
        response = urllib.request.urlopen(base_req)
        healthcheck_data = json.loads(response.read().decode())
        if healthcheck_data:  
            healthcheck_status = healthcheck_data["status"]
            healthcheck_version = healthcheck_data["version"]
            healthcheck_uptime = healthcheck_data["uptimeSeconds"]
            healthcheck_cloneFactoryAddress = healthcheck_data["cloneFactoryAddress"]
            healthcheck_lastsyncedblock = healthcheck_data["lastSyncedContractBlock"]
            healthcheck_lastsyncedtime = healthcheck_data["lastSyncedTime"]
            healthcheck_lastSyncedTimeISO = healthcheck_data["lastSyncedTimeISO"]
            return healthcheck_status, healthcheck_uptime, healthcheck_version, healthcheck_lastsyncedblock, healthcheck_lastsyncedtime, healthcheck_lastSyncedTimeISO, healthcheck_cloneFactoryAddress
        else:
            print("No contract data found in the response.")
            return None, None, None, None, None, None, None     

    except urllib.error.HTTPError as e:
            print(f"Error occurred while querying contract data: {e}")
            return None, None, None, None, None, None, None 

########### SEND METRICS TO CLOUDWATCH ###########
def lambda_handler(event, context):
    healthcheck_status, healthcheck_uptime, healthcheck_version, healthcheck_lastsyncedblock, healthcheck_lastsyncedtime, healthcheck_lastSyncedTimeISO, healthcheck_cloneFactoryAddress = get_healthcheck_data()

    # Send metrics to CloudWatch - map returned data from API calls to outbound metrics
    cloudwatch = boto3.client("cloudwatch")
    cloudwatch.put_metric_data(
        Namespace= cw_namespace,
        MetricData=[
            {"MetricName": cw_metric1, "Value": healthcheck_uptime, "Unit": "Count" },
            {"MetricName": cw_metric2, "Value": healthcheck_lastsyncedblock, "Unit": "Count" }
        ]
    )
    
    # Log the metrics in Lambda output or logs
    print(f"{current_time}")
    print(f"Metrics for {api_url}:")
    print(f"Healthcheck:")
    print(f"  -Status: {healthcheck_status}")
    print(f"  -Uptime: {healthcheck_uptime}")   
    print(f"  -Version: {healthcheck_version}")
    print(f"  -Clone Factory Address: {healthcheck_cloneFactoryAddress}")
    print(f"  -Last Synced Block: {healthcheck_lastsyncedblock}")
    print(f"  -Last Synced Time: {healthcheck_lastsyncedtime}")
    print(f"  -Last Synced Time: {healthcheck_lastSyncedTimeISO}")
    
    # Return any desired response from the Lambda function
    return {
        "statusCode": 200,
        "body": "Metrics sent to CloudWatch"
    }
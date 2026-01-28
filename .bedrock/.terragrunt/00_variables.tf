################################################################################
# VARIABLES 
################################################################################
# All variables set in ./terraform.tfvars must be initialized here
# Any of these variables can be used in any of this environment's .tf files
variable "account_shortname" {
  description = "Code describing customer  and lifecycle. E.g., mst, sbx, dev, stg, prd"
}
variable "account_lifecycle" {
  description = "environment lifecycle, can be 'prod', 'nonprod', 'sandbox'...dev and stg are considered nonprod"
  type        = string
}
variable "account_number" {
}
variable "default_region" {
}
variable "region_shortname" {
  description = "Region 4 character shortname"
  default     = "use1"
}
variable "vpc_index" {}
variable "devops_keypair" {}
variable "titanio_net_edge_vpn" {}
variable "default_tags" {
  description = "Default tag values common across all resources in this account. Values can be overridden when configuring a resource or module."
  type        = map(string)
}
variable "foundation_tags" {
  description = "Default Tags for Bedrock Foundation resources"
  type        = map(string)
}
variable "provider_profile" {
  description = "Provider config added for use in aws_config.tf"
}

################################################################################
variable "futures_validator_url_override" {
  description = "Contains information about Futures Validator URL Override"
  type        = string
  default     = ""
}
variable "futures_address" {
  description = "Contains information about Futures Address"
  type        = string
  default     = ""
}
variable "clone_factory_address" {
  description = "Contains information about Clone Factory Address"
  type        = string
  default     = ""
}
variable "oracle_address" {
  description = "Contains information about Oracle Address"
  type        = string
  default     = ""
}
variable "validator_registry_address" {
  description = "Contains information about Validator Registry Address"
  type        = string
  default     = ""
}

variable "wallets_to_watch" {
  description = "List of wallets to monitor with name and address. Each object should have walletName, walletId, and optional alarm thresholds (eth_alarm_threshold, usdc_alarm_threshold, lmr_alarm_threshold)"
  type = list(object({
    walletName           = string
    walletId             = string
    eth_alarm_threshold  = optional(number)
    usdc_alarm_threshold = optional(number)
    lmr_alarm_threshold  = optional(number)
  }))
  default = []
}

variable "proxy_ecs" {
  description = "Contains information about Proxy ECS Cluster"
  type        = map(any)
}

variable "proxy_router" {
  description = "Contains information about Proxy Node and Associated ALB"
  type        = map(any)
}

variable "proxy_routertwo" {
  description = "Contains information about Proxy Two Node and Associated ALB"
  type        = map(any)
}

variable "proxy_buyer" {
  description = "Contains information about Seller Node"
  type        = map(any)
}
variable "proxy_validator" {
  description = "Contains information about Big-V Validator Node"
  type        = map(any)
}

variable "special_nodes" {
  description = "Contains information about special node details"
  type        = map(any)
}

# Variables for Monitoring 
variable "monitoring_frequency" {
  description = "Frequency of monitoring"
  type        = string
  default     = "rate(5 minutes)"
}

variable "financials_query_create" {
  description = "Create Financials Query Lambda"
  type        = bool
  default     = false
}

variable "proxy_router_query_create" {
  description = "Create ProxyRouter Query Lambda"
  type        = bool
  default     = false
}

variable "validator_query_create" {
  description = "Create Validator Query Lambda"
  type        = bool
  default     = false
}

variable "indexer_query_create" {
  description = "Create Indexer Query Lambda"
  type        = bool
  default     = false
}
variable "oracle_query_create" {
  description = "Create Oracle Query Lambda"
  type        = bool
  default     = false
}

variable "monitoring_dashboard_create" {
  description = "Create Standard Dashboard"
  type        = bool
  default     = false
}

variable "eth_chain" {
  description = "Ethereum Chain ID (42161 for Arbitrum One, 421614 for Arbitrum Sepolia)"
  type        = string
  default     = ""
}

variable "financials_query" {
  description = "Contains information about Financial API Query Lambda"
  type        = map(any)
  default = {
    name         = ["bedrock-financial-query"]
    cw_namespace = ["proxy-financials"]
    cw_metric1   = ["btc_price"]      #Get BTC Price
    cw_metric2   = ["eth_price"]      #Get ETH Price
    cw_metric3   = ["lmr_price"]      #Get LMR Price  
    cw_metric4   = ["btc_difficulty"] #Get BTC Difficulty
    cw_metric5   = ["earnings_btc"]   # Calculate potential earnings in BTC
    cw_metric6   = ["earnings_usd"]   # Calculate potential earnings in USD
    cw_metric7   = ["breakeven_btc"]  # Calculate Breakeven in BTC for Pricing 
  }
}

variable "proxy_router_query" {
  description = "Contains information about Seller Node API Query Lambda"
  type        = map(any)
  default = {
    name         = ["bedrock-proxyrouter-query"]
    cw_namespace = ["proxy-router"]
    cw_metric1   = ["contracts_offered"]  #how many contracts are active and available 
    cw_metric2   = ["contracts_active"]   #contracts that are active 
    cw_metric3   = ["hashrate_offered"]   #hr for all contracts active and availble (should never exceed hashrate availale)
    cw_metric4   = ["hashrate_available"] #hr for inbound hashrate to node 
    cw_metric5   = ["hashrate_used"]      #hr actually used by the node 
    cw_metric6   = ["hashrate_free"]      #hr free on the node to def pool...should be > 0 
    cw_metric7   = ["miners_total"]
    cw_metric8   = ["miners_vetting"]
    cw_metric9   = ["miners_busy"]
    cw_metric10  = ["miners_partial"]
    cw_metric11  = ["miners_free"]
    cw_metric12  = ["buyers_unique"]
    cw_metric13  = ["hashrate_purchased"] #hr purchased by buyers (should be close to hashrate_used)
    cw_metric14  = ["wallet_eth"]         #wallet balance in eth
    cw_metric15  = ["wallet_lmr"]         #wallet lmr token balance
    cw_metric16  = ["miners_average_difficulty"]
    cw_metric17  = ["miners_accepted_shares"]
    cw_metric18  = ["miners_accepted_they_rejected"]
    cw_metric19  = ["miners_rejected_shares"]
    cw_metric20  = ["miners_rejected_they_accepted"]
    cw_metric21  = ["wallet_usdc"]       #wallet balance in usdc
    cw_metric22  = ["wallet_eth_oracle"] #wallet balance in eth for oracle
  }
}

variable "validator_query" {
  description = "Contains information about Validator Node API Query Lambda"
  type        = map(any)
  default = {
    name         = ["bedrock-validator-query"]
    cw_namespace = ["proxy-validator"]
    cw_metric1   = ["contracts_active"]
    cw_metric2   = ["hashrate_purchased"]
    cw_metric3   = ["hashrate_actual"]
    cw_metric4   = ["buyers_unique"]
    cw_metric5   = ["miners_total"]
    cw_metric6   = ["wallet_eth"] #wallet balance in eth
    cw_metric7   = ["wallet_lmr"] #wallet lmr token balance
    cw_metric8   = ["miners_average_difficulty"]
    cw_metric9   = ["miners_accepted_shares"]
    cw_metric10  = ["miners_accepted_they_rejected"]
    cw_metric11  = ["miners_rejected_shares"]
    cw_metric12  = ["miners_rejected_they_accepted"]
    cw_metric13  = ["wallet_usdc"] #wallet balance in usdc
  }
}

variable "indexer_query" {
  description = "Contains information about Indexer Node API Query Lambda"
  type        = map(any)
  default = {
    name         = ["bedrock-indexer-query"]
    cw_namespace = ["proxy-indexer"]
    cw_metric1   = ["uptimeSeconds"]
    cw_metric2   = ["lastSyncedContractBlock"]
  }
}

# Wallet Monitor Variables
variable "wallet_monitor_query_create" {
  description = "Create Wallet Monitor Query Lambda"
  type        = bool
  default     = false
}

variable "wallet_monitor_frequency" {
  description = "Frequency of wallet monitoring (e.g., 'rate(15 minutes)' or 'cron(0/15 * * * ? *)')"
  type        = string
  default     = "rate(15 minutes)"
}

variable "wallet_monitor_query" {
  description = "Configuration for Wallet Monitor Lambda"
  type        = map(any)
  default = {
    name                     = "bedrock-wallet-monitor"
    cw_namespace             = "wallet-monitor"
    lmr_token_address        = "0xaf5db6e1cc585ca312e8c8f7c499033590cf5c98" # Arbitrum One LMR
    usdc_token_address       = "0xaf88d065e77c8cC2239327C5EDb3A432268e5831" # Arbitrum One USDC (native)
    alarm_evaluation_periods = 2
    alarm_period             = 900 # 15 minutes in seconds
  }
}


#### SENSITIVE VARIABLES 
# Must have file secret.auto.tfvars in same folder locally and ensure same is in the .gitignore file

variable "bedrock_glpat" {
  description = "Contains Gitlab ProjectAccess Token for Bedrock"
  type        = string
  default     = ""
  sensitive   = true
}
variable "titanadmin_pubkey" {
  description = "Contains Public Key for titanadmin"
  type        = string
  default     = ""
  sensitive   = true
}

variable "node_admin" {
  description = "Node admin user"
  type        = string
  default     = "titanadmin"
  sensitive   = true
}
variable "node_password" {
  description = "Password for node admin user"
  type        = string
  sensitive   = true
}
variable "x_custom_header_bypass" {
  description = "Custom header bypass"
  type        = string
  sensitive   = true
  default     = ""
}
variable "eth_api_key" {
  description = "Etherscan API Key for V2 unified API"
  type        = string
  sensitive   = true
  default     = ""
}
variable "foreman_api_key" {
  description = "Foreman API Key"
  type        = string
  sensitive   = true
}
variable "ghissues_query_authtoken" {
  description = "GitHub Lumerin.io AuthToken"
  type        = string
  sensitive   = true
}

variable "proxy_wallet_private_key" {
  description = "Proxy-Router Private key "
  type        = string
  sensitive   = true
  default     = ""
}
variable "validator_wallet_private_key" {
  description = "Validator Private key "
  type        = string
  sensitive   = true
  default     = ""
}
variable "proxy_eth_node_address" {
  description = "Proxy-Router Eth Node Address "
  type        = string
  sensitive   = true
  default     = ""
}
variable "validator_eth_node_address" {
  description = "Validator Eth Node Address "
  type        = string
  sensitive   = true
  default     = ""
}

variable "graph_api_key" {
  description = "The Graph API Key for accessing published subgraphs"
  type        = string
  sensitive   = true
  default     = ""
}

variable "futures_subgraph_id" {
  description = "The Graph Subgraph ID for Futures (from published subgraph)"
  type        = string
  sensitive   = true
  default     = ""
}

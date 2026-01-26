########################################
# Account metadata
########################################
provider_profile  = "titanio-dev"  # Local account profile ... should match account_shortname..kept separate for future ci/cd
account_shortname = "titanio-dev"  # shortname account code 7 digit + 3 digit eg: titanio-mst, titanio-inf, or rhodium-prd
account_number    = "434960487817" # 12 digit account number 
account_lifecycle = "dev"          # [sbx, dev, stg, prd] -used for NACL and other reference
default_region    = "us-east-1"
region_shortname  = "use1"

########################################
# Environment Specific Variables
#######################################
vpc_index            = 1
devops_keypair       = "bedrock-titanio-dev-use1"
titanio_net_edge_vpn = "172.18.16.0/20"

# To call mapped vars in code: `var.proxy_ecs["create"]`
proxy_ecs = {
  create          = "true"
  protect         = "true"
  task_worker_qty = "1"
  name            = "proxy-router"
}

special_nodes = {
  qa_bugs_create  = "false"
  qa_daffy_create = "false"
  qa_lola_create  = "false"
}

# contract_defaults variables
clone_factory_address      = "0x998135c509b64083cd27ed976c1bcda35ab7a40b" # owned by 0x1441Bc52156Cf18c12cde6A92aE6BDE8B7f775D4
oracle_address             = "0x6f736186d2c93913721e2570c283dff2a08575e9" # owned by 0x0eB467381abbC5B71f275DF0c8a4E0ED8561F46f updated by 0x0eB467381abbC5B71f275DF0c8a4E0ED8561F46f
futures_address            = "0xec76867e96d942282fc7aafe3f778de34d41a311" # owned by 0x1441Bc52156Cf18c12cde6A92aE6BDE8B7f775D4
validator_registry_address = "0x959922be3caee4b8cd9a407cc3ac1c251c2007b1"
# validator_wallet             = "0xc3acdae18291bfeb0671d1caab1d13fe04164f75"
# validator_url                = "validator.dev.lumerin.io:7301"

proxy_router = {
  create                 = "true"
  monitor_metric_filters = "true"
  protect                = "false"
  svca_cnt_port          = "3333"
  svca_hst_port          = "3333"
  svca_alb_port          = "7301"
  svca_protocol          = "TCP"
  svcb_cnt_port          = "8080"
  svcb_hst_port          = "8080"
  svcb_alb_port          = "8080"
  svcb_protocol          = "HTTP"
  ecr_repo               = "proxy-router"
  svc_name               = "proxy-router"
  cnt_name               = "proxy-router"
  image_tag              = "auto"
  dns_alb                = "proxy"
  dns_alb_api            = "proxyapi"
  dns_ga                 = "proxyga"
  task_cpu               = "256"
  task_ram               = "512"
  task_worker_qty        = "1"
  pool_address           = "//f2poollmn.ecs-dev2:@btc.f2pool.com:1314"
  web_address            = "0.0.0.0:8080"
  web_public_url         = "http://proxyapi.dev.lumerin.io:8080"
}

proxy_validator = {
  create                 = "true"
  monitor_metric_filters = "true"
  protect                = "false"
  svca_cnt_port          = "3333"
  svca_hst_port          = "3333"
  svca_alb_port          = "7301"
  svca_protocol          = "TCP"
  svcb_cnt_port          = "8080"
  svcb_hst_port          = "8080"
  svcb_alb_port          = "8080"
  svcb_protocol          = "HTTP"
  ecr_repo               = "proxy-router"
  image_tag              = "auto"
  cnt_name               = "proxy-router"
  svc_name               = "proxy-validator"
  dns_alb                = "validator"
  dns_alb_api            = "validatorapi"
  task_cpu               = "256"
  task_ram               = "512"
  task_worker_qty        = "1"
  pool_address           = "//f2poollmn.ecs-dev2-val:@btc.f2pool.com:1314"
  web_address            = "0.0.0.0:8080"
  web_public_url         = "http://validatorapi.dev.lumerin.io:8080"
}

# Create Cloudwatch Metrics & Dashboards
monitoring_frequency        = "rate(1 minute)"
financials_query_create     = "true"
proxy_router_query_create   = "true"
validator_query_create      = "true"
indexer_query_create        = "true"
monitoring_dashboard_create = "true"
eth_chain                   = "421614"

# Wallet Monitor Configuration
wallet_monitor_query_create = "true"
wallet_monitor_frequency    = "rate(5 minutes)" # More frequent for dev/testing
wallet_monitor_query = {
  name                     = "bedrock-wallet-monitor"
  cw_namespace             = "wallet-monitor"
  lmr_token_address        = "0xC27DafaD85F199FD50dD3FD720654875D6815871" # Arbitrum Sepolia - update when deployed
  usdc_token_address       = "0x217C835e751DD12E7f1824b7D8ee0fB159B6EE2B" # Arbitrum Sepolia USDC (Circle test)
  alarm_evaluation_periods = 2
  alarm_period             = 900 # 5 minutes in seconds for dev
}

# Wallets to monitor for ETH, USDC, and LMR balances (dev environment)
wallets_to_watch = [
  {
    walletName          = "Seller"
    walletId            = "0x1441Bc52156Cf18c12cde6A92aE6BDE8B7f775D4"
    eth_alarm_threshold = 0.001 # Lower threshold for testnet
  },
  {
    walletName          = "Validator"
    walletId            = "0xc3acdae18291bfeb0671d1caab1d13fe04164f75"
    eth_alarm_threshold = 0.001
  },
  {
    walletName          = "MarketMaker"
    walletId            = "0x4040eEEfc184c1382d708E6fA53685Bc22992B44"
    eth_alarm_threshold = 0.001 # Market maker needs more ETH for gas
  },
  {
    walletName          = "OracleUpdater"
    walletId            = "0x0eB467381abbC5B71f275DF0c8a4E0ED8561F46f"
    eth_alarm_threshold = 0.001
  }
]

proxy_routertwo = {
  create                 = "false"
  monitor_metric_filters = "true"
  protect                = "false"
  svca_cnt_port          = "3333"
  svca_hst_port          = "3333"
  svca_alb_port          = "7301"
  svca_protocol          = "TCP"
  svcb_cnt_port          = "8080"
  svcb_hst_port          = "8080"
  svcb_alb_port          = "8080"
  svcb_protocol          = "TCP"
  ecr_repo               = "proxy-router"
  svc_name               = "proxy-routertwo"
  cnt_name               = "proxy-router"
  image_tag              = "auto"
  dns_alb                = "proxytwo"
  dns_alb_api            = "proxytwoapi"
  task_cpu               = "4096"
  task_ram               = "8192"
  task_worker_qty        = "1"
}

proxy_buyer = {
  create                 = "false"
  monitor_metric_filters = "false"
  protect                = "false"
  svca_cnt_port          = "3333"
  svca_hst_port          = "3333"
  svca_alb_port          = "7301"
  svca_protocol          = "TCP"
  svcb_cnt_port          = "8080"
  svcb_hst_port          = "8080"
  svcb_alb_port          = "8080"
  svcb_protocol          = "TCP"
  ecr_repo               = "proxy-router"
  image_tag              = "auto"
  cnt_name               = "proxy-router"
  svc_name               = "proxy-buyer"
  dns_alb                = "buyer"
  dns_alb_api            = "buyerapi"
  task_cpu               = "2048"
  task_ram               = "4096"
  task_worker_qty        = "1"
}

# Default tag values common across all resources in this account.
# Values can be overridden when configuring a resource or module.
default_tags = {
  ServiceOffering = "Cloud Foundation"
  Department      = "DevOps"
  Environment     = "dev"
  Owner           = "aws-titanio-dev@titan.io" #AWS Account Email Address 092029861612 | aws-sandbox@titan.io | OrganizationAccountAccessRole 
  Scope           = "Global"
  CostCenter      = null
  Compliance      = null
  Classification  = null
  Repository      = "https://gitlab.com/TitanInd/bedrock/foundation-afs/proxy-router-foundation.git//bedrock/02-dev"
  ManagedBy       = "Terraform"
}

# Default Tags for Cloud Foundation resources
foundation_tags = {
  Name          = "Lumerin Proxy Router - DEV"
  Capability    = null
  Application   = "Lumerin Proxy Router - DEV"
  LifecycleDate = null
}
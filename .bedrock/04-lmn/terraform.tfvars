########################################
# Account metadata
########################################
provider_profile  = "titanio-lmn"  # Local account profile ... should match account_shortname..kept separate for future ci/cd
account_shortname = "titanio-lmn"  # shortname account code 7 digit + 3 digit eg: titanio-mst, titanio-inf, or rhodium-prd
account_number    = "330280307271" # 12 digit account number 
account_lifecycle = "prd"          # [sbx, dev, stg, prd] -used for NACL and other reference
default_region    = "us-east-1"
region_shortname  = "use1"

########################################
# Environment Specific Variables
#######################################
vpc_index            = 1
devops_keypair       = "bedrock-titanio-lmn-use1"
titanio_net_edge_vpn = "172.18.16.0/20"

# To call mapped vars in code: `var.proxy_ecs["create"]`
proxy_ecs = {
  create          = "true"
  protect         = "true"
  task_worker_qty = "1"
  name            = "proxy-router"
}

special_nodes = {
  pr_coyote_create = "false"
}

# contract_defaults variables
clone_factory_address          = "0x6b690383c0391B0Cf7d20B9eB7A783030b1f3f96"
oracle_address                 = "0x6599ef8e2B4A548a86eb82e2dfbc6CEADFCEaCBd"
futures_address                = "0x8464dc5ab80e76e497fad318fe6d444408e5ccda"
validator_registry_address     = "0xcd0281d88c15ec5c84233d7bc15e57c8b75437a0"
# validator_wallet             = "0x344C98E25F981976215669E048ECcb21be16aC8e"
# validator_url                = "validator.lumerin.io:7301"

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
  task_cpu               = "2048"
  task_ram               = "4096"
  task_worker_qty        = "1"
  pool_address           = "//f2poollmn.ecs-lmn:@btc.f2pool.com:1314"
  web_address            = "0.0.0.0:8080"
  web_public_url         = "http://proxyapi.lumerin.io:8080"
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
  task_cpu               = "2048"
  task_ram               = "4096"
  task_worker_qty        = "1"
  pool_address           = "//f2poollmn.ecs-lmn-val:@btc.f2pool.com:1314"
  web_address            = "0.0.0.0:8080"
  web_public_url         = "http://validatorapi.lumerin.io:8080"
}

# Create Cloudwatch Metrics & Dashboards
# The Arbitrum contract-query lambdas are off. The wallet monitor reads Base.
monitoring_frequency        = "rate(5 minutes)"
financials_query_create     = "false"
proxy_router_query_create   = "false"
validator_query_create      = "false"
indexer_query_create        = "false"
monitoring_dashboard_create = "true"
eth_chain                   = "8453" # Base. Wallet monitor uses wallet_monitor_query.eth_chain.

# Wallet Monitor Configuration
wallet_monitor_query_create = "true"
wallet_monitor_frequency    = "rate(15 minutes)"
wallet_monitor_query = {
  name                     = "bedrock-wallet-monitor"
  cw_namespace             = "wallet-monitor"
  eth_chain                = "8453" # Base
  lmr_token_address        = "0x0000000000000000000000000000000000000000"
  usdc_token_address       = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913" # Base USDC
  alarm_evaluation_periods = 2
  alarm_period             = 900 # 15 minutes in seconds
}
# Gas wallets for the exchange. The keeper signs with the market maker.
wallets_to_watch = [
  {
    walletName          = "MarketMaker"
    walletId            = "0xc1e187E4a677Da017ecfAc011C9d381c3E7baeE4"
    eth_alarm_threshold = 0.025
  },
  {
    walletName          = "OracleUpdater"
    walletId            = "0xf19cc0cD098554f6Dd1978eDB9C1408816E1DFB1"
    eth_alarm_threshold = 0.025
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
  task_cpu               = "4096"
  task_ram               = "8192"
  task_worker_qty        = "1"
}

# Default tag values common across all resources in this account.
# Values can be overridden when configuring a resource or module.
default_tags = {
  ServiceOffering = "Cloud Foundation"
  Department      = "DevOps"
  Environment     = "lmn"
  Owner           = "aws-titanio-lmn@titan.io" #AWS Account Email Address 092029861612 | aws-sandbox@titan.io | OrganizationAccountAccessRole 
  Scope           = "Global"
  CostCenter      = null
  Compliance      = null
  Classification  = null
  Repository      = "https://gitlab.com/TitanInd/bedrock/foundation-afs/proxy-router-foundation.git//bedrock/04-lmn"
  ManagedBy       = "Terraform"
}

# Default Tags for Cloud Foundation resources
foundation_tags = {
  Name          = "Lumerin Proxy Router - LMN"
  Capability    = null
  Application   = "Lumerin Proxy Router - LMN"
  LifecycleDate = null
}
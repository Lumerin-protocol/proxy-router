########################################
# Account metadata
########################################
provider_profile  = "titanio-stg"  # Local account profile ... should match account_shortname..kept separate for future ci/cd
account_shortname = "titanio-stg"  # shortname account code 7 digit + 3 digit eg: titanio-mst, titanio-inf, or rhodium-prd
account_number    = "464450398935" # 12 digit account number 
account_lifecycle = "stg"          # [sbx, dev, stg, prd] -used for NACL and other reference
default_region    = "us-east-1"
region_shortname  = "use1"

########################################
# Environment Specific Variables
#######################################
vpc_index            = 1
devops_keypair       = "bedrock-titanio-stg-use1"
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
clone_factory_address          = "0xb5838586b43b50f9a739d1256a067859fe5b3234" # owned by SAFE:arb1:0x63E09ead6CcF8850287370B4b248B02D4D43e1Ba
oracle_address                 = "0x2c1db79d2f3df568275c940dac81ad251871faf4" # owned by SAFE:arb1:0x63E09ead6CcF8850287370B4b248B02D4D43e1Ba updated by: 0x67C1A7737e0C47E53FD4a828c9c7d81401ce912b (set in proxy-ui-foundation)
futures_address                = "0xe11594879beb6c28c67bc251aa5e26ce126b82ba" # owned by SAFE:arb1:0x63E09ead6CcF8850287370B4b248B02D4D43e1Ba
validator_registry_address     = "0xa6354b657d8a42f2006c4ad0df670a831a610ca8"
# validator_wallet             = "0x06bA6986F7B71B9115670aedFE0de759b708d599"
# validator_url                = "validator.stg.lumerin.io:7301"

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
  pool_address           = "//f2poollmn.ecs-stg:@btc.f2pool.com:1314"
  web_address            = "0.0.0.0:8080"
  web_public_url         = "http://proxyapi.stg.lumerin.io:8080"
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
  pool_address           = "//f2poollmn.ecs-stg-val:@btc.f2pool.com:1314"
  web_address            = "0.0.0.0:8080"
  web_public_url         = "http://validatorapi.stg.lumerin.io:8080"
}

# Create Cloudwatch Metrics & Dashboards
monitoring_frequency        = "rate(1 minute)"
financials_query_create     = "true"
proxy_router_query_create   = "true"
validator_query_create      = "true"
indexer_query_create        = "true"
monitoring_dashboard_create = "true"
eth_chain                   = "42161"

# Wallet Monitor Configuration
wallet_monitor_query_create = "true"
wallet_monitor_frequency    = "rate(15 minutes)"
wallet_monitor_query = {
  name                     = "bedrock-wallet-monitor"
  cw_namespace             = "wallet-monitor"
  lmr_token_address        = "0x0FC0c323Cf76E188654D63D62e668caBeC7a525b" # Arbitrum One LMR
  usdc_token_address       = "0xaf88d065e77c8cC2239327C5EDb3A432268e5831" # Arbitrum One USDC (native)
  alarm_evaluation_periods = 2
  alarm_period             = 900 # 15 minutes in seconds
}

# Wallets to monitor for ETH, USDC, and LMR balances (staging)
wallets_to_watch = [
  {
    walletName           = "Seller"
    walletId             = "0x99DFe1a2f99058B99FDc74177dBf53A93EBe3F48"
    eth_alarm_threshold  = 0.025
    # usdc_alarm_threshold = 200
    # lmr_alarm_threshold  = 1000
  },
  {
    walletName           = "Validator"
    walletId             = "0x06bA6986F7B71B9115670aedFE0de759b708d599"
    eth_alarm_threshold  = 0.025
    # usdc_alarm_threshold = 1
    # lmr_alarm_threshold  = 1000
  },
  {
    walletName           = "MarketMaker"
    walletId             = "0xdb8873E738C51eD3C59308ae666FB6bd9240D563"
    eth_alarm_threshold  = 0.025   # Market maker needs more ETH for gas
    # usdc_alarm_threshold = 40    # Market maker needs more USDC
    # lmr_alarm_threshold  = 1000
  },
  {
    walletName           = "OracleUpdater"
    walletId             = "0x67C1A7737e0C47E53FD4a828c9c7d81401ce912b"
    eth_alarm_threshold  = 0.025   
    # usdc_alarm_threshold = 1    
    # lmr_alarm_threshold  = 1000
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
  Environment     = "stg"
  Owner           = "aws-titanio-stg@titan.io" #AWS Account Email Address 092029861612 | aws-sandbox@titan.io | OrganizationAccountAccessRole 
  Scope           = "Global"
  CostCenter      = null
  Compliance      = null
  Classification  = null
  Repository      = "https://gitlab.com/TitanInd/bedrock/foundation-afs/proxy-router-foundation.git//bedrock/03-stg"
  ManagedBy       = "Terraform"
}

# Default Tags for Cloud Foundation resources
foundation_tags = {
  Name          = "Lumerin Proxy Router - STG"
  Capability    = null
  Application   = "Lumerin Proxy Router - STG"
  LifecycleDate = null
}
# USE1_1 Definition
# To use other regions or VPC: 
# 1. change find/replace the `use1_1` or `use1-1` designators with proper definition (eg, US West 2, 2nd VPC would be usw2_2 and usw2-2)
# 2. change provider aws.use1 to proper region eg: aws.usw2
# 3. change task image region in the locals below

################################
# LOCAL VARIABLES 
################################
locals {
  dns_name_daffy_use1_1_ext = "daffy."     #needs to inlude trailing "." for non prods
  dns_name_daffy_use1_1_int = "daffy-int." #needs to inlude trailing "." for non prods
  node_set_daffy_use1_1     = "a"
}
output "daffy_proxyrouter_remote" { value = var.special_nodes["qa_daffy_create"] ? "mssh -t ${module.daffy_use1_1[0].uninode_id[0]} -u ${var.account_shortname} titanadmin@${aws_route53_record.daffy_int_use1_1[0].name}" : "" }
output "daffy_proxyrouter_external" { value = var.special_nodes["qa_daffy_create"] ? "${aws_route53_record.daffy_ext_use1_1[0].name}:3333" : "" }

################################
# ProxyRouter NODE  
module "daffy_use1_1" {
  count              = var.special_nodes["qa_daffy_create"] ? 1 : 0
  source             = "git::ssh://git@gitlab.com-titan/TitanInd/bedrock/foundation-modules.git//uninode"
  providers          = { aws = aws.use1 }
  account_shortname  = var.account_shortname
  account_number     = var.account_number
  region_shortname   = var.region_shortname
  node_keypair       = var.devops_keypair
  node_protect       = false
  node_name_int      = "qa-daffy" #nodename root of instance(s)
  node_name_ext      = "qa-daffyext"
  dns_domain_name    = "lumerin" # ["titan", "lumerin", "turnip", "others..."]
  dns_create_ext     = true
  node_type          = "t3a.large"
  node_os            = "ubuntu" #[ubuntu, amlinux, others tbd] triggers lookup
  boot_vol_size      = "30"
  node_admin         = var.node_admin
  node_admin_pubkey  = var.titanadmin_pubkey
  vpc_index          = "1"                                # Which VPC in the Region selected? VPC identified by name acct-region-#
  subnet_zone        = "edge"                             #[edge, middle, private]
  node_qty_placement = ["${local.node_set_daffy_use1_1}"] # ["a", "b", "c", "b"] - module will count # of items and create in each az as specified 
  sg_rules           = ["outb-all", "remt-acc", "lumn-all", "weba-all"]
  instance_role      = "bedrock-foundation"
  node_identity = templatefile("build/desktop_ubu_v2.tftpl",
    {
      node_admin        = var.node_admin
      node_password     = var.node_password
      bedrock_glpat     = var.bedrock_glpat
      update_command    = "apt-get" # "yum" for AWS Linux
      second_vol_create = false
    }
  )
  node_tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      "Capability"         = "Hosting",
      "QSConfigName-2s3c1" = "Titan-Patch-Policy",
      "QSConfigName-c159s" = "WeeklyPatch-All"
    },
  )
}

############# EXTERNAL Route53 to node #############

resource "aws_route53_record" "daffy_ext_use1_1" {
  count    = var.special_nodes["qa_daffy_create"] ? 1 : 0
  provider = aws.special-dns
  zone_id  = var.account_lifecycle == "prd" ? data.aws_route53_zone.public_lumerin_root.zone_id : data.aws_route53_zone.public_lumerin.zone_id
  name     = var.account_lifecycle == "prd" ? "${local.dns_name_daffy_use1_1_ext}${data.aws_route53_zone.public_lumerin_root.name}" : "${local.dns_name_daffy_use1_1_ext}${data.aws_route53_zone.public_lumerin.name}"
  type     = "A"
  ttl      = "300"
  records  = [module.daffy_use1_1[0].uninode_ext_ip[0]]
}

############# INTERNAL Route53 to node #############

# # ########## DNS Record local domain -internal
resource "aws_route53_record" "daffy_int_use1_1" {
  count    = var.special_nodes["qa_daffy_create"] ? 1 : 0
  provider = aws.special-dns
  zone_id  = var.account_lifecycle == "prd" ? data.aws_route53_zone.public_lumerin_root.zone_id : data.aws_route53_zone.public_lumerin.zone_id
  name     = var.account_lifecycle == "prd" ? "${local.dns_name_daffy_use1_1_int}${data.aws_route53_zone.public_lumerin_root.name}" : "${local.dns_name_daffy_use1_1_int}${data.aws_route53_zone.public_lumerin.name}"
  type     = "A"
  ttl      = "300"
  records  = [module.daffy_use1_1[0].uninode_ip[0]]
}
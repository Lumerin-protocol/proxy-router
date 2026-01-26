
################################################################################
# APP-SPECIFIC GLOBAL LOOKUPS (data files, dns, iam, etc...)
################################################################################


################################################################################
# DEVOPS/BEDROCK SOURCE INFO
################################################################################

################################
# DNS Lookups 
################################
data "aws_route53_zone" "public_default_root" {
  provider     = aws.titanio-prd
  name         = local.target_domain
  private_zone = false
}

data "aws_route53_zone" "public_lumerin_root" {
  provider     = aws.titanio-prd
  name         = "lumerin.io"
  private_zone = false
}

data "aws_route53_zone" "public_titan_root" {
  provider     = aws.titanio-prd
  name         = "titan.io"
  private_zone = false
}

data "aws_route53_zone" "public_default" {
  provider     = aws.use1
  name         = "${substr(var.account_shortname, 8, 3)}.${local.target_domain}"
  private_zone = false
}

data "aws_route53_zone" "public_lumerin" {
  provider     = aws.use1
  name         = "${substr(var.account_shortname, 8, 3)}.lumerin.io"
  private_zone = false
}

data "aws_route53_zone" "public_titan" {
  provider     = aws.use1
  name         = "${substr(var.account_shortname, 8, 3)}.titan.io"
  private_zone = false
}
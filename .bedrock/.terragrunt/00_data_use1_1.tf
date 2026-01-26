
################################
# Regional DATA LOOKUPS 
################################
# Find the xxx.pool.titan.io Certificate created in foundation-extra
data "aws_acm_certificate" "proxy_router_ext" {
  count       = var.proxy_router["create"] ? 1 : 0
  provider    = aws.use1
  domain      = var.account_lifecycle == "prd" ? data.aws_route53_zone.public_default_root.name : data.aws_route53_zone.public_default.name
  types       = ["AMAZON_ISSUED"]
  most_recent = true
}

# WAF Protection 
data "aws_wafv2_web_acl" "bedrock_waf" {
  count    = var.proxy_router["create"] ? 1 : 0
  provider = aws.use1
  name     = "waf-bedrock-use1-1"
  scope    = "REGIONAL"
}

# Get Security Group IDs for Standard Bedrock SGs 
data "aws_security_group" "proxy_router_int_alb" {
  provider = aws.use1
  for_each = toset(local.proxy_router_int_alb)
  vpc_id   = data.aws_vpc.use1_1.id
  name     = "bedrock-${each.key}"
  # to use, define local list of strings with sec group suffixes 
  # in code for sgs, use the following: vpc_security_group_ids = [for s in data.aws_security_group.alb_sg : s.id]
}

data "aws_security_group" "proxy_service_sg" {
  provider = aws.use1
  for_each = toset(local.proxy_service_sg)
  vpc_id   = data.aws_vpc.use1_1.id
  name     = "bedrock-${each.key}"
  # to use, define local list of strings with sec group suffixes 
  # in code for sgs, use the following: vpc_security_group_ids = [for s in data.aws_security_group.proxy_router_sg_ecs : s.id]
}

data "aws_security_group" "proxy_query" {
  provider = aws.use1
  for_each = toset(local.proxy_router_query)
  vpc_id   = data.aws_vpc.use1_1.id
  name     = "bedrock-${each.key}"
  # to use, define local list of strings with sec group suffixes 
  # in code for sgs, use the following: vpc_security_group_ids = [for s in data.aws_security_group.proxy_query : s.id]
}

# Get Network Details 
data "aws_vpc" "use1_1" {
  provider = aws.use1
  tags = {
    Name = "vpc-${var.region_shortname}-${var.vpc_index}-${var.account_shortname}"
  }
}
data "aws_internet_gateway" "use1_1" {
  provider = aws.use1
  filter {
    name   = "attachment.vpc-id"
    values = [data.aws_vpc.use1_1.id]
  }
}

data "aws_subnet" "edge" {
  provider = aws.use1
  count    = 3
  filter {
    name   = "tag:Name"
    values = ["sn-use1-1-${var.account_shortname}-edge-${count.index + 1}"]
  }
  # in code for sgs, use the following: subnet_ids = [for n in data.aws_subnet.edge : n.id]
}

data "aws_subnet" "middle" {
  provider = aws.use1
  count    = 3
  filter {
    name   = "tag:Name"
    values = ["sn-use1-1-${var.account_shortname}-middle-${count.index + 1}"]
  }
  # in code for sgs, use the following: subnet_ids = [for n in data.aws_subnet.middle : n.id]
}

data "aws_subnet" "private" {
  provider = aws.use1
  count    = 3
  filter {
    name   = "tag:Name"
    values = ["sn-use1-1-${var.account_shortname}-private-${count.index + 1}"]
  }
  # in code for sgs, use the following: subnet_ids = [for n in data.aws_subnet.private : n.id]
}

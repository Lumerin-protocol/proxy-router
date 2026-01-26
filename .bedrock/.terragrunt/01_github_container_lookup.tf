################################################################################
# GITHUB REPOSITORY TAGS LOOKUP
# Queries GitHub repository tags to get latest tag for current environment
# Uses public API (no authentication required)
################################################################################

################################
# Data source to lookup GitHub repository tags for this environment only
################################
data "http" "github_repo_tags" {
  count = var.proxy_router["create"] ? 1 : 0
  url   = "https://api.github.com/repos/lumerin-protocol/proxy-router/tags"

  request_headers = {
    Accept = "application/vnd.github+json"
  }

  lifecycle {
    postcondition {
      condition     = self.status_code == 200
      error_message = "Failed to fetch GitHub repository tags (status: ${self.status_code}). Check repository name and network connectivity."
    }
  }
}

################################
# Local values for tag extraction
################################
locals {
  # Parse repository tags from GitHub API response
  # API returns array of objects with "name" field for each tag
  github_tags_raw = var.proxy_router["create"] ? try(jsondecode(data.http.github_repo_tags[0].response_body), []) : []

  # Regex pattern for this environment
  tag_pattern = var.account_lifecycle == "dev" ? "-dev$" : var.account_lifecycle == "stg" ? "-stg$" : "^v[0-9]"

  # Extract tags matching this environment only
  # GitHub returns tags in reverse chronological order (newest first)
  github_tags_filtered = [
    for tag in local.github_tags_raw :
    tag.name
    if can(regex(local.tag_pattern, tag.name)) &&
    # For lmn, also exclude dev/stg tags
    (var.account_lifecycle != "lmn" || !can(regex("-(dev|stg)$", tag.name)))
  ]

  # Get the latest tag for this environment (first in list is most recent)
  github_latest_tag = length(local.github_tags_filtered) > 0 ? local.github_tags_filtered[0] : null

  # Determine which image tag to use:
  # - If image_tag is "auto" or empty, use GitHub lookup (must succeed)
  # - If image_tag is a specific version, use that (allows pinning for rollback/testing)
  proxy_router_image_tag = var.proxy_router["create"] ? (
    var.proxy_router["image_tag"] == "auto" || var.proxy_router["image_tag"] == "" ?
    local.github_latest_tag :
    var.proxy_router["image_tag"]
  ) : ""

  proxy_validator_image_tag = var.proxy_validator["create"] ? (
    var.proxy_validator["image_tag"] == "auto" || var.proxy_validator["image_tag"] == "" ?
    local.github_latest_tag :
    var.proxy_validator["image_tag"]
  ) : ""
}

################################
# Outputs
################################
output "github_latest_tag" {
  value       = var.proxy_router["create"] ? local.github_latest_tag : null
  description = "Latest GitHub container tag for this environment"
}

output "github_tags_available" {
  value       = var.proxy_router["create"] ? local.github_tags_filtered : null
  description = "All available GitHub container tags for this environment"
}


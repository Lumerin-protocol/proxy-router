locals {
  env_name   = substr(var.account_shortname, length(var.account_shortname) - 3, length(var.account_shortname))
  env_period = 300
}

resource "aws_cloudwatch_dashboard" "auto_lumerin" {
  count          = var.monitoring_dashboard_create ? 1 : 0
  provider       = aws.use1
  dashboard_name = "Lumerin-${upper(local.env_name)}-Monitor"
  # lifecycle {
  #   ignore_changes = [
  #     dashboard_body
  #   ]
  # }

  dashboard_body = jsonencode({
    start           = "-PT12H"
    period_override = "inherit"
    widgets = [
      # Markdown widget
      {
        type   = "text"
        x      = 0
        y      = 0
        width  = 8
        height = 5
        properties = {
          markdown = "# Lumerin-${upper(local.env_name)} Seller & Validator Dashboard \n## Optimization Goals: \n* Push contract and Hashrate consumption (Offered vs Consumed) to 100% using pricing \n* Keep Offered vs Available below 80% (for now) to absorb ASIC issues and/or restart"
        }
      },
      # Seller: Hashrate Available, Offered, Purchased & Used 
      {
        type   = "metric"
        x      = 8
        y      = 0
        width  = 8
        height = 5
        properties = {
          title                = "Seller: Hashrate Rates",
          setPeriodToTimeRange = true
          metrics = [
            [{ "color" : "#ff7f0e", "expression" : "(m2/m1)*100", "id" : "e1", "label" : "Offered vs Available (<95%)", "region" : "us-east-1" }],
            [{ "expression" : "(m4/m3)*100", "label" : "Used vs Purchased (~100%)", "id" : "e2", "color" : "#2ca02c", "region" : "us-east-1" }],
            ["proxy-router", "hashrate_available", { "id" : "m1", "region" : "us-east-1", "visible" : false }],
            ["proxy-router", "hashrate_offered", { "id" : "m2", "region" : "us-east-1", "visible" : false }],
            ["proxy-router", "hashrate_purchased", { "color" : "#1f77b4", "label" : "Purchased (active contracts)", "region" : "us-east-1", "id" : "m3", "visible" : false }],
            ["proxy-router", "hashrate_used", { "color" : "#aec7e8", "label" : "Used", "region" : "us-east-1", "id" : "m4", "visible" : false }]
          ],
          view                 = "timeSeries",
          sparkline            = false,
          stacked              = false,
          setPeriodToTimeRange = true,
          region               = "us-east-1",
          stat                 = "Average",
          period               = local.env_period,
          yAxis = {
            left = {
              min       = 90,
              max       = 105,
              label     = "%"
              showUnits = false
            }
          },
          annotations = {
            horizontal = [
              {
                "color" : "#98df8a",
                "value" : 100
              },
              {
                "color" : "#ffbb78",
                "value" : 95,
                "fill" : "below"
              }
            ]
          }
        }
      },
      # ETH Balances
      {
        type   = "metric"
        x      = 16
        y      = 0
        width  = 8
        height = 5
        properties = {
          metrics = [
            ["wallet-monitor", "eth_balance", "WalletName", "Validator", { "region" : "us-east-1" }],
            ["wallet-monitor", "eth_balance", "WalletName", "Seller", { "region" : "us-east-1" }],
            ["wallet-monitor", "eth_balance", "WalletName", "OracleUpdater", { "region" : "us-east-1" }],
            ["wallet-monitor", "eth_balance", "WalletName", "MarketMaker", { "region" : "us-east-1" }],
          ],
          view    = "timeSeries",
          stacked = false,
          region  = "us-east-1",
          stat    = "Average",
          period  = 900,
          yAxis = {
            left = {
              min = 0
            }
          },
          "annotations" : {
            "horizontal" : [
              {
                "color" : "#d62728",
                "label" : "Low",
                "value" : 0.025,
                "fill" : "below"
              }
            ]
          }
          title                = "ETH Balances",
          setPeriodToTimeRange = true
        }
      },
      # Seller Consumption
      {
        type   = "metric"
        x      = 0
        y      = 5
        width  = 16
        height = 6
        properties = {
          title     = "Seller: Consumption"
          liveData  = true
          region    = "us-east-1"
          view      = "singleValue"
          sparkline = true
          period    = local.env_period
          stat      = "Average"
          metrics = [
            ["proxy-router", "hashrate_available", { "color" : "#ff7f0e", "label" : "HR - Available", "region" : "us-east-1" }],
            ["proxy-router", "hashrate_offered", { "color" : "#ff7f0e", "label" : "HR - Offered", "region" : "us-east-1" }],
            ["proxy-router", "hashrate_purchased", { "color" : "#ffbb78", "label" : "HR - Purchased", "region" : "us-east-1" }],
            ["proxy-router", "buyers_unique", { "color" : "#1f77b4", "label" : "Buyers (unique)", "region" : "us-east-1" }],
            ["proxy-router", "contracts_offered", { "color" : "#9467bd", "label" : "Contracts Offered", "region" : "us-east-1" }],
            ["proxy-router", "contracts_active", { "color" : "#c5b0d5", "label" : "Contracts Active", "region" : "us-east-1" }]
          ]
        }
      },
      # Seller: Consumption % 
      {
        type   = "metric"
        x      = 16
        y      = 0
        width  = 8
        height = 6
        properties = {
          view                 = "bar",
          title                = "Seller: Consumption %",
          region               = "us-east-1",
          liveData             = true,
          stat                 = "Maximum",
          period               = local.env_period,
          trend                = true,
          setPeriodToTimeRange = false,
          metrics = [
            [{ "expression" : "(m4/m5)*100", "id" : "e2", "label" : "Hashrate Consumption", "region" : "us-east-1", "color" : "#ffbb78" }],
            [{ "expression" : "(m2/m1)*100", "id" : "e1", "label" : "Contract Consumption", "region" : "us-east-1", "color" : "#c5b0d5" }],
            [{ "expression" : "(m4/m3)*100", "id" : "e3", "label" : "Hashrate Used vs Purchased", "region" : "us-east-1", "color" : "#98df8a" }],
            ["proxy-router", "contracts_offered", { "color" : "#ff7f0e", "id" : "m1", "label" : "contracts_offered", "region" : "us-east-1", "visible" : false }],
            ["proxy-router", "contracts_active", { "color" : "#ff7f0e", "id" : "m2", "label" : "contracts_active", "region" : "us-east-1", "visible" : false }],
            ["proxy-router", "hashrate_purchased", { "id" : "m3", "region" : "us-east-1", "visible" : false }],
            ["proxy-router", "hashrate_used", { "id" : "m4", "region" : "us-east-1", "visible" : false }],
            ["proxy-router", "hashrate_offered", { "id" : "m5", "region" : "us-east-1", "visible" : false }],
            ["proxy-router", "hashrate_available", { "id" : "m6", "region" : "us-east-1", "visible" : false }]
          ]
          yAxis = {
            left = {
              min = 0,
              max = 100
            }
          }
          annotations = {
            horizontal = [
              {
                "color" : "#2ca02c",
                "fill" : "above",
                "label" : "Goal",
                "value" : 95
              },
              {
                "color" : "#f89256",
                "fill" : "below",
                "label" : "Low",
                "value" : 20
              }
            ]
          }
        }
      },
      # Seller Warnings & Errors 
      {
        type   = "metric"
        x      = 0
        y      = 11
        width  = 8
        height = 6
        properties = {
          metrics = [
            ["proxy-router", "seller_warn", { "label" : "WARN", "color" : "#ff7f0e", "region" : "us-east-1" }],
            ["proxy-router", "seller_error", { "label" : "ERROR", "color" : "#d62728", "region" : "us-east-1" }]
          ],
          view      = "timeSeries",
          sparkline = true,
          stacked   = true,
          region    = "us-east-1",
          stat      = "Sum",
          period    = local.env_period,
          yAxis = {
            left = {
              min = 0
            },
            right = {
              min = 0
            }
          },
          title                = "Seller: Warnings & Errors",
          setPeriodToTimeRange = true
        }
      },
      # Validator Warnings & Errors 
      {
        type   = "metric"
        x      = 8
        y      = 11
        width  = 8
        height = 6
        properties = {
          metrics = [
            ["proxy-validator", "validator_warn", { "label" : "WARN", "color" : "#ff7f0e", "region" : "us-east-1" }],
            ["proxy-validator", "validator_error", { "label" : "ERROR", "color" : "#d62728", "region" : "us-east-1" }]
          ],
          view      = "timeSeries",
          sparkline = true,
          stacked   = true,
          region    = "us-east-1",
          stat      = "Sum",
          period    = local.env_period,
          yAxis = {
            left = {
              min = 0
            },
            right = {
              min = 0
            }
          },
          title                = "Validator: Warnings & Errors",
          setPeriodToTimeRange = true
        }
      },
      # Validator: Key Stats 
      {
        type   = "metric"
        x      = 16
        y      = 11
        width  = 8
        height = 6
        properties = {
          title                = "Validator: Statistics",
          setPeriodToTimeRange = true,
          view                 = "timeSeries",
          sparkline            = true,
          stacked              = false,
          region               = "us-east-1",
          stat                 = "Average",
          period               = local.env_period,
          metrics = [
            ["proxy-validator", "buyers_unique", { "label" : "Buyers" }],
            ["proxy-validator", "contracts_active", { "label" : "Contracts", "color" : "#2ca02c" }],
            ["proxy-validator", "hashrate_purchased", { "yAxis" : "right", "label" : "HR Purchased", "color" : "#9467bd" }],
            ["proxy-validator", "hashrate_actual", { "yAxis" : "right", "label" : "HR Actual", "color" : "#c5b0d5" }]
          ],
          yAxis = {
            "left" : {
              "showUnits" : false,
              "label" : "Count",
              "min" : 0
            },
            "right" : {
              "showUnits" : false,
              "label" : "PH/s",
              "min" : 0
            }
          }
        }
      },
      # Seller Key Issues 
      {
        type   = "metric"
        x      = 0
        y      = 17
        width  = 8
        height = 6
        properties = {
          metrics = [
            ["proxy-router", "seller_consequentinvalidshares", { "region" : "us-east-1", "label" : "ConsInvalidShares", "color" : "#1f77b4" }],
            ["proxy-router", "seller_invalid_work", { "region" : "us-east-1", "label" : "InvalidWork", "color" : "#ff7f0e" }],
            ["proxy-router", "seller_job_not_found", { "region" : "us-east-1", "label" : "JobNotFound", "color" : "#ffbb78" }],
            ["proxy-router", "seller_lowdiff", { "region" : "us-east-1", "label" : "LowDiff", "color" : "#98df8a" }],
            ["proxy-router", "seller_failedtoconnect", { "region" : "us-east-1", "label" : "FailedToConnect", "color" : "#c5b0d5" }],
            ["proxy-router", "contracts_active", { "stat" : "Maximum", "yAxis" : "right", "region" : "us-east-1", "label" : "ActiveContracts", "color" : "#7f7f7f" }]
          ],
          view      = "timeSeries",
          sparkline = false,
          stacked   = false,
          region    = "us-east-1",
          stat      = "Sum",
          period    = local.env_period,
          yAxis = {
            left = {
              min = 0
            },
            right = {
              min = 0
            }
          },
          title                = "Seller: Key Issues",
          setPeriodToTimeRange = true
        }
      },
      # Validator Key Issues 
      {
        type   = "metric"
        x      = 8
        y      = 17
        width  = 8
        height = 6
        properties = {
          metrics = [
            ["proxy-validator", "validator_consequentinvalidshares", { "region" : "us-east-1", "label" : "ConsInvalidShares", "color" : "#1f77b4" }],
            ["proxy-validator", "validator_invalid_work", { "region" : "us-east-1", "label" : "InvalidWork", "color" : "#ff7f0e" }],
            ["proxy-validator", "validator_job_not_found", { "region" : "us-east-1", "label" : "JobNotFound", "color" : "#ffbb78" }],
            ["proxy-validator", "validator_lowdiff", { "region" : "us-east-1", "label" : "LowDiff", "color" : "#98df8a" }],
            ["proxy-validator", "validator_failedtoconnect", { "region" : "us-east-1", "label" : "FailedToConnect", "color" : "#c5b0d5" }],
            ["proxy-validator", "contracts_active", { "stat" : "Maximum", "yAxis" : "right", "region" : "us-east-1", "label" : "ActiveContracts", "color" : "#7f7f7f" }],
            ["proxy-validator", "validator_contract_cancelled", { "yAxis" : "right", "region" : "us-east-1", "label" : "ContractsCancelled", "color" : "#d62728" }]

          ],
          view      = "timeSeries",
          sparkline = false,
          stacked   = false,
          region    = "us-east-1",
          stat      = "Sum",
          period    = local.env_period,
          yAxis = {
            left = {
              min = 0
            },
            right = {
              min = 0
            }
          },
          title                = "Validator: Key Issues",
          setPeriodToTimeRange = true
        }
      },
      # Seller: Miners 
      {
        type   = "metric"
        x      = 16
        y      = 17
        width  = 8
        height = 6
        properties = {
          title                = "Seller: Miners",
          setPeriodToTimeRange = true,
          view                 = "timeSeries",
          sparkline            = true,
          stacked              = true,
          region               = "us-east-1",
          stat                 = "Average",
          period               = local.env_period,
          metrics = [
            ["proxy-router", "miners_vetting", { "id" : "m9", "label" : "Vetting", "color" : "#ff7f0e", "region" : "us-east-1" }],
            ["proxy-router", "miners_free", { "id" : "m7", "label" : "Free", "color" : "#2ca02c", "region" : "us-east-1" }],
            ["proxy-router", "miners_partial", { "id" : "m8", "region" : "us-east-1", "color" : "#aec7e8", "label" : "Partial" }],
            ["proxy-router", "miners_busy", { "id" : "m6", "region" : "us-east-1", "color" : "#1f77b4", "label" : "Busy" }]
          ],
          yAxis = {
            left = {
              min = 0
            }
          }
        }
      },
      # Seller Node Health 
      {
        type   = "metric"
        x      = 0
        y      = 23
        width  = 12
        height = 6
        properties = {
          view                 = "singleValue",
          title                = "Seller: Node Health",
          sparkline            = true,
          stacked              = false,
          region               = "us-east-1",
          liveData             = false,
          stat                 = "Sum",
          period               = 60,
          trend                = true,
          setPeriodToTimeRange = false,
          metrics = [
            [{ "id" : "expr1m0", "label" : "CPU (%)", "expression" : "(mm1m0/mm0m0)*100", "stat" : "Average", "region" : "us-east-1", "color" : "#9467bd" }],
            ["ECS/ContainerInsights", "CpuReserved", "ClusterName", "ecs-proxy-router-${local.env_name}-use1", "ServiceName", "svc-proxy-router-${local.env_name}-use1", { "id" : "mm0m0", "visible" : false, "region" : "us-east-1", "stat" : "Average" }],
            ["ECS/ContainerInsights", "CpuUtilized", "ClusterName", "ecs-proxy-router-${local.env_name}-use1", "ServiceName", "svc-proxy-router-${local.env_name}-use1", { "id" : "mm1m0", "visible" : false, "region" : "us-east-1", "stat" : "Average" }],
            [{ "id" : "expr2m0", "label" : "MEM (%)", "expression" : "(mm3m0/ mm2m0)*100", "stat" : "Average", "region" : "us-east-1", "color" : "#98df8a" }],
            ["ECS/ContainerInsights", "MemoryReserved", "ClusterName", "ecs-proxy-router-${local.env_name}-use1", "ServiceName", "svc-proxy-router-${local.env_name}-use1", { "id" : "mm2m0", "visible" : false, "region" : "us-east-1", "stat" : "Average" }],
            ["ECS/ContainerInsights", "MemoryUtilized", "ClusterName", "ecs-proxy-router-${local.env_name}-use1", "ServiceName", "svc-proxy-router-${local.env_name}-use1", { "id" : "mm3m0", "visible" : false, "region" : "us-east-1", "stat" : "Average" }],
            [{ "id" : "expr3m0", "label" : "NET (%)", "expression" : "(mm4m0 + mm5m0)/1000000", "stat" : "Average", "region" : "us-east-1", "yAxis" : "right", "color" : "#ffbb78" }],
            ["ECS/ContainerInsights", "NetworkRxBytes", "ClusterName", "ecs-proxy-router-${local.env_name}-use1", "ServiceName", "svc-proxy-router-${local.env_name}-use1", { "id" : "mm4m0", "visible" : false, "region" : "us-east-1", "stat" : "Average" }],
            ["ECS/ContainerInsights", "NetworkTxBytes", "ClusterName", "ecs-proxy-router-${local.env_name}-use1", "ServiceName", "svc-proxy-router-${local.env_name}-use1", { "id" : "mm5m0", "visible" : false, "region" : "us-east-1", "stat" : "Average" }],
            ["ECS/ContainerInsights", "RunningTaskCount", "ClusterName", "ecs-proxy-router-${local.env_name}-use1", "ServiceName", "svc-proxy-router-${local.env_name}-use1", { "id" : "m2", "label" : "Task Count", "stat" : "Average", "region" : "us-east-1" }]
          ]
        }
      },
      # Validator Node Health 
      {
        type   = "metric"
        x      = 12
        y      = 23
        width  = 12
        height = 6
        properties = {
          view                 = "singleValue",
          title                = "Validator: Node Health",
          sparkline            = true,
          stacked              = false,
          region               = "us-east-1",
          liveData             = false,
          stat                 = "Sum",
          period               = 60,
          trend                = true,
          setPeriodToTimeRange = false,
          metrics = [
            [{ "id" : "expr1m0", "label" : "CPU (%)", "expression" : "(mm1m0 / mm0m0)*100", "stat" : "Average", "region" : "us-east-1", "color" : "#9467bd" }],
            ["ECS/ContainerInsights", "CpuReserved", "ClusterName", "ecs-proxy-router-${local.env_name}-use1", "ServiceName", "svc-proxy-validator-${local.env_name}-use1", { "id" : "mm0m0", "visible" : false, "region" : "us-east-1", "stat" : "Average" }],
            ["ECS/ContainerInsights", "CpuUtilized", "ClusterName", "ecs-proxy-router-${local.env_name}-use1", "ServiceName", "svc-proxy-validator-${local.env_name}-use1", { "id" : "mm1m0", "visible" : false, "region" : "us-east-1", "stat" : "Average" }],
            [{ "id" : "expr2m0", "label" : "MEM (%) ", "expression" : "(mm3m0/ mm2m0)*100", "stat" : "Average", "region" : "us-east-1", "color" : "#98df8a" }],
            ["ECS/ContainerInsights", "MemoryReserved", "ClusterName", "ecs-proxy-router-${local.env_name}-use1", "ServiceName", "svc-proxy-validator-${local.env_name}-use1", { "id" : "mm2m0", "visible" : false, "region" : "us-east-1", "stat" : "Average" }],
            ["ECS/ContainerInsights", "MemoryUtilized", "ClusterName", "ecs-proxy-router-${local.env_name}-use1", "ServiceName", "svc-proxy-validator-${local.env_name}-use1", { "id" : "mm3m0", "visible" : false, "region" : "us-east-1", "stat" : "Average" }],
            [{ "id" : "expr3m0", "label" : "NET (%)", "expression" : "(mm4m0 + mm5m0)/1000000", "stat" : "Average", "region" : "us-east-1", "yAxis" : "right", "color" : "#ffbb78" }],
            ["ECS/ContainerInsights", "NetworkRxBytes", "ClusterName", "ecs-proxy-router-${local.env_name}-use1", "ServiceName", "svc-proxy-validator-${local.env_name}-use1", { "id" : "mm4m0", "visible" : false, "region" : "us-east-1", "stat" : "Average" }],
            ["ECS/ContainerInsights", "NetworkTxBytes", "ClusterName", "ecs-proxy-router-${local.env_name}-use1", "ServiceName", "svc-proxy-validator-${local.env_name}-use1", { "id" : "mm5m0", "visible" : false, "region" : "us-east-1", "stat" : "Average" }],
            ["ECS/ContainerInsights", "RunningTaskCount", "ClusterName", "ecs-proxy-router-${local.env_name}-use1", "ServiceName", "svc-proxy-validator-${local.env_name}-use1", { "id" : "m2", "label" : "Task Count", "stat" : "Average", "region" : "us-east-1" }]
          ]
        }
      },

      # Seller: Miners Statistics  
      {
        type   = "metric"
        x      = 0
        y      = 29
        width  = 12
        height = 6
        properties = {
          title                = "Seller: Miners Statistics",
          setPeriodToTimeRange = true
          metrics = [
            ["proxy-router", "miners_average_difficulty", { "color" : "#9467bd", "yAxis" : "right", "label" : "Average Diff", "id" : "m1", "region" : "us-east-1" }],
            ["proxy-router", "miners_accepted_shares", { "color" : "#2ca02c", "label" : "Shares Accepted", "id" : "m2", "region" : "us-east-1" }],
            [{ "expression" : "m3+m4+m5", "label" : "Shares Rejected", "id" : "e1", "color" : "#d62728", "region" : "us-east-1" }],
            ["proxy-router", "miners_accepted_they_rejected", { "visible" : false, "id" : "m3", "region" : "us-east-1" }],
            ["proxy-router", "miners_rejected_shares", { "visible" : false, "id" : "m4", "region" : "us-east-1" }],
            ["proxy-router", "miners_rejected_they_accepted", { "visible" : false, "id" : "m5", "region" : "us-east-1" }]
          ],
          view                 = "timeSeries",
          sparkline            = false,
          stacked              = false,
          setPeriodToTimeRange = true,
          region               = "us-east-1",
          stat                 = "Average",
          period               = local.env_period,
          yAxis = {
            "left" : {
              "min" : 0,
              "showUnits" : false,
              "label" : "Shares"
            },
            "right" : {
              "min" : 0,
              "showUnits" : false,
              "label" : "Difficulty"
            }
          }
        }
      },
      # Validator: Miners Statistics  
      {
        type   = "metric"
        x      = 12
        y      = 29
        width  = 12
        height = 6
        properties = {
          title                = "Validator: Miners Statistics",
          setPeriodToTimeRange = true
          metrics = [
            ["proxy-validator", "miners_average_difficulty", { "color" : "#9467bd", "yAxis" : "right", "label" : "Average Diff", "id" : "m1", "region" : "us-east-1" }],
            ["proxy-validator", "miners_accepted_shares", { "color" : "#2ca02c", "label" : "Shares Accepted", "id" : "m2", "region" : "us-east-1" }],
            [{ "expression" : "m3+m4+m5", "label" : "Shares Rejected", "id" : "e1", "color" : "#d62728", "region" : "us-east-1" }],
            ["proxy-validator", "miners_accepted_they_rejected", { "visible" : false, "id" : "m3", "region" : "us-east-1" }],
            ["proxy-validator", "miners_rejected_shares", { "visible" : false, "id" : "m4", "region" : "us-east-1" }],
            ["proxy-validator", "miners_rejected_they_accepted", { "visible" : false, "id" : "m5", "region" : "us-east-1" }]
          ],
          view                 = "timeSeries",
          sparkline            = false,
          stacked              = false,
          setPeriodToTimeRange = true,
          region               = "us-east-1",
          stat                 = "Average",
          period               = local.env_period,
          yAxis = {
            "left" : {
              "min" : 0,
              "showUnits" : false,
              "label" : "Shares"
            },
            "right" : {
              "min" : 0,
              "showUnits" : false,
              "label" : "Difficulty"
            }
          }
        }
      }
    ]
  })
}
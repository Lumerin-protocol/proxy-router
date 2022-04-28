# Hashrouter

POC for controlling a miner through a smart contract

Usage:
  -contract.addr string
        Address of smart contract that node is servicing (default is set for testing only)
  -ethNode.addr string
        Address of Ethereum RPC node to connect to via websocket (default "wss://ropsten.infura.io/ws/v3/4b68229d56fe496e899f07c3d41cb08a")
  -file.log
        true or false - whether to output logs to a file named 'logs' (default true)
  -pool.addr string
        Address and port for mining pool (default "mining.staging.pool.titan.io:4242")
  -stratum.addr string
        Address and port for stratum (default "0.0.0.0:3333")
  -syslog
        On true adapt log to out in syslog, hide date and colors
  -web.addr string
        Address and port for api web server (default "0.0.0.0:8080")
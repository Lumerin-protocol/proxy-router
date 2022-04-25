
# Play Center Setup

* log into test aws box with `./remote-test`
* if you want to use ssh to direct miner traffic locally, you'll need to copy your public ssh key to .ssh/authorized_keys
* run `openvpn --config ./ovpn.config`
* to create a remote tunnel with ssh: `ssh titanadmin@10.32.3.248 -L 8080:192.168.5.16:80 -R 3333:0.0.0.0:3333`; or you can use ngrok
* point the miner to your server (ngrok, or aws box)
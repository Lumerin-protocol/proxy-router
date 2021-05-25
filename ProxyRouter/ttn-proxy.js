const net = require('net');
const http = require('http');
const fs = require('fs');
const uuid = require('uuid');
const { createLogger, format, transports } = require('winston');

const Web3 = require('web3');

const logger = createLogger(
{
  format: format.combine(
    format.splat(),
    format.timestamp({format:'YYYY-MM-DD HH:mm:ss'}),
    format.printf(function(info)
    {
        return `${info.timestamp} ${info.level.toUpperCase()}: ${info.message}`
            + (info.splat!==undefined?`${info.splat}`:" ");
    })
  ),
  transports: [
      new transports.Console(),
      new transports.File({
          filename:'output.log',
          maxsize: 1e6,
          maxFiles: 5,
          tailable: true,
          zippedArchive: true})
  ]
});


const config = JSON.parse(fs.readFileSync("config.json"));
const isVal = config.isVal;

let web3;


const proxy_id = process.argv.length > 2 ? Number(process.argv[2]) : 1;
const server = net.createServer();
const stale_timeout = 90e3;
const share_status = {
    pending:0,
    accepted:1,
    rejected:2
};
const clients = new Map();      // Header string to a PM or TPT session
const header_pool = new Map();  // PM header string to a PP session
const sock_pool = new Map();    // PP socket to a PP session
let upstream = null;            // LPT session



let listen_host = config.listen_host;
let listen_port = config.listen_port;


let pool_ip = config.default_pool.host;
let pool_port = config.default_pool.port;
let pool_user = config.default_pool.username;

server.on('connection', server_accept);

if (isVal)
{
    logger.info('Running as Validator');

    web3 = new Web3(new Web3.providers.WebsocketProvider("ws://" + config.node.host + ":" + config.node.port + "/"));
    block_listener();

}
else
{
    logger.info('Running Local Proxy');
    logger.info('Proxying to: %s', config.validator_host);

    server.listen(listen_port, listen_host, 6144, function()
    {
        var addr = server.address();
        logger.info('Listening on %s:%d', addr.address, addr.port);
    });
}

function header_buf(str)
{
    let hp = str.split(':');
    let o = hp[0].split('.').map(x => Number(x));
    let header = Buffer.allocUnsafe(8);
    let n = (o[0]<<24) | (o[1]<<16) | (o[2]<<8) | o[3];
    header.writeInt32BE(n, 0);
    header.writeInt32BE(Number(hp[1]), 4);
    return header;
}

function header_str(buf)
{
    if (buf.length < 8)
        return null;
    let n = buf.readInt32BE();
    let o = [n>>24, n>>16 & 0xff, n>>8 & 0xff, n & 0xff];
    let port = buf.readInt32BE(4);
    return o.join('.') + ':' + port;
}

function dotnet_guid_shuffle(arr)
{
    // reverse first 4 bytes, next 2 bytes and next 2 bytes
    // e.g.
    // c5608a2c-0128-1b49-a134-2876e54ceb5b
    // 2c8a60c5-2801-491b-a134-2876e54ceb5b
    arr.subarray(0, 4).reverse();
    arr.subarray(4, 6).reverse();
    arr.subarray(6, 8).reverse();
    return arr;
}

function Session(sock)
{
    this.sock = sock;
    this.trunk = null;
    this.header = null;
    this.expects_headers = false;
    this._partial = Buffer.allocUnsafe(0);
}

Session.prototype.hasHeader = function(buf)
{
    // As we're using IPv4 but 4 bytes for the port slot, we can be fast here
    // by just checking for a NULL in the port bytes...
    let pb = buf.slice(4, 8);
    let idx = pb.indexOf(0);
    return idx > -1;
};

Session.prototype.processLine = function(line)
{
    return line;
};

Session.prototype.process = function(chunk)
{
    // our default implementation just buffers lines...
    let sol = 0;
    let skip = 0;
    if (this._partial.length == 0 && chunk.length >= 8
        && this.hasHeader(chunk))
        this.expects_headers = true;
    if (this._partial.length == 0 && this.expects_headers)
        skip = 8;
    let eol = -1;
    let lines = [];
    let rv = Buffer.allocUnsafe(0);

    while ((eol = chunk.indexOf('\n', sol+skip)) != -1)
    {
        if (this._partial.length)
        {
            lines.push(Buffer.concat([this._partial, chunk.slice(sol,eol+1)]));
            this._partial = Buffer.allocUnsafe(0);
            if (this.expects_headers)
                skip = 8;
        }
        else
            lines.push(chunk.slice(sol,eol+1));
        sol = eol+1;
    }

    if (lines.length)
    {
        lines = lines.map(this.processLine, this);
        rv = Buffer.concat(lines);
        this._partial = Buffer.allocUnsafe(0);
    }

    if (sol < chunk.length)
    {
        this._partial = Buffer.concat([this._partial, chunk.slice(sol)]);
    }
    return rv;
};

function PoolSession(sock)
{
    Session.call(this, sock);
    this.pool_user = null;
    this.pool_id = 0;
    this.deviceGUID = null;
    this.workerName = null;
    this.submitIDs = [];
    this.last_diff = 1;
    this.UUID = uuid.v4();
}

PoolSession.prototype = Object.create(Session.prototype);
PoolSession.prototype.constructor = PoolSession;

PoolSession.prototype.processLine = function(line)
{
    // - rewrite worker/auth name
    // - capture submit IDs
    // - capture set_diff
    // - store shares/responses to redis

    if (line.length == 0)
        return line; // As per Jethro, return line

    if (this.pool_user == null)
        return line;

    try
    {
        let jo = JSON.parse(line.toString());
        let id = jo.id;
        let method = jo.method;
        let now = Math.floor(Date.now()/1000);
        let idx = -1;
        let start, end, key;
        switch (method)
        {
            case 'mining.authorize':
                this.deviceGUID = jo.params[0];
                this.workerName = this.pool_user + '.' + this.deviceGUID.replace(/-/g,'').substr(0,15);
                // rewrite the worker name
                idx = line.indexOf(this.deviceGUID);
                start = line.slice(0, idx);
                end = line.slice(idx + this.deviceGUID.length);
                return Buffer.concat([start, Buffer.from(this.workerName), end]);
            case 'mining.submit':
                idx = line.indexOf(this.deviceGUID);
                if (idx == -1)
                    return line;
                this.submitIDs.push(id);

                /*
                // push the submit to redis
                key = `shares:${this.deviceGUID}:${this.pool_id}:${this.UUID}:${id}`;
                redis.hmset(key, 'status', share_status.pending, 'difficulty', this.last_diff, 'timestamp', now);
                redis.expire(key, 43200);

                // and add/update the session hash
                key = `${this.deviceGUID}:session`;
                let that = this;
                redis.hget(key, 'start', function(err, res)
                {
                    if (res == null)
                        redis.hmset(key, 'start', now, 'difficulty', that.last_diff);
                });
                redis.expire(key, 600);
                */

                // rewrite the worker name
                start = line.slice(0, idx);
                end = line.slice(idx + this.deviceGUID.length);
                return Buffer.concat([start, Buffer.from(this.workerName), end]);
            case 'mining.set_difficulty':
                this.last_diff = jo.params[0];
                return line;
            case 'mining.configure':
                return line;
        }
        if (id == null)
            return line;
        idx = this.submitIDs.indexOf(id);
        if (idx != -1)
        {
            this.submitIDs.splice(idx, 1);

            /*
            // push the submit result to redis
            let share = Buffer.allocUnsafe(44);
            Buffer.from(dotnet_guid_shuffle(uuid.parse(this.deviceGUID))).copy(share);
            share.writeUInt32LE(pool_id, 16);
            Buffer.from(uuid.parse(this.UUID)).copy(share, 20);
            share.writeUInt32LE(id, 36);
            share.writeUInt32LE(this.last_diff, 40);
            let ar = jo.error == null ? 'accepted' : 'rejected';
            key = `shares:${ar}`;
            redis.zadd(key, now, share);
            // and update session hash
            key = `${this.deviceGUID}:session`;
            redis.hincrby(key, 'difficulty', this.last_diff);
            redis.expire(key, 600);
            // and update the share hash

            // As pet Jethro advice, hard code to bypass Redis 
            //key = `shares:${this.deviceGUID}:${pool_id}:${this.UUID}:${id}`;
            key = `shares:${this.deviceGUID}:$pool_id:$UUID:$id`; // Hard code it and Redis should still allow to be run

            if (jo.error != null || jo.result != true)
            {
                redis.hmset(key, 'status', share_status.rejected,
                    'error', JSON.stringify(jo.error));
            }
            else
            {
                redis.hset(key, 'status', share_status.accepted);
            }
            */
        }
    }
    catch (e) {}
    return line;
};

function server_accept(con)
{
    con.on('data', client_data);
    con.on('error', client_error);
    con.once('close', client_close);
    con.setNoDelay(true);
    con.setTimeout(stale_timeout, con.destroy);
    let ra = con.remoteAddress + ':' + con.remotePort;
    logger.info('New client: %s', ra);
    if (!isVal)
    {
        if (upstream == null)
        {
            logger.info('Creating our single upstream trunk to: %s', config.validator_host + ":" + config.validator_port);
            let sock = net.createConnection(Number(config.validator_port), config.validator_host);

            upstream = new Session(sock);
            upstream.expects_headers = true;

            sock.setNoDelay(true);
            sock.setTimeout(stale_timeout, () =>
            {
                sock.destroy();
                upstream = null;
            });
            sock.on('data', upstream_data);
            sock.on('end', upstream_end);
            sock.once('close', upstream_close);
            sock.on('error', upstream_error);
        }
        let pm = new Session(con);
        pm.trunk = upstream;
        pm.header = header_buf(ra);
        clients.set(ra, pm);
        logger.info('Proxy with pid: %d, has %d miners connected', process.pid, clients.size);
    }
    else
    {
        let tpt = new Session(con);
        tpt.header = header_buf(ra);
        clients.set(ra, tpt);
    }
}

function tpt_data(data)
{
    let ra = this.remoteAddress + ':' + this.remotePort;
    let tpt = clients.get(ra);
    let pd = tpt.process(data);
    let skip = tpt.expects_headers ? 8 : 0;
    if (pd.length == 0)
        return;
    let sol = 0;
    let eol = 0;
    do
    {
        eol = pd.indexOf('\n', sol+skip);
        let hb = tpt.expects_headers ? pd.slice(sol, sol+8) : tpt.header;
        let hs = header_str(hb);
        let pp = header_pool.get(hs);
        let pp_write = function(sl, el, dl)
        {
            // "this" is the PP socket
            let pp = sock_pool.get(this);
            // strip the header (if required) and write to the PP
            let buf = dl.slice(sl+skip, el+1);
            try {
                // first capture/rewrite any *outgoing* we're concerned with
                buf = pp.processLine(buf);
                this.write(buf);
            } catch (x) {}
        };
        if (pp == null)
        {
            logger.info('Creating a proxied pool connection for: %s', hs);
            let sock = net.createConnection({port:pool_port, host:pool_ip},
            (function()
            {
                let sl = sol;
                let el = eol;
                let sh = hs;
                // as the initial send is deferred, we need to copy the slice
                let dl = Buffer.from(pd.slice(sl,el+1));
                return function()
                {
                    // "this" is the PP socket
                    header_pool.set(sh, sock_pool.get(this));
                    pp_write.call(this, 0, dl.length-1, dl);
                };
            })());
            sock.setNoDelay(true);
            sock.setTimeout(stale_timeout, sock.destroy);
            sock.on('data', pool_data);
            sock.on('end', pool_end);
            sock.once('close', pool_close);
            sock.on('error', pool_error);
            pp = new PoolSession(sock);
            pp.pool_user = pool_user;
            pp.pool_id = pool_id;
            pp.trunk = tpt;
            pp.header = hb;
            sock_pool.set(sock, pp);
            logger.info('Proxy id: %d, pid: %d, has %d pool sockets', proxy_id, process.pid, sock_pool.size);
        }
        else
        {
            pp_write.call(pp.sock, sol, eol, pd);
        }
        sol = eol + 1;
    }
    while (sol < pd.length);
}

function miner_data(data)
{
    let ra = this.remoteAddress + ':' + this.remotePort;
    let pm = clients.get(ra);
    let pd = pm.process(data);
    // For each line, write the header and line
    if (pd.length == 0)
        return;
    let sol = 0;
    let eol = 0;
    do
    {
        eol = pd.indexOf('\n', sol);
        try {
            pm.trunk.sock.write(pm.header);
            pm.trunk.sock.write(pd.slice(sol, eol+1));
        } catch (e) {}
        sol = eol + 1;
    } while (sol < pd.length);
}

// Read a PM or TPT
function client_data(data)
{
    if (!isVal)
        miner_data.call(this, data);
    else
        tpt_data.call(this, data);
}

function client_close()
{
    let ra = this.remoteAddress + ':' + this.remotePort;
    logger.info('%s close: %s', !isVal ? 'PM' : 'TPT',  ra);
    clients.delete(ra);
}

function client_error(err)
{
    let ra = this.remoteAddress + ':' + this.remotePort;
    logger.info('%s %s with error: %s', !isVal ? 'PM' : 'TPT', ra, err.message);
    clients.delete(ra);
}

// Read LPT and write to a PM
function upstream_data(data)
{
    if (upstream == null)
        return;
    let pd = upstream.process(data);
    if (pd.length == 0)
        return;
    // Each line has a header which identifies the PM to send to...
    let sol = 0;
    let eol = 0;
    do
    {
        eol = pd.indexOf('\n', sol+8);
        let hb = pd.slice(sol, sol+8);
        let hs = header_str(hb);
        let pm = clients.get(hs);
        try {
            // Exclude the header
            pm.sock.write(pd.slice(sol+8, eol+1));
        } catch (e) {}
        sol = eol + 1;
    } while (sol < pd.length);
}

function upstream_end()
{
    logger.info('LPT end');
    upstream = null;
}

function upstream_close()
{
    logger.info('LPT close');
    upstream = null;
}

function upstream_error()
{
    logger.info('LPT error');
}

// Read a PP and write to TPT
function pool_data(data)
{
    let pp = sock_pool.get(this);
    if (pp == null)
        return;
    let tpt = pp.trunk;
    if (tpt == null)
        return;
    let pd = pp.process(data);
    // For each line, write the header (if required) and write line
    if (pd.length == 0)
        return;
    let sol = 0;
    let eol = 0;
    do
    {
        eol = pd.indexOf('\n', sol);
        try {
            if (tpt.expects_headers)
                tpt.sock.write(pp.header);
            tpt.sock.write(pd.slice(sol, eol+1));
        } catch (e) {}
        sol = eol + 1;
    } while (sol < pd.length);
}

function pool_end()
{
    let pp = sock_pool.get(this);
    let hs = header_str(pp.header);
    header_pool.delete(hs);
    sock_pool.delete(this);
    logger.info('PP for %s end', hs);
}

function pool_close()
{
    let pp = sock_pool.get(this);
    if (pp == null)
        return;
    let hs = header_str(pp.header);
    header_pool.delete(hs);
    sock_pool.delete(this);
    logger.info('PP for %s close', hs);
}

function pool_error(err)
{
    let pp = sock_pool.get(this);
    let hs = header_str(pp.header);
    header_pool.delete(hs);
    sock_pool.delete(this);
    logger.info('PP for %s with error: %s', hs, err.message);
}

// vim: sw=4 ts=4 et

//-------------------------

function block_listener(){
/*
    var subscription = web3.eth.subscribe('newBlockHeaders', function(error, block){
        if (error) {
            logger.info(error);
        } else {

            
            //var contract = "0x2d9a998fa591ef40563dc56bac835d03680f8d23";
            //var cmd = "0x70a08231";
            //var results = await ethCall("", contract, cmd);

            //pool_ip = results[0].Host;
            //pool_port = results[0].Port;
            //pool_user = results[0].Username;
            
        }    
         
    }); */ 
}

pragma solidity ^0.4.18;

// CONTRACT PROPERTY OF TITAN.io 
// Description: This is a TEST contract and not to be used for real world use. 
//
// This contract permits updates of a single: Hostname, Port, Username, Password
// This contract also permits a full URI (Ref. as URL)

contract ProxyRouter {
    
    string public URL; // Full URL
    
    // Permit for storing individual changes or above for full incoming website POST
    string public Hostname;
    string public Port;
    string public Username;
    string public Password;
    
    function incomingURL(string theURL) public {
        URL = theURL;
    }
    
    function incomingHostname(string theHostname) public {
        Hostname = theHostname;
    }
    
    function incomingPort(string thePort) public {
        Port = thePort;
    }

    function incomingUsername(string theUsername) public {
        Username = theUsername;
    }
    
    function incomingPassword(string thePassword) public {
        Password = thePassword;
    }
    
    function getURL() public view returns(string) {
        return URL;
    }
    
}

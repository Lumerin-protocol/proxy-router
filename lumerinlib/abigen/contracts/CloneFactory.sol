//SPDX-License-Identifier: MIT

pragma solidity >0.8.0;

import "./Clones.sol";
import "./Implementation.sol";
import "./LumerinToken.sol";

/// @title CloneFactory
/// @author Josh Kean (Lumerin)
/// @notice Variables passed into contract initializer are subject to change based on the design of the hashrate contract

//CloneFactory now responsible for minting, purchasing, and tracking contracts
contract CloneFactory {
    address baseImplementation;
    address validator;
    address lmnDeploy; //deployed address of lumerin token
    address titanFund; //fund where lumerin tokens are sent for titan transaction
    address[] public rentalContracts; //dynamically allocated list of rental contracts
    bool titanCut; //bool to turn on and off the cut of funds, in for testing, should always be true in production
    Lumerin lumerin;
    TempMarketPlace marketplace;

    constructor(address _lmn, address _validator, address _titanFund) {
        Implementation _imp = new Implementation();
        marketplace = new TempMarketPlace(); //deploying the marketplace as part of this contract so the address doesn't have to be provided manually
        baseImplementation = address(_imp);
        lmnDeploy = _lmn; //deployed address of lumeirn token
        validator = _validator;
        titanFund = _titanFund;
        lumerin = Lumerin(_lmn);
        titanCut = false;
    }

    event contractCreated(address indexed _address, string _pubkey); //emitted whenever a contract is created
    event clonefactoryContractPurchased(address indexed _address); //emitted whenever a contract is purchased

    //function to create a new Implementation contract
    function setCreateNewRentalContract(
        uint256 _price,
        uint256 _limit,
        uint256 _speed,
        uint256 _length,
        address _validator,
        string memory _pubKey
    ) external returns (address) {
        address _newContract = Clones.clone(baseImplementation);
        Implementation(_newContract).initialize(
            _price,
            _limit,
            _speed,
            _length,
            msg.sender,
            lmnDeploy,
            address(this),
            _validator
        );
        rentalContracts.push(_newContract); //add clone to list of contracts
        emit contractCreated(_newContract, _pubKey); //broadcasts a new contract and the pubkey to use for encryption
        return _newContract;
    }

    function setTitanCut(bool _cut) public {
        titanCut = _cut;
    }

    //function to purchase a hashrate contract
    //requires the clonefactory to be able to spend tokens on behalf of the purchaser
    function setPurchaseRentalContract(
        address contractAddress,
        string memory _cipherText
    ) external {
        Implementation targetContract = Implementation(contractAddress);
        uint256 _price = targetContract.price();
        require(
            lumerin.allowance(msg.sender, address(this)) >= _price,
            "not authorized to spend required funds"
        );
        if (titanCut) {
            uint num;
            uint den;
            (num, den) = marketplace.getTitanPercentage();
            uint256 titanPoolCut = _price*num/den;
            uint256 contractCut  = _price-titanPoolCut;
            bool titanTransfer = lumerin.transferFrom(
                msg.sender,
                titanFund,
                titanPoolCut
            );
            require(titanTransfer, "lumeirn tranfer failed");
            bool contractTransfer = lumerin.transferFrom(
                msg.sender,
                contractAddress,
                contractCut
            );
            require(contractTransfer, "lumeirn tranfer failed");
        } else {
            bool tokensTransfered = lumerin.transferFrom(
                msg.sender,
                contractAddress,
                _price
            );
            require(tokensTransfered, "lumeirn tranfer failed");
        }
        targetContract.setPurchaseContract(_cipherText, msg.sender);
        emit clonefactoryContractPurchased(contractAddress);
    }

    function getContractList() external view returns (address[] memory) {
        address[] memory _rentalContracts = rentalContracts;
        return _rentalContracts;
    }
}


/*
the contract TempMarketPlace is to provide an external source of the percentage
of each transaction titan will take from each transaction.
The close factory will call into the fully developed and audited 
version of the market place contract at a later time
*/
contract TempMarketPlace {
    uint256 numerator;
    uint256 denominator;

    constructor() {
        numerator = 25;
        denominator = 1000;
    }

    function getTitanPercentage() public view returns(uint256, uint256) {
        return(numerator, denominator);
    }
}





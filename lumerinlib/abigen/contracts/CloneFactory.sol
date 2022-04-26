//SPDX-License-Identifier: MIT

pragma solidity >0.8.0;

import "./Clones.sol";
import './Implementation.sol';
import "./LumerinToken.sol";

/// @title CloneFactory
/// @author Josh Kean (Lumerin)
/// @notice Variables passed into contract initializer are subject to change based on the design of the hashrate contract

//CloneFactory now responsible for minting, purchasing, and tracking contracts
contract CloneFactory {
    address baseImplementation;
    address validator;
    address lmnDeploy;
    address webfacingAddress;
    address[] public rentalContracts; //dynamically allocated list of rental contracts
    Lumerin lumerin;

    constructor(address _lmn, address _validator) {
        Implementation _imp = new Implementation();
        baseImplementation = address(_imp);
        lmnDeploy = _lmn; //deployed address of lumeirn token
        validator = _validator;
        lumerin = Lumerin(_lmn);
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
        bool tokensTransfered = lumerin.transferFrom(
            msg.sender,
            contractAddress,
            _price
        );
        require(tokensTransfered, "lumeirn tranfer failed");
        targetContract.setPurchaseContract(_cipherText, msg.sender);
        emit clonefactoryContractPurchased(contractAddress);
    }

    function getContractList() external view returns (address[] memory) {
        address[] memory _rentalContracts = rentalContracts;
        return _rentalContracts;
    }
}




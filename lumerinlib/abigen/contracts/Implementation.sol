//SPDX-License-Identifier: UNLICENSED

pragma solidity >0.8.0;

import "./Initializable.sol";
import "./Escrow.sol";

//MyToken is place holder for actual lumerin token, purely for testing purposes
contract Implementation is Initializable, Escrow {
    enum ContractState {
        Available,
        Running
    }

    ContractState public contractState;
    uint256 public price; //cost to purchase contract
    uint256 public limit; //variable used to aid in the lumerin nodes decision making
    uint256 public speed; //th/s of contract
    uint256 public length; //how long the contract will last in seconds
    uint256 public startingBlockTimestamp; //the timestamp of the block when the contract was purchased
    address public buyer; //address of the current purchaser of the contract
    address public seller; //address of the seller of the contract
    address cloneFactory; //used to limit where the purchase can be made
    address validator; //validator to be used. Can be set to 0 address if validator not being used
    string public encryptedPoolData; //encrypted data for pool target info

    event contractPurchased(address indexed _buyer); //make indexed
    event contractClosed();
    event purchaseInfoUpdated();
    event cipherTextUpdated(string newCipherText);

    function initialize(
        uint256 _price,
        uint256 _limit,
        uint256 _speed,
        uint256 _length,
        address _seller,
        address _lmn,
        address _cloneFactory, //used to restrict purchasing power to only the clonefactory
        address _validator
    ) public initializer {
        price = _price;
        limit = _limit;
        speed = _speed;
        length = _length;
        seller = _seller;
        cloneFactory = _cloneFactory;
        validator = _validator;
        contractState = ContractState.Available;
        setParameters(_lmn);
    }

    function getPublicVariables()
        public
        view
        returns (
            ContractState,
            uint256,
            uint256,
            uint256,
            uint256,
            uint256,
            address,
            address,
            string memory
        )
    {
        return (
            contractState,
            price,
            limit,
            speed,
            length,
            startingBlockTimestamp,
            buyer,
            seller,
            encryptedPoolData
        );
    }

    //function that the clone factory calls to purchase the contract
    function setPurchaseContract(
        string memory _encryptedPoolData,
        address _buyer
    ) public {
        require(
            contractState == ContractState.Available,
            "contract is not in an available state"
        );
        require(
            msg.sender == cloneFactory,
            "this address is not approved to call the purchase function"
        );
        encryptedPoolData = _encryptedPoolData;
        buyer = _buyer;
        startingBlockTimestamp = block.timestamp;
        contractState = ContractState.Running;
        createEscrow(seller, buyer, price);
        emit contractPurchased(msg.sender);
    }

    //allows the buyers to update their mining pool information
    //during the lifecycle of the contract
    function setUpdateMiningInformation(string memory _newEncryptedPoolData)
        external
    {
        require(
            msg.sender == buyer,
            "this account is not authorized to update the ciphertext information"
        );
        require(
            contractState == ContractState.Running,
            "the contract is not in the running state"
        );
        encryptedPoolData = _newEncryptedPoolData;
        emit cipherTextUpdated(_newEncryptedPoolData);
    }

    //function which can edit the cost, length, and hashrate of a given contract
    function setUpdatePurchaseInformation(
        uint256 _price,
        uint256 _limit,
        uint256 _speed,
        uint256 _length,
        uint256 _closeoutType
    ) external {
        uint256 durationOfContract = block.timestamp - startingBlockTimestamp;
        require(
            msg.sender == seller,
            "this is account is not authorized to update the contract parameters"
        );
        require(
            contractState == ContractState.Running,
            "this is account is not in the running state"
        );
        require(
            _closeoutType == 2 || _closeoutType == 3,
            "you can only use closeout options 2 or 3"
        );
        require(
            durationOfContract >= length,
            "the contract has yet to be carried to term"
        );
        setContractCloseOut(_closeoutType);
        price = _price;
        limit = _limit;
        speed = _speed;
        length = _length;
        emit purchaseInfoUpdated();
    }

    function setContractVariableUpdate() internal {
        buyer = seller;
        encryptedPoolData = "";
        contractState = ContractState.Available;
        startingBlockTimestamp = 0;
    }

    //purpose is to calculate how many tokens the buyer is owed incase the contract
    //is canceled early
    function buyerPayoutCalc() internal view returns (uint256) {
        uint256 durationOfContract = block.timestamp - startingBlockTimestamp;
        if (durationOfContract <= length) {
            return
                uint256(price * uint256(length - durationOfContract)) /
                uint256(length);
        }
        return 0;
    }

    function setContractCloseOut(uint256 closeOutType) public {
        if (closeOutType == 0) {
            //this is a function call to be triggered by the buyer or validator
            //in the event that a contract needs to be canceled early for any reason
            require(
                msg.sender == buyer || msg.sender == validator,
                "this account is not authorized to trigger an early closeout"
            );
            uint256 buyerPayout = buyerPayoutCalc();
            if (buyerPayout > 0) {
                withdrawFunds(myToken.balanceOf(address(this))-buyerPayout, buyerPayout);
            }
            setContractVariableUpdate();
            emit contractClosed();
        } else if (closeOutType == 1) {
            //this is a function call for the seller to withdraw their funds
            //at any time during the smart contracts lifecycle
            require(
                msg.sender == seller,
                "this account is not authorized to trigger a mid-contract closeout"
            );
            getDepositContractHodlingsToSeller(buyerPayoutCalc());
        } else if (closeOutType == 2 || closeOutType == 3) {
            require(
                block.timestamp - startingBlockTimestamp >= length,
                "the contract has yet to be carried to term"
            );
            if (closeOutType == 3) {
                withdrawFunds(myToken.balanceOf(address(this)), 0);
            }
            setContractVariableUpdate();
            emit contractClosed();
        }
    }
}











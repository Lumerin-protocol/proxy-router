// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title Marketplace Escrow
/// @author Lance Seidman (Lumerin)
/// @notice This first version will be used to hold lumerin temporarily for the Marketplace Hash Rental.

import "./ReentrancyGuard.sol";
import "./LumerinToken.sol";

contract Escrow is ReentrancyGuard {
    address public escrow_purchaser; // Entity making a payment...
    address public escrow_seller; // Entity to receive funds...
    uint256 public contractTotal; // How much should be escrowed...
    uint256 public receivedTotal; // Optional; Keep a balance for how much has been received...
    Lumerin myToken;

    //internal function which will be called by the hashrate contract
    function setParameters(address _titanToken) internal {
        myToken = Lumerin(_titanToken);
    }

    //internal function which transfers current hodled tokens into sellers account
    function getDepositContractHodlingsToSeller(uint256 remaining) internal {
        myToken.transfer(
            escrow_seller,
            myToken.balanceOf(address(this)) - remaining
        );
    }

    // @notice This will create a new escrow based on the seller, buyer, and total.
    // @dev Call this in order to make a new contract.
    function createEscrow(
        address _escrow_seller,
        address _escrow_purchaser,
        uint256 _lumerinTotal
    ) internal {
        escrow_seller = _escrow_seller;
        escrow_purchaser = _escrow_purchaser;
        contractTotal = _lumerinTotal;
    }

    // @notice Validator can request the funds to be released once determined it's safe to do.
    // @dev Function makes sure the contract was fully funded
    // by checking the State and if so, release the funds to the seller.
    // sends lumerin tokens to the appropriate entities.
    // _buyer will obtain a 0 value unless theres a penalty involved
    function withdrawFunds(uint256 _seller, uint256 _buyer)
        internal
        nonReentrant
    {
        myToken.transfer(escrow_seller, _seller);
        if (_buyer != 0) {
            myToken.transfer(escrow_purchaser, _buyer);
        }
    }
}


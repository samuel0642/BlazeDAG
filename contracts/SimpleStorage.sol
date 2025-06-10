// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/**
 * @title SimpleStorage
 * @dev A simple smart contract to demonstrate EVM compatibility on BlazeDAG
 */
contract SimpleStorage {
    uint256 private storedValue;
    address public owner;
    
    event ValueChanged(uint256 newValue, address changedBy);
    
    constructor() {
        owner = msg.sender;
        storedValue = 0;
    }
    
    /**
     * @dev Store a value
     * @param _value The value to store
     */
    function setValue(uint256 _value) public {
        storedValue = _value;
        emit ValueChanged(_value, msg.sender);
    }
    
    /**
     * @dev Retrieve the stored value
     * @return The stored value
     */
    function getValue() public view returns (uint256) {
        return storedValue;
    }
    
    /**
     * @dev Get the owner of the contract
     * @return The owner address
     */
    function getOwner() public view returns (address) {
        return owner;
    }
    
    /**
     * @dev Increment the stored value by 1
     */
    function increment() public {
        storedValue += 1;
        emit ValueChanged(storedValue, msg.sender);
    }
    
    /**
     * @dev Decrement the stored value by 1
     */
    function decrement() public {
        require(storedValue > 0, "Value cannot be negative");
        storedValue -= 1;
        emit ValueChanged(storedValue, msg.sender);
    }
} 
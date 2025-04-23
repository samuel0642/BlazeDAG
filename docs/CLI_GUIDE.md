# BlazeDAG CLI Guide

## Table of Contents
1. [Installation](#installation)
2. [Starting the Node](#starting-the-node)
3. [Running the Chain](#running-the-chain)
4. [CLI Commands](#cli-commands)
5. [Account Management](#account-management)
6. [Transaction Management](#transaction-management)
7. [Consensus and Wave-based Protocol](#consensus-and-wave-based-protocol)
8. [Block Operations](#block-operations)
9. [Network Operations](#network-operations)
10. [Running Multiple Validators](#running-multiple-validators)
11. [Troubleshooting](#troubleshooting)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/CrossDAG/BlazeDAG.git
cd BlazeDAG
```

2. Build the project:
```bash
make build
```

## Starting the Node

### Basic Node
```bash
./bin/blazedag
```

### Validator Node
```bash
./bin/blazedag --validator --node-id <node_id> --port <port>
```

Example:
```bash
./bin/blazedag --validator --node-id validator1 --port 3000
```

Node ID Requirements:
- Must be unique in the network
- Used to identify the validator
- Should be descriptive (e.g., validator1, validator2)
- Cannot contain special characters

### Custom Port
```bash
./bin/blazedag --port 3000
```

### Combined Options
```bash
./bin/blazedag --validator --node-id <node_id> --port <port>
```

## Running the Chain

### Starting a Single Node
1. Start a basic node:
```bash
./bin/blazedag
```

2. Create an account:
```bash
newaccount
```

3. Check the account:
```bash
accounts
```

4. Send a transaction:
```bash
send <from_address> <to_address> <amount>
```

5. Monitor blocks:
```bash
blocks
```

### Starting a Validator Node
1. Start a validator node with a unique ID:
```bash
./bin/blazedag --validator --node-id validator1 --port 3000
```

2. Check validator status:
```bash
status
```

3. Monitor consensus:
```bash
blocks
```

### Running Multiple Validators
1. Start first validator:
```bash
./bin/blazedag --validator --node-id validator1 --port 3000
```

2. Start second validator:
```bash
./bin/blazedag --validator --node-id validator2 --port 3001
```

3. Start third validator:
```bash
./bin/blazedag --validator --node-id validator3 --port 3002
```

4. Check network status:
```bash
peers
```

### Chain Operations

#### Creating Transactions
1. Create accounts:
```bash
newaccount
newaccount
```

2. Send transaction:
```bash
send <from_address> <to_address> <amount>
```

3. Verify transaction:
```bash
blocks
```

#### Block Creation (Validators Only)
1. Check validator status:
```bash
status
```

2. Create block:
```bash
createblock
```

3. Monitor block:
```bash
blocks
```

#### Consensus Monitoring
1. Check current wave:
```bash
status
```

2. View block details:
```bash
block <hash>
```

3. Monitor network:
```bash
peers
```

### Chain Management

#### State Management
- State is automatically committed after each block
- State changes are tracked in the DAG
- State root is updated with each block

#### Wave Management
- Waves progress automatically
- Each wave has a timeout
- Waves are numbered sequentially

#### Round Management
- Rounds are part of each wave
- Multiple blocks per round
- Round duration is configurable

#### Block Management
- Blocks are created by validators
- Blocks reference previous blocks
- Blocks contain transactions

#### Transaction Management
- Transactions are added to mempool
- Transactions are included in blocks
- Transactions are executed in order

### Best Practices

1. **Node Operation**
   - Keep node running continuously
   - Monitor system resources
   - Check logs regularly

2. **Transaction Management**
   - Verify amounts before sending
   - Keep track of transaction hashes
   - Monitor transaction status

3. **Validator Operations**
   - Maintain stable network connection
   - Monitor consensus status
   - Regular status checks

4. **Network Management**
   - Use unique ports for validators
   - Monitor peer connections
   - Regular network health checks

## CLI Commands

### Help
```bash
help
```
Shows all available commands and their usage.

### Status
```bash
status
```
Displays:
- Node ID
- Connected Peers
- Current Wave
- Validator Status
- Total Accounts

### Peers
```bash
peers
```
Lists all connected peers in the network.

### Blocks
```bash
blocks
```
Shows recent blocks with:
- Block Hash
- Height
- Wave
- Timestamp
- Transaction Count
- Consensus Status
- Confirmations

### Block Details
```bash
block <hash>
```
Shows detailed information about a specific block:
- Block Hash
- Height
- Wave
- Timestamp
- Consensus Status
- Confirmations
- Transaction Details

## Account Management

### Create New Account
```bash
newaccount
```
Creates a new account with:
- Random address
- Initial balance of 1,000,000 units
- Nonce starting at 0

### List Accounts
```bash
accounts
```
Shows all accounts with:
- Address
- Balance
- Nonce

### Check Balance
```bash
balance <address>
```
Displays the balance of a specific account.

## Transaction Management

### Send Transaction
```bash
send <from> <to> <amount>
```
Example:
```bash
send c97be34ac22487ea4bc4268aec50b4187adc3c9c 87d5a6209a138d057df870e95fd951cb44be38b0 1000
```

Transaction Parameters:
- `from`: Sender's address
- `to`: Recipient's address
- `amount`: Amount to send

Transaction Properties:
- Gas Price: 1
- Gas Limit: 21000
- Automatic nonce management

## Consensus and Wave-based Protocol

### Wave-based Consensus
BlazeDAG uses a wave-based consensus model where:
- Each wave represents a consensus round
- Waves are numbered sequentially (0, 1, 2, ...)
- Each wave has multiple rounds within it
- Consensus is achieved through wave completion

### Round Management
- Rounds are the basic time units in the protocol
- Each round has a fixed duration (configurable)
- Rounds are numbered within each wave
- Multiple blocks can be created in a single round

### Wave Structure
```
Wave N
├── Round 0
│   ├── Block A
│   └── Block B
├── Round 1
│   ├── Block C
│   └── Block D
└── Round 2
    ├── Block E
    └── Block F
```

### Consensus Process
1. **Wave Start**
   - New wave begins
   - Validators prepare for consensus
   - Round counter resets

2. **Round Execution**
   - Validators create blocks
   - Transactions are processed
   - References are established

3. **Wave Completion**
   - Consensus is reached
   - State is finalized
   - Next wave begins

### Wave Commands
```bash
# Check current wave
status

# View wave details
blocks
```

### Wave Properties
- **Duration**: Configurable wave timeout
- **Validators**: Minimum required validators
- **Quorum**: Required for consensus
- **References**: Block reference rules

### Block References
- Blocks must reference previous blocks
- References establish DAG structure
- Multiple references allowed
- Reference types:
  - Standard references
  - Wave references
  - Round references

## Block Operations

### Create Block (Validators Only)
```bash
createblock
```
Creates a new block containing pending transactions from the mempool.

Block Properties:
- Version: 1
- Round: Current wave
- Wave: Current wave
- Height: Current height + 1
- Parent Hash: Latest block hash
- Timestamp: Current time

## Network Operations

### Node Status
```bash
status
```
Shows:
- Node ID
- Connected Peers
- Current Wave
- Validator Status
- Total Accounts

### Peer Information
```bash
peers
```
Lists all connected peers with their IDs.

## Running Multiple Validators

1. Start First Validator:
```bash
./bin/blazedag --validator --node-id validator1 --port 3000
```

2. Start Additional Validators:
```bash
./bin/blazedag --validator --node-id validator2 --port 3001
./bin/blazedag --validator --node-id validator3 --port 3002
```

Validator Properties:
- Automatic peer discovery
- Consensus participation
- Block creation rights
- Transaction validation

## Troubleshooting

### Common Issues

1. **Failed to Start Node**
   - Check if port is available
   - Verify validator status
   - Check system resources

2. **Transaction Failures**
   - Verify account balances
   - Check nonce values
   - Ensure valid addresses

3. **Block Creation Issues**
   - Verify validator status
   - Check mempool for transactions
   - Monitor consensus status

4. **Network Issues**
   - Check peer connections
   - Verify network configuration
   - Monitor node status

### Logging

Logs are stored in `chaindata/chain.log` and include:
- Node startup/shutdown
- Transaction processing
- Block creation
- Network events
- Error messages

### Best Practices

1. **Account Management**
   - Keep track of account addresses
   - Monitor balances regularly
   - Use descriptive labels for accounts

2. **Transaction Management**
   - Verify amounts before sending
   - Keep track of transaction hashes
   - Monitor transaction status

3. **Validator Operations**
   - Maintain stable network connection
   - Monitor system resources
   - Regular status checks

4. **Network Management**
   - Use unique ports for validators
   - Monitor peer connections
   - Regular network health checks 
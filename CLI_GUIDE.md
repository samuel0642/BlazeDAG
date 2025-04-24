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
11. [Monitoring and Logging](#monitoring-and-logging)
12. [Troubleshooting](#troubleshooting)

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

### Configuration
BlazeDAG uses a YAML configuration file (`config.yaml`) to configure node behavior. The configuration includes:
- Node ID
- Consensus settings
- Network settings
- State settings
- EVM settings

### Basic Node
1. Create a basic config.yaml:
```yaml
node_id: "node1"
consensus:
  wave_timeout: 10s
  round_duration: 5s
  validator_set:
    - "node1"
  quorum_size: 1
  listen_addr: "localhost:3000"
  seeds: []
  total_validators: 1
```

2. Start the node:
```bash
./bin/blazedag
```

### Validator Node
1. Create a validator config.yaml:
```yaml
node_id: "validator1"
consensus:
  wave_timeout: 10s
  round_duration: 5s
  validator_set:
    - "validator1"
    - "validator2"
    - "validator3"
  quorum_size: 2
  listen_addr: "localhost:3000"
  seeds:
    - "localhost:3001"
    - "localhost:3002"
  total_validators: 3
```

2. Start the validator:
```bash
./bin/blazedag
```

### Custom Configuration
You can specify a different config file:
```bash
./bin/blazedag -config custom_config.yaml
```

### Running Multiple Validators
1. Create separate config files for each validator:

config.validator1.yaml:
```yaml
node_id: "validator1"
consensus:
  wave_timeout: 10s
  round_duration: 5s
  validator_set:
    - "validator1"
    - "validator2"
    - "validator3"
  quorum_size: 2
  listen_addr: "localhost:3000"
  seeds:
    - "localhost:3001"
    - "localhost:3002"
  total_validators: 3
```

config.validator2.yaml:
```yaml
node_id: "validator2"
consensus:
  wave_timeout: 10s
  round_duration: 5s
  validator_set:
    - "validator1"
    - "validator2"
    - "validator3"
  quorum_size: 2
  listen_addr: "localhost:3001"
  seeds:
    - "localhost:3000"
    - "localhost:3002"
  total_validators: 3
```

config.validator3.yaml:
```yaml
node_id: "validator3"
consensus:
  wave_timeout: 10s
  round_duration: 5s
  validator_set:
    - "validator1"
    - "validator2"
    - "validator3"
  quorum_size: 2
  listen_addr: "localhost:3002"
  seeds:
    - "localhost:3000"
    - "localhost:3001"
  total_validators: 3
```

2. Start each validator with its config:
```bash
# Terminal 1
./bin/blazedag -config config.validator1.yaml

# Terminal 2
./bin/blazedag -config config.validator2.yaml

# Terminal 3
./bin/blazedag -config config.validator3.yaml
```

### Configuration Parameters

#### Node Configuration
- `node_id`: Unique identifier for the node
- `consensus`: Consensus-related settings
  - `wave_timeout`: Maximum duration for a wave
  - `round_duration`: Duration of each round
  - `validator_set`: List of validator addresses
  - `quorum_size`: Minimum votes required for consensus
  - `listen_addr`: Address to listen for incoming connections
  - `seeds`: List of seed node addresses
  - `total_validators`: Total number of validators

#### State Configuration
- `state`: State-related settings
  - `genesis_file`: Path to genesis file

#### EVM Configuration
- `evm`: EVM-related settings
  - `chain_id`: Chain identifier
  - `gas_limit`: Maximum gas per block

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

## Monitoring and Logging

### Starting with Logging

1. Start the node with logging enabled:
```bash
./bin/blazedag 2>&1 | tee chain.log
```

This will:
- Run the node
- Show output in terminal
- Save all output to chain.log

### Monitoring Wave and Round Progression

1. Check current wave and round:
```bash
status
```
This shows:
- Current wave number
- Current round number
- Wave status (Proposing, Voting, Committing, Finalized)
- Leader status

2. Monitor wave progression:
```bash
tail -f chain.log | grep "wave"
```
This will show:
- Wave start events
- Wave completion events
- Wave timeout events
- Wave status changes

3. Monitor round progression:
```bash
tail -f chain.log | grep "round"
```
This will show:
- Round start events
- Round completion events
- Round timeout events
- Round status changes

### Log File Structure

The log file contains detailed information about:
- Wave transitions
- Round transitions
- Block creation
- Consensus events
- Network events
- Error messages

Example log entries:
```
[INFO] Starting new wave: 1
[INFO] Round 0 started in wave 1
[INFO] Block created in round 0, wave 1
[INFO] Round 0 completed in wave 1
[INFO] Round 1 started in wave 1
[INFO] Wave 1 completed
[INFO] Starting new wave: 2
```

### Real-time Monitoring

1. Monitor all events:
```bash
tail -f chain.log
```

2. Monitor specific events:
```bash
# Monitor wave events
tail -f chain.log | grep "wave"

# Monitor round events
tail -f chain.log | grep "round"

# Monitor block events
tail -f chain.log | grep "block"

# Monitor consensus events
tail -f chain.log | grep "consensus"
```

### Log Analysis

1. Count waves:
```bash
grep "Starting new wave" chain.log | wc -l
```

2. Count rounds per wave:
```bash
grep "Round.*started in wave" chain.log | awk '{print $NF}' | sort | uniq -c
```

3. Count blocks per round:
```bash
grep "Block created in round" chain.log | awk '{print $NF}' | sort | uniq -c
```

### Best Practices for Monitoring

1. **Regular Checks**
   - Monitor wave progression
   - Check round completion
   - Verify block creation
   - Monitor consensus status

2. **Log Management**
   - Rotate logs regularly
   - Archive important logs
   - Monitor log size
   - Set up log alerts

3. **Performance Monitoring**
   - Track wave duration
   - Monitor round completion time
   - Check block creation rate
   - Monitor consensus speed

4. **Error Monitoring**
   - Watch for wave timeouts
   - Monitor round failures
   - Check consensus errors
   - Track network issues

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
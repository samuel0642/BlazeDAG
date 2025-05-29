# BlazeDAG Sync - DAG Synchronization Between Validators

This implementation focuses **only on DAG synchronization** between validators, without consensus waves.

## 🎯 What It Does

- **Creates blocks at each round** that reference previous round blocks
- **Broadcasts blocks** to other validators over TCP network
- **Receives blocks** from other validators and adds to local DAG
- **Forms proper DAG structure** with round-based progression
- **Syncs between multiple validators** in real-time

## 🏗 Architecture

```
Round 1:  [Block A1] [Block B1] [Block C1]
             |          |          |
Round 2:  [Block A2] [Block B2] [Block C2]
             |          |          |
Round 3:  [Block A3] [Block B3] [Block C3]
```

Each block references blocks from the previous round, forming a DAG.

## 🚀 Quick Start

### Build
```bash
go build -o dagsync ./cmd/dagsync/simple_main.go
```

### Run Single Validator
```bash
./dagsync -id="validator1" -listen="localhost:4001"
```

### Run Multiple Validators
```bash
# Terminal 1
./dagsync -id="validator1" -listen="localhost:4001" -peers="localhost:4002,localhost:4003"

# Terminal 2  
./dagsync -id="validator2" -listen="localhost:4002" -peers="localhost:4001,localhost:4003"

# Terminal 3
./dagsync -id="validator3" -listen="localhost:4003" -peers="localhost:4001,localhost:4002"
```

### Run Demo
```bash
./scripts/demo_dagsync.sh
```

## 📊 What You'll See

```
2025/05/29 16:24:30 DAG Sync [validator1]: Starting DAG synchronization
2025/05/29 16:24:30 DAG Sync [validator1]: Started at localhost:4001, connecting to 0 peers
2025/05/29 16:24:32 DAG Sync [validator1]: Advanced to round 1
2025/05/29 16:24:32 Added block to DAG - Hash: abc123..., Height: 1, Validator: validator1, References: 0
2025/05/29 16:24:32 DAG Sync [validator1]: Created block for round 1 with 0 references
2025/05/29 16:24:32 DAG Sync [validator1]: Broadcasted block (round 1) to 0 peers
2025/05/29 16:24:34 DAG Sync [validator1]: Advanced to round 2
2025/05/29 16:24:34 Added block to DAG - Hash: def456..., Height: 2, Validator: validator1, References: 1
```

## 🔧 Command Line Options

- `-id`: Validator ID (default: "validator1")
- `-listen`: Listen address (default: "localhost:3001") 
- `-peers`: Comma-separated peer addresses (default: "")
- `-round-duration`: Duration between rounds (default: 2s)

## 🎛 Key Features

### ✅ **DAG Round Progression**
- Each validator advances rounds every 2 seconds
- Creates one block per round
- References blocks from previous round

### ✅ **Network Synchronization**
- TCP connections between validators
- JSON message broadcasting
- Automatic peer reconnection

### ✅ **Block Structure**
- Each block contains transactions
- References to previous round blocks
- Proper DAG formation

### ✅ **Real-time Monitoring**
- Live logs showing round progression
- Block creation and broadcasting
- Network connection status

## 🔍 Architecture Details

### Block Creation Process
1. **Round Timer**: Every 2 seconds, advance to next round
2. **Reference Collection**: Get blocks from previous round
3. **Block Creation**: Create new block with references
4. **Local Storage**: Add block to local DAG
5. **Network Broadcast**: Send block to all connected peers

### Network Protocol
- **Transport**: TCP with JSON messages
- **Message Types**: Block broadcasts
- **Connection Management**: Auto-reconnect to peers
- **Data Format**: Structured JSON with block data

### DAG Structure
- **Rounds**: Sequential progression (1, 2, 3, ...)
- **References**: Each block references previous round blocks
- **Height**: DAG height increases with new blocks
- **Validators**: Each validator contributes blocks to the shared DAG

## 🎪 Demo Scenarios

### Single Validator
Shows basic block creation and round progression without networking.

### Multiple Validators
Shows proper DAG synchronization with:
- Network connectivity
- Block broadcasting
- Reference formation
- Distributed DAG growth

## 🔮 Next Steps

After this DAG sync is working properly, the next step would be to add:
- **Consensus waves** that operate independently
- **Leader selection** for wave proposals  
- **Block finalization** through consensus
- **Transaction ordering** via consensus commits

But for now, this focuses purely on **DAG transport layer synchronization**.

---

**Status**: ✅ Working - DAG sync between validators operational 
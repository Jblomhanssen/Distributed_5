# Distributed Auction System

A fault-tolerant distributed auction system using primary-backup replication.

## Building

```bash
# Generate protobuf code
make proto

# Build all components
go build -o bin/primary ./primary
go build -o bin/backup ./backup
go build -o bin/client ./client
```

## Running the System

**Note:** Port 5000 is often used by macOS ControlCenter, so we use 5001 and 5002.

### Terminal 1: Start Backup
```bash
go run backup/*.go -port 5002
```

### Terminal 2: Start Primary
```bash
go run primary/*.go -port 5001 -backup localhost:5002
```

### Terminal 3: Run Client
```bash
go run client/*.go -server localhost:5001
```

## System Architecture

- **Primary (port 5001)**: Handles client requests, executes operations, replicates to backup
- **Backup (port 5002)**: Receives updates from primary, maintains identical state
- **Client**: Sends bids and queries to primary

## Replication Protocol

Implements the 5-stage primary-backup protocol:
1. **Request**: Client sends bid to primary
2. **Coordination**: Primary checks for duplicate requests (idempotency)
3. **Execution**: Primary executes bid on local auction state
4. **Agreement**: Primary replicates to backup and waits for ACK
5. **Response**: Primary responds to client only after backup confirms

## Testing

The client runs automated test scenarios:
- Normal bidding progression
- Invalid bids (too low, negative, zero)
- Same bidder placing multiple bids
- Result queries

## Fault Tolerance

System tolerates 1 crash failure:
- f=1 requires f+1=2 replicas (primary + backup)
- Backup acknowledges updates before primary responds to client
- Ensures no lost updates if primary crashes

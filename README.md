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

### Terminal 1: Start Backup
```bash
go run backup/main.go -port 5001
```

### Terminal 2: Start Primary
```bash
go run primary/main.go -port 5000 -backup localhost:5001
```

### Terminal 3: Run Client
```bash
go run client/main.go -server localhost:5000
```

## System Architecture

- **Primary (port 5000)**: Handles client requests, executes operations, replicates to backup
- **Backup (port 5001)**: Receives updates from primary, maintains identical state
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

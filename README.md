# impl-raft-go

A Go implementation of the Raft consensus algorithm.

## Getting Started

### Prerequisites

*   **Go 1.23+**

### Build

```bash
go build -o raft-node .
```

## Configuration

The server is configured using command-line flags.

| Flag | Type | Description |
| :--- | :--- | :--- |
| `-id` | `uint64` | **Required**. The unique ID of this server instance. Must be a non-zero integer. |
| `-addrs` | `string` | **Required**. A comma-separated list of all peers in the cluster, formatted as `ID=IP:PORT`. |

## Usage Example

To run a 3-node cluster locally, open three terminal terminals and run the following commands:

**Node 1**
```bash
./raft-node -id 1 -addrs "1=127.0.0.1:8081,2=127.0.0.1:8082,3=127.0.0.1:8083"
```

**Node 2**
```bash
./raft-node -id 2 -addrs "1=127.0.0.1:8081,2=127.0.0.1:8082,3=127.0.0.1:8083"
```

**Node 3**
```bash
./raft-node -id 3 -addrs "1=127.0.0.1:8081,2=127.0.0.1:8082,3=127.0.0.1:8083"
```

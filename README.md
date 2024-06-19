Pi\_t Experiment
================

Overview
--------

This repository serves as a distributed system project that simulates onion routing with nodes registering to a bulletin board, sending and processing messages securely. Each node maintains a queue of incoming messages, reports its queue length to the bulletin board, and processes messages upon receiving a start signal from the bulletin board.

Features
--------

*   **Node Registration**: Nodes register with the bulletin board and receive an acknowledgment.
*   **Queue Management**: Nodes handle client requests, queue messages, and periodically report queue lengths.
*   **Message Processing**: Nodes build and process onions (messages) and start processing upon receiving a signal from the bulletin board.
*   **Bulletin Board**: Manages active nodes and coordinates message processing.

Installation
------------

1.  Clone the repository:

```bash
git clone https://github.com/HannahMarsh/pi_t-experiment.git cd pi_t-experiment
```

2.  Install dependencies:

```bash 
go mod tidy
```

Usage
-----

### Running the Bulletin Board

```bash
go run cmd/bulletin-board/main.go
```

### Running a Node

```bash
go run cmd/node/main.go -id=1
```

### Endpoints

*   **Register Node**: `POST /register`
*   **Get Active Nodes**: `GET /nodes`
*   **Receive Message**: `POST /receive`
*   **Start Run**: `POST /start`

Example Workflow
----------------

1.  Start the bulletin board.
2.  Start multiple nodes.
3.  Nodes periodically report their queue lengths.
4.  Nodes receive messages from clients and build onions.
5.  The bulletin board signals nodes to start processing when conditions are met.

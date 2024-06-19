Implementing Pi\_t
================

## Introduction

This project focuses on implementing Pi_t, a differentially anonymous mixnet architecture, to explore its performance under 
various conditions. We will conduct experiments to determine the minimum number of rounds required for a given server load 
and desired parameters ϵ and δ. The experiment will be deployed on AWS with potentially hundreds of nodes, each acting as 
a relay, and will use a bulletin board to manage node communication.

## Background

An anonymous communication channel allows parties to communicate over the Internet while concealing their identities. 
Onion routing is a widely used technique where messages are encapsulated in layers of encryption and sent through a series 
of intermediary nodes (relays). This project implements Pi_t, an advanced mixnet architecture that enhances the standard 
onion routing by ensuring differential privacy.

Differential privacy in this context means that the observations of an adversary cannot significantly distinguish between 
any two communication patterns, thereby protecting the anonymity of the communicating parties.

## Objectives

1. **Deploy Pi_t on AWS**: Set up a network of nodes acting as relays.
2. Implement a replicated bulletin board to list all participating nodes and their public keys.
3. **Determine optimal parameters**: Run experiments to find the minimum number of rounds required for specific values of \( \epsilon \) and \( \delta \), while considering server load and churn rates.

## Experiment Design
The experiment will involve the following steps:

1. **Setup Nodes on AWS**:
  - Deploy hundreds of nodes on AWS, each configured to act as a relay.
  - Ensure nodes can communicate with the bulletin board to register their status.

2. **Bulletin Board**:
  - Implement a fault-tolerant, replicated bulletin board that maintains a list of active nodes and their public keys.
  - The bulletin board will also broadcast start times and coordinate the rounds of message passing.

3. **Message Passing and Rounds**:
  - Nodes will send messages in rounds, encapsulating each message in multiple layers of encryption (onions).
  - Each node will peel off one layer of encryption and forward the message to the next node.
  - The process will repeat for a specified number of rounds.

4. **Parameter Selection**:
  - Choose appropriate values for \( \epsilon \) and \( \delta \) to ensure differential privacy.
  - Calculate the minimum number of rounds required for these values given the server load and churn rates.

## Choosing Parameters
To determine the optimal parameters for the experiment, we consider the following:

- **Onion Size and Layers**: The size of each onion depends on the number of layers, which corresponds to the number of rounds.
- **Server Load**: Given a server load \( x \) and desired \( \epsilon \) and \( \delta \) values, determine the minimum number of rounds required.
- **Churn Rate**: The rate at which nodes go offline (churn rate) affects the maximum number of rounds for maintaining the desired message delivery rate.


Installation
------------

1.  Clone the repository:

```bash
git clone https://github.com/HannahMarsh/pi_t-experiment.git;
cd pi_t-experiment
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

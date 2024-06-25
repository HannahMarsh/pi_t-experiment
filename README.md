Implementing &Pi;<sub>t</sub>
================


**TODO**
- Client: Encrypt nonces into header
- Client: use prf to calculate when to generate checkpoint onions (when forming onion)
- Client: calculate how many checkpoint onions each node on path should expect
- Client calculate time window for when onion should arrive at each hop
- Node: calculate expected number of nonces for each layer
- Node: check if onion is late or the nonce is not in expected set, add bruises if so
- 

## Introduction

This project focuses on implementing &Pi;<sub>t</sub>, a differentially anonymous mixnet architecture, to explore its performance under
various conditions. We will conduct experiments to determine the minimum number of rounds required for a given server load
and desired parameters ϵ and δ. The experiment will be deployed on AWS with potentially hundreds of nodes, each acting as
a relay, and will use a bulletin board to manage node communication.

## Background

An anonymous communication channel allows parties to communicate over the Internet while concealing their identities.
Onion routing is a widely used technique where messages are encapsulated in layers of encryption and sent through a series
of intermediary nodes (relays). This project implements &Pi;<sub>t</sub>, an advanced mixnet architecture that is designed to enhance 
anonymity in asynchronous networks by introducing the concept of "bruising" onions. This protocol ensures secure and anonymous 
communication while handling delays and tampering effectively.

## Components

### Clients

### Bulletin Board

### Mixers

### Gatekeepers



## Protocol

### Parameters
- **_x_**: The server load (number of onions each node receives per round).
- **_L_**:The length of the routing path (number of hops).
- **_&tau;_**: The fraction of expected checkpoint onions needed for a node to progress its local clock.
- **_h_**: The heartbeat interval in seconds
- **_N_**: The minimum number of active nodes in the network at the start of the protocol.
- **_R_**: Approximate ratio of gatekeepers to mixers in a routing path.
- **_&epsilon;_**: The privacy loss in the worst case scenario.
- **_&delta;_**: The probability of differential privacy violation due to the adversary's actions.
- Threshold for number of bruises before an onion is discard by a gatekeeper.

### No Global Clock:

- Each node maintains a local clock (_c<sub>j</sub>_) to track the progression of onion layers. A node does not progress 
its local clock until it receives a sufficient number of checkpoint onions for the current layer (specified by _&tau;_).

### Session Keys:

- A client _k_ shares session keys _sk<sub>i,k</sub>_ with each intermediary node _P<sub>i</sub>_ using the Diffie-Hellman key exchange.
- These keys are used by the client and nodes with pseudorandom functions _F1_(_sk<sub>i,k</sub>_, _j_) and _F2_(_sk<sub>i,k</sub>_, _j_).
  - **_F1_(_sk<sub>i,k</sub>_, _j_)**: If the result is 0, then a checkpoint onion is expected to be received by _P<sub>i</sub>_ at hop _j_ and _y_ = **_F2_(_sk<sub>i,k</sub>_, _j_)** is used to calculate the expected nonce of that checkpoint onion.
- **Checkpoints (_Y<sub>k</sub>_)**: The set of expected nonces (calculated by _F2_) for the _k_-th layer checkpoint onions.

### Node / Client Registration:

- Nodes publish their existence and public keys to the bulletin board.
  - Nodes send periodic heartbeat messages so that the bulletin board can maintain a count of all active nodes in the network.
- Clients register their intent to send messages with the bulletin board.
- When a sufficient number of nodes have registered, the bulletin board broadcasts a start signal

### 1. Initialization:


- When a client _k_ is notified of the start of a run, it receives a list of active nodes that the bulletin board sees in the network.
- For each active node _P<sub>i</sub>_, the client performs a Diffie-Hellman key exchange to establish a session key _sk<sub>i,k</sub>_.
  - The client generates a Diffie-Hellman key pair and sends the public key to _P<sub>i</sub>_.
  - _P<sub>i</sub>_ generates its own Diffie-Hellman key pair, computes the shared session key, and sends its public key back to the client.
- For each message to be sent, the client constructs a routing path by selecting a random subset of 
[Mixers](#3-mixing-and-bruising) and [Gatekeepers](#5-gatekeeping) in the network.
    - The first node in the path is always a Mixer.
    - The last node before the final destination is always a Gatekeeper.
- This routing path is used to construct the onion.
- ([FormOnion](https://github.com/HannahMarsh/&Pi;<sub>t</sub>-experiment/blob/main/internal/&Pi;<sub>t</sub>/&Pi;<sub>t</sub>_functions.go#:~:text=func%20FormOnion)):
  The onion is constructed in layers, with the innermost layer containing the message encrypted with the recipient's public key.
  - Each subsequent layer _j_ contains encrypted metadata that includes:
    - A pseudorandom nonce that is unique to each onion (used to detect tampering).
    - The window of expected arrival time for the onion (used to detect delayed arrival).
    - The next hop in the routing path.
- For each participant in the routing path, the client uses its corresponding session key and pseudorandom function F1 
to determine if it should create a checkpoint onion for that layer. It then uses F2 to generate a nonce for each 
checkpoint onion with its own random routing path.
  - The construction of checkpoint onions follows the same layer-by-layer encryption process as the regular onions. The 
  only difference is that checkpoint onions (a.k.a. dummy onions) don't carry a payload, and instead their purpose is to 
  provide cover for the "real" payload-carrying onions.
- All onions are sent to their first hop (a Mixer).

### 3. Mixing and Bruising:

- When a Mixer receives an onion and decrypts its outer layer, it reveals the following data:
  - A bruise counter that tracks the number of times the onion has been detected by Mixers to be delayed or tampered with.
  - The nonce (decrypted using the session key shared with the original sender).
  - The window of expected arrival time for the onion.
  - The next hop in the path (another Mixer or a Gatekeeper).
- The Mixer checks for delays or signs of tampering.
  - To detect a delay, the mixer compares the received time with an expected time window. If an onion arrives outside this window, it is considered delayed.
  - To check for tampering, the mixer verifies the nonce against its expected set _Y<sub>k</sub>_ (calculated with session key).
    - If the nonce is valid, the node removes the nonce from _Y<sub>k</sub>_.
    - Otherwise, the onion is considered tampered with.
- If the onion is delayed or tampered with, the Mixer increments a "bruise" counter on the onion, which is encrypted with the public key of the next hop.
- The onion is then forwarded to the next node in the path.

### 4. Intermediate Nodes:

- The onion continues to travel through the network of Mixers:
  - Each Mixer decrypts its layer, possibly adds bruises, and forwards the onion.
  - This process continues until the onion reaches a Gatekeeper.

### 5. Gatekeeping:

- The Gatekeeper receives the onion and checks the bruise counter. 
- If the bruise counter exceeds a predefined threshold, the Gatekeeper discards the onion.
  - A threshold is determined based on the network's tolerance for delays and tampering
- If the onion is acceptable, the Gatekeeper forwards it to the next node (which can be another Mixer or a Gatekeeper, depending on the path).

### 6. Final Destination

- The recipient (client) always receives the onion from a Gatekeeper, never directly from a Mixer.
- The recipient receives the onion and decrypts it using their private key.
- The message is revealed, completing the communication process.

## Adversary Simulation Framework

### Potential Adversarial Functions:

- Observe all received onions and their metadata. 
- Modify the bruise counter or other metadata.
- Selectively drop onions to cause disruption, such as making onions appear delayed or tampered with when they reach the next hop.
- Inject their own onions to create noise or mislead other nodes.

### Verifying Differential Privacy:

1. Create neighboring pairs of datasets that differ by one message or communication path.
2. Run the protocol on both neighboring datasets.
3. Record the adversary’s view for each dataset.
4. Measure how the distributions of the adversary’s views differ between the neighboring datasets. 
5. Calculate the empirical probability of the adversary’s view for each dataset.
6. Verify that the privacy loss conforms to the differential privacy inequality (for &epsilon; and &delta;).



## Notes

### No Global Clock:

- In the &Pi;<sub>t</sub> protocol, each node maintains a local clock (_c<sub>j</sub>_) to track the progression of onion layers.
  - **Threshold (_&tau;_)**: A system parameter representing the fraction of checkpoint onions needed for the node to progress its local clock.
  - **Checkpoints (_Y<sub>k</sub>_)**: A set of expected nonces for the k-th layer checkpoint onions.

1. **Receiving Onions**:
    - A node _P<sub>i</sub>_ (acting as a mixer) receives an onion _O_ and determines whether it was received "on time" 
   or not relative to _P<sub>i</sub>_'s local clock.
    - If the onion _O_ arrived late, _P<sub>i</sub>_ bruises the onion and forwards the bruised onion _O'_ to the next destination.

2. **Processing Onions**:
    - If _P<sub>i</sub>_ is the last mixer on the routing path, it sends the peeled onion _O'_ to the first gatekeeper _G<sub>1</sub>_.
    - If _P<sub>i</sub>_ is either early or on time, it places the peeled onion _O'_ in its message outbox.

3. **Checking Nonces**:
    - If processing _O_ reveals a non-empty nonce _y_ &ne; &perp;, _P<sub>i</sub>_ checks whether _y_ belongs to the set 
   _Y<sub>k</sub>_ (the set of _k_-th layer checkpoint nonces P<sub>i</sub> expects to see from the onions it receives).
    - If _y_ is expected, _P<sub>i</sub>_ increments _c<sub>k</sub>_ by one and updates _Y<sub>k</sub>_ to exclude _y_.

4. **Advancing the Local Clock**:
    - Upon processing a sufficient number of j-th layer onions (i.e., if _c<sub>j</sub>_ &geq; &tau; |_Y<sub>j</sub>_|), 
   _P<sub>i</sub>_ sends out these onions (but not the onions for future hops) in random order and advances its local clock _c<sub>j</sub>_ by one.
    - Onions are batch-processed and sent out in random order at honest intermediaries when batch-processed.

### Example Workflow

1. **Node Registration**:
  - Nodes register with the bulletin board, indicating their willingness to participate in the protocol. The bulletin board keeps track of active nodes and sends a start signal when enough nodes have registered.

2. **Sending and Receiving Onions**:
  - A client sends both regular and checkpoint onions into the network.
  - Nodes receive these onions and process them according to the local clock mechanism.

3. **Detecting Late Onions**:
  - A node determines if an onion is late by comparing its arrival time to the node's internal clock. Late onions are bruised and forwarded.

4. **Verifying Nonces**:
  - For each onion, the node verifies the included nonce against its expected set Y<sub>k</sub>.
  - If the nonce is valid, the node increments its local clock counter c<sub>k</sub> and removes the nonce from Y<sub>k</sub>.

5. **Clock Advancement**:
  - Once the node processes enough onions for the current layer (meeting the threshold &tau;), it advances its local clock, prepares the onions for the next hop, and sends them out.

### Summary

In the &Pi;<sub>t</sub> protocol, nodes use local clocks to manage the timing and sequence of onion processing. This mechanism involves verifying nonces, detecting late onions, and advancing the clock based on a threshold of processed checkpoint onions. This approach ensures synchronized processing and robust detection of network disruptions without relying on a global clock.


### Path Information

- When the sender creates the onion, it includes time-related metadata in each layer.
  - This metadata can specify expected delays or time windows for each hop based on the overall path length.
  - Nodes may dynamically adjust their expectations based on real-time network conditions.
    - For instance, if a node detects increased network latency, it can widen its expected time window temporarily.

### Detailed Process

1. **Onion Creation**:
  - The sender estimates the total expected travel time based on the number of hops and typical latency.
  - This estimation includes a margin to account for variability and is included in the onion's metadata.
2. **Transmission**:
  - The sender sends the onion to the first Mixer, starting the timing process.
3. **Mixer Processing**:
  - Each Mixer, upon receiving an onion, checks the timestamp or timing information included in the onion’s layer.
  - The Mixer compares this timestamp with its synchronized clock to determine if the onion arrived within the expected time window.
4. **Threshold Checking**:
  - If the onion arrives within the expected time window, the Mixer processes and forwards it without adding bruises.
  - If the onion arrives outside the expected time window, the Mixer considers it delayed and increments the bruise counter.
5. **Forwarding**:
  - The Mixer updates the timing information in the onion’s metadata to reflect the current time and any adjustments needed for the next hop.
6. **Path Continuation**:
  - The onion continues through the network, with each subsequent Mixer performing similar checks and adjustments based on the timing metadata.

## Summary

- **Configuration and Synchronization**: Mixers use predefined parameters and synchronized clocks to determine expected time windows.
- **Metadata Inclusion**: Time-related metadata is included in each onion layer, guiding Mixers on expected arrival times.
- **Dynamic Adjustments**: Real-time network conditions may lead to dynamic adjustments in expected time windows.
- **Processing and Forwarding**: Each Mixer checks the onion's arrival time against the expected window and increments the bruise counter if necessary.

By leveraging synchronized clocks, predefined parameters, and real-time adjustments, Mixers in the Bruisable Onion protocol can effectively manage expected time windows to detect delays and maintain the integrity of the routing process.

## Objectives

1. **Deploy &Pi;<sub>t</sub> on AWS**: Set up a network of nodes acting as relays.
2. Implement a replicated bulletin board to list all participating nodes and their public keys.
3. **Determine optimal parameters**: Run experiments to find the minimum number of rounds required for specific values of
    \epsilon  and 
   \delta , while considering server load and churn rates.

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

- Choose appropriate values for  \epsilon  and  \delta  to ensure differential privacy.
- Calculate the minimum number of rounds required for these values given the server load and churn rates.

## Choosing Parameters

To determine the optimal parameters for the experiment, we consider the following:

- **Onion Size and Layers**: The size of each onion depends on the number of layers, which corresponds to the number of rounds.
- **Server Load**: Given a server load  x  and desired  \epsilon  and  \delta  values, determine the minimum
  number of rounds required.
- **Churn Rate**: The rate at which nodes go offline (churn rate) affects the maximum number of rounds for maintaining the
  desired message delivery rate.

Installation
------------

1. Clone the repository:

```bash
git clone https://github.com/HannahMarsh/&Pi;<sub>t</sub>-experiment.git;
cd &Pi;<sub>t</sub>-experiment
```

2. Install dependencies:

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

* **Register Node**: `POST /register`
* **Get Active Nodes**: `GET /nodes`
* **Receive Message**: `POST /receive`
* **Start Run**: `POST /start`

Example Workflow
----------------

1. Start the bulletin board.
2. Start multiple nodes.
3. Nodes periodically report their queue lengths.
4. Nodes receive messages from clients and build onions.
5. The bulletin board signals nodes to start processing when conditions are met.

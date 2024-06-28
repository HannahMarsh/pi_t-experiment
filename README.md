Implementing &Pi;<sub>t</sub>
================


**TODO**
- Client: calculate how many checkpoint onions each node on path should expect
- Client calculate time window for when onion should arrive at each hop
- Node: calculate expected number of nonces for each layer
- Node: check if onion is late or the nonce is not in expected set, add bruises if so


## Introduction

This project focuses on implementing &Pi;<sub>t</sub>, a differentially anonymous mixnet architecture, to explore its performance under
various conditions. We will conduct experiments to determine the minimum number of rounds required for a given server load
and desired parameters ϵ and δ. 

## Background

An anonymous communication channel allows parties to communicate over the Internet while concealing their identities.
Onion routing is a widely used technique where messages are encapsulated in layers of encryption and sent through a series
of intermediary nodes (relays). This project implements &Pi;<sub>t</sub>, an advanced mixnet architecture that ensures differential 
privacy, which means the adversary's view when Alice sends a message to Bob is statistically close to the view when Alice sends a 
message to Carol instead. This is significant because it provides privacy guarantees even if the adversary can observe a fraction 
of the network nodes and network traffic.


## &Pi;<sub>t</sub> Overview

- Each layer of the onion can be "bruised" if it does not meet certain criteria, and too many bruises will lead to the onion being dropped.
- Mixers are responsible for re-encrypting and forwarding onions to random next hops.
- Gatekeepers validate and forward onions according to a predefined path. 
- Checkpoint onions are special onions sent at various stages to verify the presence and correct operation of other nodes. 
  This way nodes can detect that it has not received the expected number of checkpoint onions, and can infer that an attack may be in progress.

  

## Components

### Bulletin Board

### Clients

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

### Summary

In the &Pi;<sub>t</sub> protocol, nodes use local clocks to manage the timing and sequence of onion processing. This mechanism involves verifying nonces, detecting late onions, and advancing the clock based on a threshold of processed checkpoint onions. This approach ensures synchronized processing and robust detection of network disruptions without relying on a global clock.


### Path Information

- When the sender creates the onion, it includes time-related metadata in each layer.
  - This metadata can specify expected delays or time windows for each hop based on the overall path length.
  - Nodes may dynamically adjust their expectations based on real-time network conditions.
    - For instance, if a node detects increased network latency, it can widen its expected time window temporarily.

Installation
------------

1. Clone the repository:

```bash
git clone https://github.com/HannahMarsh/pi_t-experiment.git;
cd pi_t-experiment
```

2. Install dependencies:

```bash 
go mod tidy
```

Usage
-----

All configurations are set in the [`config/config.yaml`](config/config/yaml) file.

### Running the Bulletin Board

```bash
go run cmd/bulletin-board/main.go
```

### Running a Node

```bash
go run cmd/node/main.go -id=1
```

### Running a Client

```bash
go run cmd/client/main.go -id=1
```

### Serving Metrics

```bash
go run cmd/metrics/main.go -port 8200
```

## Endpoints

### Bulletin Board
- **Register Client**: `POST /register`
- **Register Node**: `POST /register`
- **Get Active Nodes**: `GET /nodes`

### Node & Client
- **Receive Onion**: `POST /receive`
- **Get Status**: `GET /status`
- **Start Run**: `POST /start`

### Metrics
- **Messages**: `GET /messages`
- **Nodes**: `GET /nodes`
- **Clients**: `GET /clients`
- **Checkpoint Onion Counts**: `GET /checkpoints`
- **Visualize Onion Paths**: `GET /visualization`


For a small number of clients/nodes, this makes debugging easier.

![](img/vis.png)


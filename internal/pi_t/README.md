# Directory: `/internal/pi_t`  


This directory contains the core implementation of the $\Pi_t$ onion routing protocol. The key components include functions
for forming and peeling onion layers, as well as supporting cryptographic operations and onion structure management.

## Directory Structure

```bash
├── crypto
│   └── keys
│       ├── ecdh.go
│       └── ephemeral.go
├── formOnion.go
├── onion_model
│   ├── content.go
│   ├── header.go
│   ├── onion.go
│   └── sepal.go
└── peelOnion.go
```

Key Components
--------------

- **[formOnion.go](/internal/pi_t/formOnion.go)**: Implements the `FormOnion` function.
- **[peelOnion.go](/internal/pi_t/peelOnion.go)**: Implements the `PeelOnion` function.
- **[crypto/keys](/internal/pi_t/crypto/keys) (dir)**: Provides functions for key generation, encryption, and decryption.
- **[onion_model](/internal/pi_t/onion_model) (dir)**: Defines the onion structure, including the [header](/internal/pi_t/onion_model/header.go), [content](/internal/pi_t/onion_model/content.go), and [sepal](/internal/pi_t/onion_model/sepal.go).

Usage
-----

### Key Generation

Generate a ($sk$, $pk$) key pair:

```go
privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
```

### Forming a Message-Bearing Onion

```go
// First, determine routing path
mixers := []string{"192.168.252.191:8080", "192.158.1.38:8080"} // assume each is chosen randomly
gatekeepers := []string{"168.212.226.204:8080", "192.168.123.132:8080"} // assume each is chosen randomly
recipient := "192.168. 4.10:8080" // final client destination

// Next, gather public keys (procided by bulletin board)
publicKeys := []string{mixerPk1, mixerPk2, gatekeeperPk1, mixerPk2, recipientPk}

// Construct empty metadata array
metadata := make([]onion_model.Metadata, 6)

// Create and marshall a message which will be revealed by the client receiver at the end of the protocol
msg := NewMessage("Hello " + recipient + "!"),
msgBytes, _ := json.Marshal(msg)

// Form the onion
allPossibleOnions, err := pi_t.FormOnion(string(msgBytes), mixers, gatekeepers, recipient, publicKeys, metadata, 2)

// Send the fully constructed onion (located at position [0][0]) to the first mixer
SendOnion(allPossibleOnions[0][0], mixers[0])
```

### Forming a Checkpoint Onion

```go
mixers :=       // provided by bulletin board
gatekeepers :=  // provided by bulletin board
recipient := "192.168. 4.10:8080" // a client recipient is chosen randomly

publicKeys := []string{mixerPk1, mixerPk2, gatekeeperPk1, mixerPk2, recipientPk} // provided by the bulletin board
metadata := // provided by the bulletin board

// Form the checkpoint onion (with an empty message)
allPossibleOnions, err := pi_t.FormOnion("", mixers, gatekeepers, recipient, publicKeys, metadata, 2)

// Send the fully constructed checkpoint onion (located at position [0][0]) to the first mixer
SendOnion(allPossibleOnions[0][0], mixers[0])
```

### Peeling an Onion and Bruising

**`PeelOnion` example usage:**

```go
role, layer, metadata, peeledOnion, nextDestination, err := PeelOnion(onionBase64String, privateKeyPEM)
if err != nil {
    log.Fatalf("PeelOnion() error: %v", err)
}
// `role`: Indicates the role of the current relay processor. Values include "mixer", "gatekeeper", or "finalGatekeeper".
// `layer`: The current layer.
// `metadata`: Metadata associated with the peeled layer. If it is a checkpoint onion, it contains the nonce. Otherwise, it is empty.
// `peeledOnion`: The peeled onion which may still need to be processed before ready to be forwarded to the next hop.
// `nextDestination`: The address of the next node in the path.

nonce := metadata.Nonce

if role == onion_model.MIXER {      // If our role is a mixer...
  if nonce != "" {                  // and if the onion is a checkpoint onion
	  
    // Verify the nonce to check if we need to remove a null block of key block
	  
    if verifyNonce(nonce, layer) {
		    // If the nonce is valid, remove the right-most null block
        peeled.Sepal = peeled.Sepal.RemoveBlock()
    } else {                        
		    // Otherwise, bruise the onion by removing the left-most key block
        peeled.Sepal = peeled.Sepal.AddBruise()
    }
  }
} else {                            // If our role is a gatekeeper
	  // Check if there are any valid key slots left
}
// Forward to the next hop
```

### Onion Components

The `onion_model` package includes:


#### Header ([onion_model/header.go](/internal/pi_t/onion_model/header.go)):

- Consists of three parts: the [ ephemeral public key $`epk_{i}`$ ](#epk), a [ ciphertext $`E_i`$ ](#Ei) and the [ rest of the header $`B_i`$ ](#Bi).
  - **$`epk_i`$**: The ephemeral public key is used along with $`P_i`$'s private key $sk_i$ to compute the shared secret $s_i$ from the original client. <a name="epk"></a>
  - **$E_i$**: An encryption under the shared secret $s_i$ of the tuple <a name="Ei"></a> $(i, y_i, k_i)$ where:
    - $i$&nbsp; is the position in the route.
    - $y_i$ is the metadata.
    - $k_i$ is the layer key (the shared key for that layer). <a name="layer-key"></a>
  - **$B_i$**: The rest of the header, which includes: <a name="Bi"></a>
    - The nonce $y_i$.
    - The time window for the onion's arrival.
    - The next hop in the routing path.

#### Content ([onion_model/content.go](/internal/pi_t/onion_model/content.go)):

- Contains the payload or the next layer of the onion.
- Encrypted under the [ layer key, $`k`$ ](#layer-key).
- For intermediate relays, it contains the encrypted content of the next onion layer.
- For the final recipient, it contains the actual message.

#### Sepal ([onion_model/sepal.go](/internal/pi_t/onion_model/sepal.go)):

- Protects the inner layers of the onion by absorbing bruises (caused by delays, tampering, or replay attacks) during transit.
- Consists of key-blocks and null-blocks.
- The key-blocks are encrypted versions of the bulb master key $K$.
- The null-blocks are encrypted versions of the value 0.
- As each mixer processes the onion, it peels a layer from the sepal:
  - If unbruised, the rightmost sepal block is dropped, retaining the same number of key blocks.
  - If bruised, the leftmost sepal block is dropped, reducing the number of key blocks.
  - This ensures that if the number of bruises exceeds a threshold $d$, the final gatekeeper cannot recover the master key $K$, making the onion
    undecryptable.

    
Example Workflow
----------------

1.  Clients and relays generate key pairs using the function `keys.KeyGen()`. 
2.  Clients and relays register their public keys with the bulletin board.
3.  The bulletin board calculates checkpoint onions based on the server load and number of registered participants.
4.  The bulletin board signals the start of a run.
     - Clients are sent a `ClientStartRunApi` which includes their checkpoint onions and all public keys.
     - Relays are send a `RelayStartRunApi` which includes the nonces they should expect to receive at each layer.
5.  Clients construct a routing path for each real and checkpoint onion to be sent, then uses `FormOnion()`.
     - Checkpoint onions have metadata which includes the nonce for each layer.
     - Message-bearing onions have empty metadata.
6.  Clients send the onions to the first hop in the path.
7.  When receiving an onion, a relay calls `PeelOnion()` and adds bruises/removes null blocks when necessary. Then it sends tot he next hop.
8.  Finally, a client receives the onion, peels it, and reveals the message.


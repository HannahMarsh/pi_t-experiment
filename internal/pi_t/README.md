# Directory: `/internal/pi_t`  
  
  
- This directory contains the implementation of several Onion Routing functions from Pi_t. The key components are:
  - [`pi_t_functions.go`](pi_t_functions.go): Functions to form and peel onion layers.
  - [`keys`](tools/keys): Helper package for key generation and encryption/decryption.
  - [`prf`](prf): Package for Pseudo-Random Functions (PRF) used in the protocol.

## Usage

### Key Generation

Generate an ECDH key pair:

```go
privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
if err != nil {
    log.Fatalf("KeyGen() error: %v", err)
}
```

### Forming an Onion

Create an onion with the `FormOnion` function:

```go
destination, onion, err := pi_t.FormOnion(privateKeyPEM, publicKeyPEM, payload, publicKeys, routingPath, checkpoint)
if err != nil {
    log.Fatalf("FormOnion() error: %v", err)
}
```

### Peeling an Onion

Peel an onion layer with the `PeelOnion` function:

```go
peeled, bruises, nonceVerification, err := pi_t.PeelOnion(onion, privateKeyPEM)
if err != nil {
  log.Fatalf("PeelOnion() error: %v", err)
}
```

### Adding a Header After Peeling

After peeling an onion layer and obtaining the OnionPayload, you might need to re-encrypt it with a new header, typically 
to forward it to the next hop in the routing path. The `AddHeader` function returns the base64-encoded onion with the 
updated bruise counter and the necessary encryption information for the next hop. 

```go
headerAdded, err := AddHeader(peeled, bruises + 1, privateKeyPEM, publicKeyPEM)
if err != nil {
  log.Fatalf("AddHeader() error: %v", err)
}
// `peeled`: The OnionPayload obtained from peeling the onion layer. It contains the decrypted payload and metadata about the current layer.
// `bruises + 1`: The new bruise counter value.
// `privateKeyPEM1`: The PEM-encoded private key of the current node. This key is used to decrypt the shared key and re-encrypt the payload.
// `publicKeyPEM1`: The PEM-encoded public key of the current node. This key is included in the header as the sender's public key for the next node.
```



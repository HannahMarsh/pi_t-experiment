# Implementing $`\Pi_t`$ :tulip:

## Introduction

This project aims to implement the $`\Pi_t`$ (_"t"_ for _"tulip"_ or _"threshold"_) protocol in a service-model environment.
The focus of this experiment is on evaluating and comparing the
performance and privacy guarantees of $`\Pi_t`$ against other protocols (such as those described in [\[VDHLZZ15\]](#vdHLZZ15)
and [\[TLG17\]](#TGL-17)) under similar conditions.

([Jump to Installation](#installation-for-development))

## Background

A communication protocol achieves anonymity if no attacker can distinguish who is communicating with whom based on 
observable data such as network traffic. Onion routing is a well-established technique for achieving anonymity, where messages 
are encapsulated in layers of encryption and sent through a series of intermediary relays (relays). However, onion routing 
_alone_ does not protect against adversaries who can observe all network traffic, such as AS-level or ISP-level attackers.

Mix networks (or _mixnets_) enhance onion routing by mixing messages at each relay, making it harder for adversaries to 
correlate incoming and outgoing messages. Various mixnet architectures have been proposed, including
[Vuvuzela](#vdHLZZ15) and [Stadium](#TGL-17). Despite their strong anonymity guarantees, these
solutions assume synchronous communication, where time progresses in rounds, and message transmissions are lossless 
and instantaneous. In practice, however, network communication is asynchronous and thus adversaries can exploit timing 
discrepancies to correlate messages entering and leaving the network.

$`\Pi_t`$ (_"t"_ for _"tulip"_ or _"threshold"_), is the first provably anonymous onion 
routing protocol for the asynchronous communications setting. As described in [\[ALU24\]](#ALU24), this protocol introduces 
several novel concepts such as [checkpoint onions]() and [bruising]().
Theoretical analysis demonstrates that $`\Pi_t`$ can provide _differential privacy_ (see definition [below](#differential-privacy)) even under strong 
adversarial models. This analysis assumes a peer-to-peer network where relays must discover each other, exchange keys, and
manage communication paths independently. Unfortunately this leads to several practical challenges such as increased complexity, lower fault tolerance,
and inconsistent security enforcement.

This project aims to transition $`\Pi_t`$ to a service-model environment by introducing a fault-tolerant bulletin board that maintains a
consistent view of all active relays and coordinates the start of a run. 

### Differential Privacy

[Differential privacy](#DMNS06) is a mathematical framework for ensuring that the results of data analysis do not reveal any specific individual's data.

In the context of $`\Pi_t`$ and other onion routing protocols, a more nuanced form of differential privacy, called [ _($`\epsilon`$, $`\delta`$)-Differential Privacy_ ](https://www.cis.upenn.edu/~aaroth/Papers/privacybook.pdf), ensures that an adversary observing network traffic cannot (with high confidence)
distinguish between two neighboring communication patterns. This means that the inclusion or exclusion of a single individual's data does not significantly affect the outcome of any analysis.

Epsilon (&epsilon;) and delta ($`\delta`$) are the parameters that define our _(&epsilon;, $`\delta`$)-differential privacy_ guarantees:
- **&epsilon;**: A non-negative parameter that bounds the multiplicative difference in probabilities of any outcome
  occurring whether an individual's data is included or not. Smaller values of &epsilon; indicate stronger privacy guarantees.
- **$\delta$**: A non-negative parameter that bounds the additive difference in probabilities, allowing for a small
  probability of error. Smaller values of $`\delta`$ also indicate stronger privacy guarantees.

Formally, a randomized algorithm or mechanism is _&epsilon;, $`\delta`$)-differentially private_ if for every pair of neighboring inputs
$`\sigma_0`$ and $`\sigma_1`$ and for every set $`\mathcal{V}`$ of adversarial views,

$$
\Pr[\text{View}^{\mathcal{A}}(\sigma_0) \in \mathcal{V}] \leq e^{\epsilon} \cdot \Pr[\text{View}^{\mathcal{A}}(\sigma_1) \in \mathcal{V}] + \delta
$$


This means the adversary's view when Alice sends a message to Bob is statistically close to the 
view when Alice sends a message to Carol instead. 



## $`\Pi_t`$ Implementation Overview

&nbsp;


<figure>
  <figcaption><b>Figure 1</b> - <em>Routing Path Visualization, Example "Scenario 0" (with N clients, R Relays, l1 mixers, and l rounds)</em></figcaption>
  <img src="img/onion-routing.png" alt="Routing Path Visualization" width="100%"/>
</figure>

### Parameters
(_also defined in [/config/config.yml](config/config.yml)_)

- **$N$**: The minimum number of clients.
- **$n$**: The minimum number of relays.
- **$x$**: Server load (i.e. the expected number of onions each relay processes per round).
- **$\ell_1$**: The number of mixers in each routing path.
- **$\ell_2$**: The number of gatekeepers in each routing path.
- **$L$**: The number of rounds (also the length of the routing path, equal to $`\ell_1 + \ell_2 + 1`$ ).
- **$d$**: The number of non-null key-blocks in $S_1$. (thus $d$ is the threshold for number of bruises before an onion is discard by a gatekeeper).
- **$\tau$**: ( $\tau \lt \(1 − \gamma\)\(1 − \chi\)$ ) The fraction of expected checkpoint onions needed for a relay to progress its local clock.
- **$\epsilon$**: The privacy loss in the worst case scenario.
- **$\delta = 10^{-4}$**: The fixed probability of differential privacy violation.
- **$\lambda$**: The security parameter. We assume every quantity of the system, including $`N`$, $`R`$, $`L`$ are polynomially bounded by $`\lambda`$.
- **$\theta$**: The maximum fraction of bruisable layers that can be bruised before the innermost tulip bulb becomes 
  unrecoverable. Note that $d = \theta \cdot \ell_1$
- **$\chi$**: The fraction of $N$ relays that can be corrupted and controlled by the adversary (the subset is chosen prior to execution). Note that $\chi \lt \theta - 0.5$ and $\chi \lt \frac{d}{\ell_1} - 0.5$
- **`BulletinBoardUrl`**: The IP address and port of the bulletin board.
- **`MetricsPort`**: The port that all aggregated metrics are served (on the Bulletin board's IP address).

### No Global Clock:

- Each relay maintains a local clock ($c_j$) to track the progression of onion layers. A relay does not progress   
  its local clock until it receives a sufficient number of checkpoint onions for the current layer (specified by $\tau$).

### Keys:

- **_Long-term identity keys_** ($pk_{i}$, $sk_{i}$) are established for each party $P_i$ (both clients and relays). They are persistent and are used across multiple sessions.
- **_Ephemeral session keys_** ($epk_{i,j}$, $esk_{i,j}$) are generated by a client for each $P_i$ in the $j$-th position in an onion's routing path. They are short-term, meaning a processing party $P_i$ can only compute the shared secret during round $j$. (See [/internal/pi_t/tools/keys/ephemeral.go](https://github.com/HannahMarsh/pi_t-experiment/blob/main/internal/pi_t/tools/keys/ephemeral.go) for implementation)
    - **Onion formation**: The client uses $esk_{i,j}$ and $P_i$'s public key $pk_i$ to generate a shared secret $s_{i,j}$.
      - The client includes $epk_{i,j}$ in the $j$-th layer's header.
    - **Peeling**: The processing party $P_i$ uses the [ ephemeral public key $`epk_{i,j}`$ ](#epk) and its own private key $sk_i$ to compute the shared secret $s_{i,j}$.
      - $s_{i,j}$ is then used by the processing party to decrypt the header's [ ciphertext $`E_i`$ ](#Ei)
- See [internal/pi_t/keys/ecdh.go](internal/pi_t/tools/keys/ecdh.go) for this project's ECDH usage.


### Tulip Bulb Structure:

<table>
  <tr>
    <td>Header ($H_i$)</td>
    <td>Content ($C_i$)</td>
    <td>Stepel ($S_i$)</td>
  </tr>
  <tr>
   <td>
    <table>
     <tr>
      <td> $E_i$ </td>
      <td> $B_{i,1}$ </td>
      <td> $B_{i,2}$ </td>
      <td> ... </td>
      <td> $B_{i,l-1}$ </td>
     </tr>
    </table>
   </td>
   <td>
    <table>
     <tr>
      <td> $\{ . . . \{ \{ \{m\}_{k_{l}} \}_{k_{l-1}} \}_{k_{l-2}} . . . \}_{k_{1}}$ </td>
     </tr>
    </table>
   </td>
   <td>
    <table style="border: 0;">
     <tr>
      <td>
       $\langle K \rangle$-blocks:<br>
       <table>
        <tr>
         <td> $S_{i,1}$ </td>
         <td> $S_{i,2}$ </td>
         <td> ... </td>
         <td> $S_{i,d}$ </td>
        </tr>
       </table>
      </td>
       <td>
        $\langle 0 \rangle$-blocks:<br>
       <table>
        <tr>
         <td> $S_{i,d+1}$ </td>
         <td> $S_{i,d+2}$ </td>
         <td> ... </td>
         <td> $S_{i,l_{1}+1}$ </td>
        </tr>
       </table>
      </td>
     </tr>
    </table>
   </td>
  </tr>
</table>

<table>
 <tr>
  <td>Header ($H_i$)</td>
 </tr>
 <tr>
  <td>
   <table>
    <tr>
     <td>$E_i$</td>
     <td>$B_i$</td>
    </tr>
    <tr>
     <td>
      Enc( <br>
      &nbsp; $pk$ ( $P_i$ ), <br>
      &nbsp; $t_i$ ,  <br>
      &nbsp; ( Role, $i$ , $y_i$ , $\vec{A}_i$ , $k_i$ ) <br>
      )
     </td>
     <td>
      <table>
       <tr>
        <td>$B_{i,1} = $</td>
        <td>$B_{i,2} = $</td>
        <td>...</td>
        <td>$B_{i,l-1} = $</td>
       </tr>
       <tr>
        <td>
         <table>
          <tr>
           <td>
            Encrypted with $k_{i}$:
           </td>
          </tr>
          <tr>
           <td>
            <table>
             <tr> <td> $I_{i+1}$ </td> <td> $E_{i+1}$ </td> </tr>
            </table>
           </td>
          </tr>
         </table>
        </td>
        <td>
         <table>
          <tr>
           <td>Encrypted with $k_{i}$:</td>
          </tr>
          <tr>
           <td>
            <table>
             <tr>
              <td>
               Encrypted with $k_{i+1}$:
              </td>
             </tr>
             <tr>
              <td>
               <table>
                <tr> <td> $I_{i+2}$ </td> <td> $E_{i+2}$ </td> </tr>
               </table>
              </td>
             </tr>
            </table>
           </td>
          </tr>
         </table>
        </td>
        <td>...</td>
        <td>
         <table>
          <tr>
           <td>Encrypted with $k_{i}$:</td>
          </tr>
          <tr>
           <td>
            <table>
             <tr>
              <td>Encrypted with $k_{i+1}$:</td>
             </tr>
             <tr>
              <td>
               <table>
                <tr>
                 <td>...</td>
                </tr>
                <tr>
                 <td>
                  <table>
                   <tr>
                    <td>
                     Encrypted with $k_{i+l-2}$:
                    </td>
                   </tr>
                   <tr>
                    <td>
                     <table>
                      <tr> <td> $I_{i+l-1}$ </td> <td> $E_{i+l-1}$ </td> </tr>
                     </table>
                    </td>
                   </tr>
                  </table>
                 </td>
                </tr>
               </table>
              </td>
             </tr>
            </table>
           </td>
          </tr>
         </table>   
        </td>
       </tr>
      </table>
     </td>
    </tr>
   </table>
  </td>
 </tr>
</table>

#### Header ($H$):

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

#### Content ($C$):

- Contains the payload or the next layer of the onion.
- Encrypted under the [ layer key, $`k`$ ](#layer-key).
- For intermediate relays, it contains the encrypted content of the next onion layer.
- For the final recipient, it contains the actual message.

#### Sepal ($S$):

- Protects the inner layers of the onion by absorbing bruises (caused by delays, tampering, or replay attacks) during transit.
- Consists of key-blocks and null-blocks.
- The key-blocks are encrypted versions of the bulb master key $K$.
- The null-blocks are encrypted versions of the value 0.
- As each mixer processes the onion, it peels a layer from the sepal:
  - If unbruised, the rightmost sepal block is dropped, retaining the same number of key blocks.
  - If bruised, the leftmost sepal block is dropped, reducing the number of key blocks.
  - This ensures that if the number of bruises exceeds a threshold $d$, the final gatekeeper cannot recover the master key $K$, making the onion
    undecryptable.

### 1. Relay / Client Registration:

- Relays publish their existence and public keys to the bulletin board.
  - See [internal/model/relay/relay.go](internal/model/relay/relay.go)
  - Relays send periodic heartbeat messages so that the bulletin board can maintain a list of all active relays in the network.
- Clients register their intent to send messages with the bulletin board.
  - See [internal/model/client/client.go](internal/model/client/client.go)
- When a sufficient number of relays are active (given by $N$ ), and a sufficient number of clients have registered their 
  intent-to-send messages (given by $R$ ), the bulletin board broadcasts a start signal along with the following information.
  - Each participating client receives:
    - a list of active Mixers and Gatekeepers (along with their public keys and which checkpoint onions the client needs to create).
  - All participating relays receive:
    - a list of expected nonces it should receive for each round _j_. 
  - See [internal/model/bulletin_board/bulletin_board.go](internal/model/bulletin_board/bulletin_board.go)

### 2. Initialization:

- When a client $k$ is notified of the start of a run, it receives from the bulletin board:
  - A list of participating Mixers and Gatekeepers where each relay relay $P_i$ is associated with a public key $pk_i$ and a 
    list of sets P_$Y_1,...,Y_{l_1}$ , where $Y_j$ represents the subset of nonces $P_i$ expects to receive during round _j_ 
    which the client is responsible for sending.
- For each message to be sent, the client constructs a routing path by selecting a random subset of $l_1& [Mixers](#3-mixing-and-bruising)
  and $l_2$ [Gatekeepers](#5-gatekeeping) in the network.
  - routing_path[ $1...l_1$ ] are Mixers.
  - routing_path[ $l_1 + 1...l_1 + l_2$ ] are Gatekeepers.
  - routing_path[ $l_1 + l_2 + 1$ ] is the final destination.

#### Forming the Onion:

- The onion is constructed in layers, with the innermost layer containing the message encrypted with the recipient's public key.
  - Each subsequent layer $j$ contains encrypted metadata that includes:
    - A pseudorandom nonce that is unique to each onion (used to detect replay attacks).
    - The window of expected arrival time for the onion (used to detect delayed arrival).
    - The next hop in the routing path.
  - For each participant in the routing path, the client uses its corresponding session key and pseudorandom function F1   
    to determine if it should create a checkpoint onion for that layer. It then uses F2 to generate a nonce for each   
    checkpoint onion with its own random routing path.
    - The construction of checkpoint onions follows the same layer-by-layer encryption process as the regular onions.   
      The only difference is that checkpoint onions (a.k.a. dummy onions) don't carry a payload and instead provide cover for the "real"
      payload-carrying onions.
    - Each layer of the onion contains the encrypted shared key which is used by the next relay in the path to decrypt the layer. This shared key is
      encrypted with the public key of the respective relay and included in the header of each layer.
- All onions are sent to their first hop (a Mixer).

### 3. Mixing and Bruising:

- When a Mixer receives an onion and decrypts its outer layer (header), it reveals the following data:
  - Multiple key slots that contain copies of the decryption key. If an onion is bruised, one of these key slots is invalidated.
  - The nonce (decrypted using the session key shared with the original sender).
  - The window of expected arrival time for the onion.
  - The next hop in the path (another Mixer or a Gatekeeper).
- The Mixer checks for delays or signs of tampering.
  - To detect a delay, the mixer compares the received "time" (see [local time](#no-global-clock)) with an expected time window. If an onion arrives
    outside this window, it is considered delayed.
  - To check for tampering, the mixer verifies the nonce against its expected set $Y_k$ (calculated with session key).
    - If the nonce is valid, the relay removes the nonce from $Y_k$.
    - Otherwise, the onion is considered tampered with.
- If the onion is delayed or tampered with, the Mixer invalidates one of the key slots in the onion.
- The onion is then forwarded to the next relay in the path.
- The number of protection layers is managed in a way that does not reveal any positional information. For instance,   
  additional dummy layers might be used to mask the actual number of active layers.

### 4. Intermediate Relays:

- The onion continues to travel through the network of Mixers:
  - Each Mixer decrypts its layer, possibly adds bruises (invalidates key slots), and forwards the onion.
  - This process continues until the onion reaches a Gatekeeper.

### 5. Gatekeeping:

- The Gatekeeper receives the onion and checks the number of valid key slots.
- If the number of valid key slots is below a predefined threshold, the Gatekeeper discards the onion.
  - A threshold is determined based on the network's tolerance for delays and replay attacks
- If the onion is acceptable, the Gatekeeper forwards it to the next relay (which can be another Mixer or a Gatekeeper, depending on the path).

### 6. Final Destination

- The recipient (client) always receives the onion from a Gatekeeper, never directly from a Mixer.
- The recipient receives the onion and decrypts it using their private key.
- The message is revealed, completing the communication process.

## Adversary Simulation Framework

### Potential Adversarial Functions:

- Observe all received onions and their metadata.
- Bruise or delay onions that pass through their layer (but cannot modify bruise count).
- Selectively drop onions to cause disruption, such as making onions appear delayed when they reach the next hop.
- Inject their own onions, replicate onions (replay attack) to create noise or mislead other relays.

### Verifying Differential Privacy:

1. Create neighboring pairs of datasets that differ by one message or communication path.
2. Run the protocol on both neighboring datasets.
3. Record the adversary’s view for each dataset.
4. Measure how the distributions of the adversary’s views differ between the neighboring datasets.
5. Calculate the empirical probability of the adversary’s view for each dataset.
6. Verify that the privacy loss conforms to the differential privacy inequality (for &epsilon; and &delta;).

## Notes

### No Global Clock:

- In the $`\Pi_t`$ protocol, each relay maintains a local clock ($c_j$) to track the progression of onion layers.
  - **Threshold (_&tau;_)**: A system parameter representing the fraction of checkpoint onions needed for the relay to progress its local clock.
  - **Checkpoints ($Y_k$)**: A set of expected nonces for the k-th layer checkpoint onions.

1. **Receiving Onions**:

- A relay $P_i$ (acting as a mixer) receives an onion $O$ and determines whether it was received "on time"   
  or not relative to $P_i$'s local clock.
- If the onion $O$ arrived late, $P_i$ bruises the onion and forwards the bruised onion _O'_ to the next destination.

2. **Processing Onions**:

- If $P_i$ is the last mixer on the routing path, it sends the peeled onion _O'_ to the first gatekeeper $G_1$.
- If $P_i$ is either early or on time, it places the peeled onion _O'_ in its message outbox.

3. **Checking Nonces**:

- If processing $O$ reveals a non-empty nonce $y$ &ne; &perp;, $P_i$ checks whether $y$ belongs to the set   
  $Y_k$ (the set of $k$-th layer checkpoint nonces P<sub>i</sub> expects to see from the onions it receives).
- If $y$ is expected, $P_i$ increments $c_k$ by one and updates $Y_k$ to exclude $y$.

4. **Advancing the Local Clock**:

- Upon processing a sufficient number of j-th layer onions (i.e., if $c_j$ &geq; &tau; |$Y_j$|),   
  $P_i$ sends out these onions (but not the onions for future hops) in random order and advances its local clock $c_j$ by one.
- Onions are batch-processed and sent out in random order at honest intermediaries when batch-processed.



## Experiment Setup

- Clients, $\[C_1...C_R\]$
  - We will choose target senders, $C_1$ and $C_2$
- Relays, $\[R_1...R_N\]$
- Adversary, $`\mathcal{A}`$
  - The adversary always drops onions from $C_1$
  - $`\mathcal{A}`$'s observables, $\text{View}(\sigma_i)$, for a scenario, $i$, include the number of onions sent and received by each client and relay.
    - Let $O_{k,i}$ be the distribution (over many executions of scenario $i$) of the number of onions that client $C_k$ receives by the end of the run.

### Senarios

- We consider two neighboring scenarios for our experiment:
  - **Scenario 0 ($\sigma_0$)**:
    - $C_1$ sends a message to $C_R$
    - $C_2$ sends a message to $C_{R-1}$
  - **Scenario 1 ($\sigma_1$)**:
    - $C_1$ sends a message to $C_{R-1}$
    - $C_2$ sends a message to $C_R$

- In both scenarios, there are also dummy (checkpoint) onions to provide cover traffic.
- For example, in Scenario 1 where $C_2$ sends a message to $C_N$, the number of onions, $O_N$, received by $C_N$ will be shifted to the right by 1 compared to
  $O_{R-1}$ since $C_{R-1}$'s onion was dropped by $`\mathcal{A}`$.

### Adversary's Task

The adversary observes the network volume (number of onions each client and relay are sending and receiving), along with routing information (who each relay are sending to/receiving from each round).
<s> Each round, the adversary updates the
probability distribution of where the message-bearing onion is likely located. The adversary's goal is to determine the most probable client $\[C_2...C_N\]$
that received a message-bearing onion from $C_1$. </s>

### Computing the Adversary's Advantage

- We aim to compute the ratio that the adversary is correct (i.e., the "advantage").
- The "advantage" is essentially a measure of how well the adversary can use the observed data to make correct assumptions about which client sent the onion.
- This is ideally bounded by $e^\epsilon$.



Installation (for development)
------------  

Clone the repository:

```bash  
git clone https://github.com/HannahMarsh/pi_t-experiment.git;
cd pi_t-experiment
```

Install dependencies:

```bash
bash go mod tidy
```

Build the project:

```bash
go build -v ./...
```  

Development
-----  

Run tests:

```bash
go test -v ./...
```


Usage
-----  

All configurations are initialized in the [`config/config.yaml`](config/config/yaml) file.

### Running the Bulletin Board

```bash  
go run cmd/bulletin-board/main.go -logLevel=<logLevel>
```  
- Options:
  - `<logLevel>`: (optional) The logging level (e.g., "info", "debug", "warn", "error").

### Running a Relay

```bash  
go run cmd/relay/main.go -id=<id> -host=<host> -port=<port> -promPort=<promPort> -logLevel=<logLevel>
```  
- Options:
  - `<id>`: The unique identifier for the relay.
  - `<host>`: (optional) The public host IP for the relay. If not given, the public IP will be retrieved automatically.
  - `<port>`: The port number for the relay.
  - `<promPort>`: The port number for scraping the relay's Prometheus metrics.
  - `<logLevel>`: (optional) The logging level (e.g., "info", "debug", "warn", "error").

### Running a Client

```bash  
go run cmd/client/main.go -id=<id> -host=<host> -port=<port> -promPort=<promPort> -logLevel=<logLevel>
```  
- Options:
  - `<id>`: The unique identifier for the client.
  - `<host>`: (optional) The public host IP for the client. If not given, the public IP will be retrieved automatically.
  - `<port>`: The port number for the client.
  - `<promPort>`: The port number for scraping the client's Prometheus metrics.
  - `<logLevel>`: (optional) The logging level (e.g., "info", "debug", "warn", "error").

## Endpoints

### Bulletin Board

- **Register Client**: `POST /registerClient`
- **Register Relay**: `POST /registerRelay`
- **Updating config with new parameters (for next run)**: `POST /updateConfig`

### Relay & Client

- **Receive Onion**: `POST /receive`
- **Get Status**: `GET /status`
- **Start Run**: `POST /start`
- **Prometheus Metrics**: `GET /metrics` - Note that this is served on a different port

---

### References

- <a name="ALU24"></a>[\[ALU24\]](https://ia.cr/2024/885) - Ando M, Lysyanskaya A, Upfal E. Bruisable Onions: Anonymous Communication in the 
Asynchronous Model. _Cryptology ePrint Archive_. 2024.
  - [Link to PDF](https://eprint.iacr.org/2024/885.pdf)
- <a name="TGL-17"></a>[\[TGL+17\]](https://doi.org/10.1145/3132747.3132783) - Nirvan Tyagi, Yossi Gilad, Derek Leung, 
  Matei Zaharia, and Nickolai Zeldovich. Stadium: A distributed metadata-private messaging system. In Proceedings of the 26th
  Symposium on Operating Systems Principles, Shanghai, China, October 28-31, 2017, pages 423–440. ACM, 2017.
  - [Link to PDF](https://dl.acm.org/doi/pdf/10.1145/3132747.3132783)
- <a name="vdHLZZ15"></a>[\[vdHLZZ15\]](https://doi.org/10.1145/2815400.2815417) - Jelle van den Hooff, David Lazar, Matei Zaharia, and Nickolai Zeldovich. Vuvuzela: scalable private 
  messaging resistant to traffic analysis. In Ethan L. Miller and Steven Hand, editors, Proceedings of the 25th Symposium 
  on Operating Systems Principles, SOSP 2015, Monterey, CA, USA, October 4-7, 2015, pages 137–152. ACM, 2015.
  - [Link to PDF](https://dl.acm.org/doi/pdf/10.1145/2815400.2815417)


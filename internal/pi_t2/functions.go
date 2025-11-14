package pi_t2

import (
	"crypto"
	"encoding/json"

	pl "github.com/HannahMarsh/PrettyLogger"
)

const (
	MIXER           = "mixer"
	GATEKEEPER      = "gatekeeper"
	LAST_GATEKEEPER = "lastGatekeeper"
	RECIPIENT       = "recipient"
)

// TulipOnion represents a single onion at some stage.
type TulipOnion struct {
	Header  []byte   // H_i
	Content []byte   // C_i
	Sepal   [][]byte // S_i: array of sepal blocks
}

// ============================
// Header
// ============================

type Header struct {
	E []byte // encryption of E_i = (I, y_i, k_i) under public key of the I-th hop
	B []byte // encryption of B_i = (next hop, H_{I+1}) under k_i
}

type E struct { // E_i
	I    int    // hop index
	Role string // mixer, gatekepper, lastGatekeeper
	Y    []byte // y_i: meta data for the ith party
	K    []byte // k_i: layer key for layer I
}

type B struct { // B_i: encrypted under k_i
	NextHop []byte // next hop ip address
	H       []byte // header for next onion layer
}

func buildHeader(
	i int,
	y []byte,
	layerKey []byte,
	publicKey crypto.PublicKey,
	role string,
	nextHop []byte,
	nextHeader []byte) ([]byte, error) {
	E_i := E{
		I:    i,
		Y:    y,
		Role: role,
		K:    layerKey,
	}
	if role == LAST_GATEKEEPER {
		E_i.K = nil // master key (layer key) is found in sepal when processing
	}

	E_i_bytes, err := json.Marshal(E_i)
	if err != nil {
		return nil, pl.WrapError(err, "marshaling E_%d", i)
	}

	E_i_encrypted, err := hybridEncrypt(publicKey, E_i_bytes)
	if err != nil {
		return nil, pl.WrapError(err, "encrypting E_%d", i)
	}

	B_i := B{
		NextHop: nextHop,
		H:       nextHeader,
	}
	B_i_bytes, err := json.Marshal(B_i)
	if err != nil {
		return nil, pl.WrapError(err, "marshaling B_%d", i)
	}

	B_i_encrypted, err := encryptSymmetric(B_i_bytes, layerKey)
	if err != nil {
		return nil, pl.WrapError(err, "encrypting B_%d", i)
	}

	H_i := Header{
		E: E_i_encrypted,
		B: B_i_encrypted,
	}
	H_i_bytes, err := json.Marshal(H_i)
	if err != nil {
		return nil, pl.WrapError(err, "marshaling H_%d", i)
	}

	return H_i_bytes, nil
}

type Hop struct {
	PublicKey  crypto.PublicKey
	PrivateKey crypto.PrivateKey // for testing
	Role       string
	Address    string
}

func MakePath(pks []crypto.PublicKey, addresses []string, numMixers int, numGatekeepers int) ([]Hop, error) {
	n := numMixers + numGatekeepers + 1

	if numMixers <= 0 || numGatekeepers <= 0 {
		return nil, pl.NewError("Must be at least 1 mixer and at least one gatekeeper. ell_1 = %d, ell_2 = %d.", numMixers, numGatekeepers)
	}
	if len(pks) != n {
		return nil, pl.NewError("Given %d public keys, but needed %d", len(pks), n)
	}
	if len(addresses) != numMixers+numGatekeepers+1 {
		return nil, pl.NewError("Given %d addresses, but needed %d", len(addresses), n)
	}

	hops := make([]Hop, n)

	for i := 0; i < n; i++ {
		var role string
		switch {
		case i < numMixers:
			role = MIXER
		case i < n-2:
			role = GATEKEEPER
		case i == n-2:
			role = LAST_GATEKEEPER
		case i == n-1:
			role = RECIPIENT
		}
		hops[i] = Hop{
			PublicKey:  pks[i],
			PrivateKey: nil,
			Address:    addresses[i],
			Role:       role,
		}
	}
	return hops, nil
}

func FormOnion(msg []byte, pks []crypto.PublicKey, addresses []string, numMixers int, numGatekeepers int, d int) (*TulipOnion, error) {
	path, err := MakePath(pks, addresses, numMixers, numGatekeepers)
	if err != nil {
		return nil, pl.WrapError(err, "making path")
	}
	if d > numMixers {
		return nil, pl.NewError("Threshold (d=%d) needs to be no greater than numMixers (ell_1 = %d)", d, numMixers)
	}
	if o, err := formOnion(msg, path, d); err != nil {
		return nil, pl.WrapError(err, "forming onion")
	} else {
		return o, nil
	}
}

func formOnion(msg []byte, path []Hop, d int) (*TulipOnion, error) {
	L := len(path)
	layerKeys := make([][]byte, L-1)
	for i := 0; i < len(layerKeys); i++ {
		layerKeys[i] = randomKey()
	}
	masterKey := layerKeys[len(layerKeys)-1]

	content, err := hybridEncrypt(path[L-1].PublicKey, msg) // encrypt message under public key of final recipient
	if err != nil {
		return nil, pl.WrapError(err, "encrypting content")
	}

	// last gatekeeper
	header, err := buildHeader(L-2, nil, masterKey, path[L-2].PublicKey, LAST_GATEKEEPER, []byte(path[L-1].Address), nil) // for last gatekeeper
	if err != nil {
		return nil, pl.WrapError(err, "building header for last gatekeeper")
	}

	content, err = encryptSymmetric(content, masterKey)
	if err != nil {
		return nil, pl.WrapError(err, "encrypting content for last gatekeeper")
	}

	// wrap in encryption under masterKey
	sepal := make([][]byte, L-2)
	// inner-most layer of sepal containing key blocks and dummy blocks
	for i := 0; i < len(sepal); i++ {
		if i < d {
			// encrypt master key
			if sepal[i], err = hybridEncrypt(path[L-2].PublicKey, masterKey); err != nil {
				return nil, pl.WrapError(err, "encrypting sepal, master key")
			}
		} else {
			dummy := make([]byte, 32) // dummy blocks
			if sepal[i], err = hybridEncrypt(path[L-2].PublicKey, dummy); err != nil {
				return nil, pl.WrapError(err, "encrypting sepal, dummy block")
			}
		}
	}

	for i := L - 3; i >= 0; i-- {
		y := make([]byte, 32) // metadata

		header, err = buildHeader(i, y, layerKeys[i], path[i].PublicKey, path[i].Role, []byte(path[i+1].Address), header)
		if err != nil {
			return nil, pl.WrapError(err, "building header for %s, layer %d", path[i].Role, i)
		}
		content, err = encryptSymmetric(content, layerKeys[i]) // wrap in encryption under layer key
		if err != nil {
			return nil, pl.WrapError(err, "encrypting content for %s, layer %d", path[i].Role, i)
		}

		// wrap sepal in layer of encryption
		for j := 0; j < len(sepal); j++ {
			sepal[j], err = encryptSymmetric(sepal[j], layerKeys[i])
			if err != nil {
				return nil, pl.WrapError(err, "encrypting sepal for layer %d block %d", i, j)
			}
		}
	}

	onion := &TulipOnion{
		Header:  header,
		Content: content,
		Sepal:   sepal,
	}
	return onion, nil
}

type PeeledOnion struct {
	HopIndex  int    // hop index
	Metadata  []byte // y_i: meta data for the ith party
	NextHop   string // next hop ip address
	NextOnion *TulipOnion
	Content   []byte // for final recipient
	Role      string
}

func peelOnion(onion *TulipOnion, privateKey crypto.PrivateKey, bruise bool) (*PeeledOnion, error) {
	if onion.Header == nil { // final recipient
		content, err := hybridDecrypt(privateKey, onion.Content)
		if err != nil {
			return nil, pl.WrapError(err, "final recipient, content did not decrypt")
		}

		return &PeeledOnion{
			Content: content,
			Role:    RECIPIENT,
		}, nil
	}
	var header Header
	if err := json.Unmarshal(onion.Header, &header); err != nil {
		return nil, pl.WrapError(err, "unmarshaling header")
	}

	E_i_bytes, err := hybridDecrypt(privateKey, header.E)
	if err != nil {
		return nil, pl.WrapError(err, "decrypting E")
	}

	var E_i E
	if err = json.Unmarshal(E_i_bytes, &E_i); err != nil {
		return nil, pl.WrapError(err, "unmarshaling E after decrypting")
	}

	if bruise && E_i.Role != MIXER {
		return nil, pl.NewError("Only mixers can bruise onions. You are a %s", E_i.Role)
	}

	layerKey := E_i.K

	if E_i.Role == LAST_GATEKEEPER {
		layerKey = onion.Sepal[0]
		if layerKey, err = hybridDecrypt(privateKey, layerKey); err != nil {
			return nil, pl.WrapError(err, "Onion unrecoverable, bruised too many times but not dropped")
		}
	}

	B_i_bytes, err := decryptSymmetric(header.B, layerKey)
	if err != nil {
		return nil, pl.WrapError(err, "decrypting B")
	}

	var B_i B
	if err = json.Unmarshal(B_i_bytes, &B_i); err != nil {
		return nil, pl.WrapError(err, "unmarshaling B_after decrypting")
	}

	content, err := decryptSymmetric(onion.Content, layerKey)
	if err != nil {
		return nil, pl.WrapError(err, "decrypting content")
	}

	var sepal [][]byte = nil

	if E_i.Role != LAST_GATEKEEPER {
		sepal = make([][]byte, len(onion.Sepal))

		// remove layer of encryption from each sepal block
		for i := 0; i < len(sepal); i++ {
			if sepal[i], err = decryptSymmetric(onion.Sepal[i], layerKey); err != nil {
				return nil, pl.WrapError(err, "decrypting sepal as %s, block %d", E_i.Role, i)
			}
		}

		if E_i.Role == GATEKEEPER {
			// todo: check onion is valid and that not too bruised

		} else if E_i.Role == MIXER {
			if bruise {
				sepal = sepal[1:] // drop left most block (key block)
			} else {
				sepal = sepal[:len(sepal)-1] // drop right most block (dummy block)
			}
		}
	}

	return &PeeledOnion{
		HopIndex: E_i.I,
		Metadata: E_i.Y,
		NextHop:  string(B_i.NextHop),
		NextOnion: &TulipOnion{
			Header:  B_i.H,
			Content: content,
			Sepal:   sepal,
		},
		Role:    E_i.Role,
		Content: nil,
	}, nil
}

func PeelOnion(onion *TulipOnion, privateKey crypto.PrivateKey) (*PeeledOnion, error) {
	if o, err := peelOnion(onion, privateKey, false); err != nil {
		return nil, pl.WrapError(err, "peeling onion")
	} else {
		return o, nil
	}
}

func BruiseOnion(onion *TulipOnion, privateKey crypto.PrivateKey) (*PeeledOnion, error) {
	if o, err := peelOnion(onion, privateKey, true); err != nil {
		return nil, pl.WrapError(err, "bruising onion")
	} else {
		return o, nil
	}
}

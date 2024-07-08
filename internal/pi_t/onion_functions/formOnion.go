package onion_functions

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/keys"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"golang.org/x/exp/slog"
	"hash/fnv"
	"sort"
	"strings"
)

const fixedLegnthOfMessage = 256
const saltLength = 16

// OnionLayer represents each layer of the onion with encrypted content, header, and sepals.
type OnionLayer struct {
	Header  Header
	Content string
	Sepal   Sepal
}

type Sepal struct {
	Blocks []string
}
type Header struct {
	E string
	B []string
	A []string // verification hashes
}

type B_type struct {
	Address    string
	CypherText string
}

type CypherText struct {
	Tag       string
	Recipient string
	Layer     int
	Key       string
}

func (s Sepal) PeelSepal(layerKey []byte, addBruise bool, d int) (peeledSepal Sepal, err error) {

	peeledSepal = Sepal{Blocks: make([]string, len(s.Blocks))}

	// first decrypt all non-dropped blocks with the layer key
	for j, sepalBlock := range s.Blocks {
		_, decryptedString, err := keys.DecryptStringWithAES(layerKey, sepalBlock)
		if err != nil {
			return Sepal{}, pl.WrapError(err, "failed to decrypt sepal block")
		} else {
			peeledSepal.Blocks[j] = decryptedString
		}
	}
	if addBruise { // "drop" left-most sepal block that hasn't already been bruised
		//slog.Info("Dropping left-most sepal block")
		peeledSepal.Blocks = peeledSepal.Blocks[1:]
	} else { // "drop" right-most sepal block that hasn't already been dropped
		//slog.Info("Dropping right-most sepal block")
		peeledSepal.Blocks = peeledSepal.Blocks[:len(peeledSepal.Blocks)-1]
	}
	return peeledSepal, nil
}

func formSepal(masterKey string, d int, layerKeys [][]byte, l int, l1 int) (sepal Sepal, A [][]string, err error) {
	keyBlocks, err := formKeyBlocks(masterKey, d, layerKeys[:l], l1) // salted and encrypted under k_{1}...k_{l-1}
	if err != nil {
		return Sepal{}, nil, pl.WrapError(err, "failed to construct key blocks")
	}
	nullBlocks, err := formKeyBlocks("null", l1-d+1, layerKeys[:l1+2], l1) // salted and encrypted under k_{1}...k_{l1+1}
	if err != nil {
		return Sepal{}, nil, pl.WrapError(err, "failed to construct null blocks")
	}
	T := make([][]string, l+3) //append(keyBlocks, nullBlocks...)
	for i := 1; i < l+3; i++ {
		if i < len(keyBlocks) {
			if i < len(nullBlocks) {
				T[i] = append(keyBlocks[i], nullBlocks[i][1:]...)
			} else {
				T[i] = keyBlocks[i]
			}
		} else if i < len(nullBlocks) {
			T[i] = nullBlocks[i]
		}
	}

	A = generateAllPossibleHashes(l1, d, T)

	blocks := T[1][1:]
	sepal = Sepal{Blocks: blocks}

	return sepal, A, nil
}

func generateAllPossibleHashes(l1 int, d int, T [][]string) [][]string {
	//A := make([][][]string, l1+1)
	hashes := make([][]string, l1+1)
	for i := range hashes {
		//A[i] = make([][]string, 0)
		hashes[i] = make([]string, 0)
	}
	possibleBruises := utils.GenerateUniquePermutations(d, l1-d)
	for numBruises := 0; numBruises <= d; numBruises++ {
		possibleBruises = append(possibleBruises, utils.GenerateUniquePermutations(numBruises, l1-numBruises)...)
	}
	for i := range possibleBruises {
		num := 0
		for j := range possibleBruises[i] {
			if possibleBruises[i][j] {
				num++
			}
			if num == 3 && j < len(possibleBruises[i])-1 {
				possibleBruises[i] = possibleBruises[i][:j+1] // onion wont be processed after gatekeeper drops it
				break
			}
		}
	}
	//A[0] = [][]string{T[1]}
	hashes[0] = []string{hash(strings.Join(T[1], ""))}

	for _, possibility := range possibleBruises {
		numBruises := 0
		numNonBruises := 0
		for i, doBruise := range possibility {
			if doBruise {
				numBruises++
			} else {
				numNonBruises++
			}
			s := utils.Copy(T[i+2])
			s = utils.DropFromLeft(s, numBruises)
			s = utils.DropFromRight(s, numNonBruises)

			h := hash(strings.Join(s, ""))
			if !utils.Contains(hashes[i+1], func(str string) bool {
				return str == h
			}) {
				hashes[i+1] = append(hashes[i+1], h)
				//A[i+1] = append(A[i+1], s)
			}
		}
	}
	return utils.Map(hashes, func(h []string) []string {
		sort.Strings(h)
		return h
	})
}

// T[i][j] is the jth sepal block without the i - 1 outer encryption layers.
func formKeyBlocks(wrappedValue string, numBlocks int, layerKeys [][]byte, l1 int) (T [][]string, err error) {
	T = make([][]string, len(layerKeys)+1)
	for i := range T {
		T[i] = make([]string, numBlocks+1)
	}

	for j := 1; j <= numBlocks; j++ {
		value := wrappedValue
		T[len(layerKeys)][j] = wrappedValue

		for i := len(layerKeys) - 1; i >= 1; i-- {
			k := layerKeys[i]
			saltedValue := value //, err := saltEncodedValue(value, saltLength)
			if err != nil {
				return nil, pl.WrapError(err, "failed to salt value")
			}
			_, value, err = keys.EncryptStringWithAES(k, saltedValue)
			if err != nil {
				return nil, pl.WrapError(err, "failed to encrypt inner block")
			}
			T[i][j] = value
		}
	}
	return T, nil
}

// FormOnion creates a forward onion from a message m, a path P, public keys pk, and metadata y.
// Parameters:
// - m: a fixed length message
// - P: a routing path (sequence of addresses representing l1 mixers and l2 gatekeepers such that len(P) = l1 + l2 + 1)
// - l1: the number of mixers in the routing path
// - l2: the number of gatekeepers in the routing path
// - pk: a list of public keys for the entities in the routing path
// - y: metadata associated with each entity (except the last destination entity) in the routing path
// Returns:
// - A list of lists of onions, O = (O_1, ..., O_l), where each O_i contains all possible variations of the i-th onion layer.
//   - The first list O_1 contains just the onion for the first mixer.
//   - For 2 <= i <= l1, the list O_i contains i options, O_i = (O_i,0, ..., O_i,i-1), each O_i,j representing the i-th onion layer with j prior bruises.
//   - For l1 + 1 <= i <= l1 + l2, the list O_i contains l1 + 1 options, depending on the total bruising from the mixers.
//   - The last list O_(l1 + l2 + 1) contains just the innermost onion for the recipient.
func FORMONION(publicKey, privateKey, m string, mixers []string, gatekeepers []string, recipient string, publicKeys []string, metadata []string, d int) ([]OnionLayer, error) {

	message := padMessage(m)

	path := append(append(append([]string{""}, mixers...), gatekeepers...), recipient)
	l1 := len(mixers)
	l2 := len(gatekeepers)
	l := l1 + l2 + 1

	// Generate keys for each layer and the master key
	layerKeys := make([][]byte, l+1)
	for i := range layerKeys {
		layerKey, _ := keys.GenerateSymmetricKey()
		layerKeys[i] = layerKey //base64.StdEncoding.EncodeToString(layerKey)
	}
	K, _ := keys.GenerateSymmetricKey()
	masterKey := base64.StdEncoding.EncodeToString(K)

	// Initialize the onion structure
	onionLayers := make([]OnionLayer, l+1)

	// Construct first sepal for M1
	sepal, A, err := formSepal(masterKey, d, layerKeys, l, l1)
	if err != nil {
		return nil, pl.WrapError(err, "failed to create sepal")
	}

	// build penultimate onion layer

	// form content
	C, err := formContent(layerKeys, l, message, K)
	if err != nil {
		return nil, pl.WrapError(err, "failed to form content")
	}

	// tag array
	tags := make([]string, l+1)
	tags[l] = hash(C[l])

	// ciphertext array
	E := make([]string, l+1)
	E[l], err = Enc(privateKey, publicKeys[l-1], tags[l], recipient, l, layerKeys[l])
	if err != nil {
		return nil, pl.WrapError(err, "failed to encrypt ciphertext")
	}

	// header array
	H := make([]Header, l+1)
	H[l] = Header{
		E: E[l],
		B: []string{},
	}

	B := make([][]string, l+1)
	for i, _ := range B {
		B[i] = make([]string, l+1)
	}
	B[l-1][1], err = EncryptB(recipient, E[l], K)
	if err != nil {
		return nil, pl.WrapError(err, "failed to encrypt B_l_minus_1_1")
	}

	onionLayers[l] = OnionLayer{
		Header:  H[l],
		Content: C[l],
		Sepal: Sepal{
			Blocks: []string{},
		},
	}

	for i := l - 1; i >= 1; i-- {
		B[i][1], err = EncryptB(path[i+1], E[i+1], layerKeys[i])
		if err != nil {
			return nil, pl.WrapError(err, "failed to encrypt B_i_1")
		}
		for j := 2; j <= l-j+1; j++ {
			B[i][j], err = EncryptB("", B[i+1][j-1], layerKeys[i])
		}
		B_i_1_to_C_i := append(B[i][1:], C[i])
		concat := strings.Join(B_i_1_to_C_i, "")
		tags[i] = hash(concat)
		role := "mixer"
		if i == l-1 {
			role = "lastGatekeeper"
		} else if i > l1 {
			role = "gatekeeper"
		}
		E[i], err = Enc(privateKey, publicKeys[i-1], tags[i], role, i, layerKeys[i]) // TODO add y_i, A_i
		if i < len(A) {
			H[i] = Header{
				E: E[i],
				B: B[i],
				A: A[i-1],
			}
		} else {
			H[i] = Header{
				E: E[i],
				B: B[i],
			}
		}

		onionLayers[i] = OnionLayer{
			Header:  H[i],
			Content: C[i],
			Sepal: Sepal{
				Blocks: []string{},
			},
		}
	}

	onionLayers[1].Sepal = sepal

	return onionLayers[1:], nil
}

func formContent(layerKeys [][]byte, l int, message []byte, K []byte) ([]string, error) {
	C_arr := make([]string, l+1)
	C_l, _, err := keys.EncryptWithAES(layerKeys[l], message)
	if err != nil {
		return nil, pl.WrapError(err, "failed to encrypt C_l")
	}
	_, C_arr[l], err = keys.EncryptWithAES(K, C_l)
	if err != nil {
		return nil, pl.WrapError(err, "failed to encrypt C_l_minus_1")
	}
	for i := l - 1; i >= 1; i-- {
		_, C_arr[i], err = keys.EncryptStringWithAES(layerKeys[i], C_arr[i+1])
		if err != nil {
			return nil, pl.WrapError(err, "failed to encrypt C_i")
		}
	}
	return C_arr, nil
}

func EncryptB(address string, E string, layerKey []byte) (string, error) {
	b, err := json.Marshal(B_type{
		Address:    address,
		CypherText: E,
	})
	if err != nil {
		return "", pl.WrapError(err, "failed to marshal b")
	}
	_, bEncrypted, err := keys.EncryptWithAES(layerKey, b)
	return bEncrypted, nil
}

func Enc(privateKey, publicKey string, tag string, role string, layer int, layerKey []byte) (string, error) {
	sharedKey, err := keys.ComputeSharedKey(privateKey, publicKey)
	if err != nil {
		return "", pl.WrapError(err, "failed to compute shared key")
	}
	ciphertext := CypherText{
		Tag:       tag,
		Recipient: role,
		Layer:     layer,
		Key:       base64.StdEncoding.EncodeToString(layerKey),
	}
	cypherBytes, err := json.Marshal(ciphertext)
	if err != nil {
		return "", pl.WrapError(err, "failed to marshal ciphertext")
	}

	_, E_l, err := keys.EncryptWithAES(sharedKey, cypherBytes)
	if err != nil {
		return "", pl.WrapError(err, "failed to encrypt ciphertext")
	}
	return E_l, nil
}

func generateSaltSpace(length int) []byte {
	space := make([]byte, length)
	_, err := rand.Read(space)
	if err != nil {
		panic(err)
	}
	return space
}

//func randomStringWithSameLength(str string) (string, error) {
//	bytes, err := base64.StdEncoding.DecodeString(str)
//	if err != nil {
//		return "", pl.WrapError(err, "failed to decode string")
//	}
//	randomBytes := make([]byte, len(bytes))
//	for i := range randomBytes {
//		_, err := rand.Read(randomBytes[i : i+1])
//		if err != nil {
//			return "", pl.WrapError(err, "failed to generate random data")
//		}
//	}
//}

func generateEncodedSaltSpace(length int) string {
	space := generateSaltSpace(length)
	return base64.StdEncoding.EncodeToString(space)
}

func saltEncodedValue(value string, length int) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", pl.WrapError(err, "failed to decode value")
	}
	salt := generateSaltSpace(length)
	saltedValue := append(decoded, salt...)
	return base64.StdEncoding.EncodeToString(saltedValue), nil
}

func hash(s string) string {
	h := fnv.New32a()
	_, err := h.Write([]byte(s))
	if err != nil {
		slog.Error("failed to hash string", err)
		return ""
	}
	return fmt.Sprint(h.Sum32())
}

//// formSepal creates the sepal blocks for the onion layers.
//func formSepal(d, l1 int, masterKey []byte, pubKeys [][]byte) ([][]byte, error) {
//	sepals := make([][]byte, l1+1)
//	salt := make([]byte, 16)
//	for i := 0; i < l1+1; i++ {
//		if _, err := io.ReadFull(rand.Reader, salt); err != nil {
//			panic(err)
//		}
//		if i < d {
//			// Sepal block contains the master key
//			sepals[i] = masterKey
//		} else {
//			// Sepal block contains a dummy value
//			sepals[i] = make([]byte, 32)
//		}
//		for j := 0; j < len(pubKeys); j++ {
//			cipherText, err := keys.EncryptWithAES(pubKeys[j], append(sepals[i], salt...))
//			if err != nil {
//				return nil, pl.WrapError(err, "failed to encrypt sepal block")
//			}
//			sepals[i], err = base64.StdEncoding.DecodeString(cipherText)
//			if err != nil {
//				return nil, pl.WrapError(err, "failed to decode sepal block")
//			}
//		}
//	}
//	return sepals, nil
//}
//
//// FormOnion creates a forward onion from a message m, a path P, public keys pk, and metadata y.
//// Parameters:
//// - m: a fixed length message
//// - P: a routing path (sequence of addresses representing l1 mixers and l2 gatekeepers such that len(P) = l1 + l2 + 1)
//// - l1: the number of mixers in the routing path
//// - l2: the number of gatekeepers in the routing path
//// - pk: a list of public keys for the entities in the routing path
//// - y: metadata associated with each entity (except the last destination entity) in the routing path
//// Returns:
//// - A list of lists of onions, O = (O_1, ..., O_l), where each O_i contains all possible variations of the i-th onion layer.
////   - The first list O_1 contains just the onion for the first mixer.
////   - For 2 <= i <= l1, the list O_i contains i options, O_i = (O_i,0, ..., O_i,i-1), each O_i,j representing the i-th onion layer with j prior bruises.
////   - For l1 + 1 <= i <= l1 + l2, the list O_i contains l1 + 1 options, depending on the total bruising from the mixers.
////   - The last list O_(l1 + l2 + 1) contains just the innermost onion for the recipient.
//func FormOnion(m string, P []string, l1, l2 int, pk []string, y []string) (O [][]OnionLayer, err error) {
//	paddedMessage := padMessage(m)
//
//	// Convert public keys and metadata to byte slices
//	publicKeys := make([][]byte, len(pk))
//	metadata := make([][]byte, len(y))
//	for i := range publicKeys {
//		publicKeys[i] = []byte(pk[i])
//	}
//	for i := range metadata {
//		metadata[i] = []byte(y[i])
//	}
//
//	// Generate keys for each layer and the master key
//	layerKeys := make([][]byte, l1+l2+1)
//	for i := range layerKeys {
//		layerKeys[i], _ = keys.GenerateSymmetricKey()
//	}
//	masterKey, _ := keys.GenerateSymmetricKey()
//
//	// Initialize the onion structure
//	onionLayers := make([][]OnionLayer, l1+l2+1)
//
//	// Create the first sepal
//	sepals, err := formSepal(l1, l1, masterKey, layerKeys)
//	if err != nil {
//		return nil, pl.WrapError(err, "failed to create sepal")
//	}
//
//	// Create the first onion layer with all variations for the first mixer
//	onionLayers[0] = make([]OnionLayer, 1)
//	header, _ := keys.EncryptWithAES(publicKeys[0], []byte(fmt.Sprintf("Mixer %d", 1)))
//	content, _ := keys.EncryptWithAES(layerKeys[0], paddedMessage)
//	onionLayers[0][0] = OnionLayer{
//		Content: content,
//		Header:  header,
//		Sepal:   sepals,
//	}
//
//	// Create layers for remaining mixers
//	for i := 1; i <= l1; i++ {
//		variations := i + 1
//		onionLayers[i] = make([]OnionLayer, variations)
//		for j := 0; j < variations; j++ {
//			sepalCopy := make([][]byte, len(sepals)-1)
//			copy(sepalCopy, sepals[1:])
//			header, _ := keys.EncryptWithAES(publicKeys[i], []byte(fmt.Sprintf("Mixer %d", i+1)))
//			content, _ := keys.EncryptWithAES(layerKeys[i], []byte(onionLayers[i-1][j].Content))
//			onionLayers[i][j] = OnionLayer{
//				Content: content,
//				Header:  header,
//				Sepal:   sepalCopy,
//			}
//		}
//	}
//
//	// Create layers for gatekeepers
//	for i := l1 + 1; i <= l1+l2; i++ {
//		variations := l1 + 1
//		onionLayers[i] = make([]OnionLayer, variations)
//		for j := 0; j < variations; j++ {
//			header, _ := keys.EncryptWithAES(publicKeys[i], []byte(fmt.Sprintf("Gatekeeper %d", i-l1)))
//			content, _ := keys.EncryptWithAES(layerKeys[i], []byte(onionLayers[i-1][j].Content))
//			onionLayers[i][j] = OnionLayer{
//				Content: content,
//				Header:  header,
//				Sepal:   sepals[:len(sepals)-1],
//			}
//		}
//	}
//
//	// The last layer for the recipient
//	onionLayers[l1+l2] = make([]OnionLayer, 1)
//	header, _ = keys.EncryptWithAES(publicKeys[len(publicKeys)-1], []byte("Recipient"))
//	content, _ = keys.EncryptWithAES(layerKeys[l1+l2], paddedMessage)
//	onionLayers[l1+l2][0] = OnionLayer{
//		Content: content,
//		Header:  header,
//		Sepal:   sepals[:1],
//	}
//
//	return onionLayers
//
//}

func padMessage(message string) []byte {
	var nullTerminator byte = '\000'
	var paddedMessage = make([]byte, fixedLegnthOfMessage)
	var mLength = len(message)

	for i := 0; i < fixedLegnthOfMessage; i++ {
		if i >= mLength || i == fixedLegnthOfMessage-1 {
			paddedMessage[i] = nullTerminator
		} else {
			paddedMessage[i] = message[i]
		}
	}
	return paddedMessage
}

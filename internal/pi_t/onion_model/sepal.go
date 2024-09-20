package onion_model

import (
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/crypto/keys"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"strings"
)

// Sepal of an onion handles layered encryption of blocks.
type Sepal struct {
	Blocks []string // List of encrypted blocks in the sepal.
}

// PeelSepal decrypts the blocks in the sepal using the provided layer key.
func (s Sepal) PeelSepal(layerKey []byte) (peeledSepal Sepal, err error) {

	peeledSepal = Sepal{Blocks: make([]string, len(s.Blocks))} // Initialize a new sepal for the peeled blocks.

	// Decrypt all non-dropped blocks with the layer key.
	for j, sepalBlock := range s.Blocks {
		if sepalBlock == "" || sepalBlock == "null" {
			peeledSepal.Blocks[j] = sepalBlock // Retain empty or null blocks as is.
			continue
		}
		_, decryptedString, err := keys.DecryptStringWithAES(layerKey, sepalBlock)
		if err != nil {
			return Sepal{}, pl.WrapError(err, "failed to decrypt sepal block")
		} else {
			peeledSepal.Blocks[j] = decryptedString // Store the decrypted block.
		}
	}
	return peeledSepal, nil // Return the peeled sepal.
}

// AddBruise "drops" the left-most sepal block, simulating the addition of a bruise.
func (s Sepal) AddBruise() Sepal {
	return Sepal{
		Blocks: utils.DropFirstElement(s.Blocks),
	}
}

// RemoveBlock "drops" the right-most sepal block.
func (s Sepal) RemoveBlock() Sepal {
	return Sepal{
		Blocks: utils.DropLastElement(s.Blocks),
	}
}

// FormSepals generates the sepals for each layer of the onion.
func FormSepals(masterKey string, d int, layerKeys [][]byte, l int, l1 int, l2 int, hash func(string) string) (A [][]string, S_i [][]Sepal, err error) {
	// Generate key blocks for the onion using the master key.
	keyBlocks, err := formKeyBlocks(masterKey, d, layerKeys[:l])
	if err != nil {
		return nil, nil, pl.WrapError(err, "failed to construct key blocks")
	}
	// Generate null blocks for the onion.
	nullBlocks, err := formKeyBlocks("null", l1-d+1, layerKeys[:l1+2])
	if err != nil {
		return nil, nil, pl.WrapError(err, "failed to construct null blocks")
	}
	// Combine key blocks and null blocks.
	T := make([][]string, l+3)
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

	// Generate all possible sepals for the onion.
	A, S_i = generateAllPossibleSepals(l1, l2, d, T, hash)
	return A, S_i, nil
}

// generateAllPossibleSepals generates all possible sepals for the onion, including those with bruises.
func generateAllPossibleSepals(l1 int, l2 int, d int, T [][]string, hash func(string) string) ([][]string, [][]Sepal) {
	type sepalWrapper struct {
		S    Sepal  // Sepal structure.
		Hash string // Hash of the sepal's content.
	}
	allPossibleSepals := make([][]sepalWrapper, l1+l2) // Initialize the list of possible sepals.

	// Generate possible bruise combinations.
	possibleBruises := make([][]bool, 0)
	for numBruises := 0; numBruises <= l1; numBruises++ {
		possibleBruises = append(possibleBruises, utils.GenerateUniquePermutations(numBruises, l1-numBruises)...)
	}

	bruiseCount := make([]int, 0)

	// Calculate possible sepals as received by mixers and the first gatekeeper.
	for _, possibility := range possibleBruises {
		numBruises := 0
		numNonBruises := 0
		for i, doBruise := range possibility {

			s := utils.Copy(T[i+1])
			s = utils.DropFromLeft(s, numBruises)
			s = utils.DropFromRight(s, numNonBruises)

			if doBruise {
				numBruises++
			} else {
				numNonBruises++
			}

			h := hash(strings.Join(s[1:], ""))
			if !utils.Contains(allPossibleSepals[i], func(sw sepalWrapper) bool {
				return sw.Hash == h
			}) {
				allPossibleSepals[i] = append(allPossibleSepals[i], sepalWrapper{
					S:    Sepal{Blocks: s[1:]},
					Hash: h,
				})
			}
			if i == len(possibility)-1 && utils.ContainsElement(bruiseCount, numBruises) == false {
				bruiseCount = append(bruiseCount, numBruises)
			}
		}
	}

	// Calculate sepals as received by the last l2 - 1 gatekeepers.
	for i := l1; i < len(allPossibleSepals); i++ {
		for _, numBruises := range bruiseCount {
			if numBruises < d {
				s := []string{T[i+1][numBruises+1]}
				h := hash(strings.Join(s, ""))
				allPossibleSepals[i] = append(allPossibleSepals[i], sepalWrapper{
					S:    Sepal{Blocks: s},
					Hash: h,
				})
			}
		}
	}

	// Sort the possible sepals by their hash values.
	sorted := utils.Map(allPossibleSepals, func(s []sepalWrapper) []sepalWrapper {
		utils.Sort(s, func(a, b sepalWrapper) bool {
			return a.Hash < b.Hash
		})
		return s
	})

	// Return the sorted sepals and their corresponding hashes.
	return utils.Map(sorted, func(s []sepalWrapper) []string {
			return utils.Map(s, func(sw sepalWrapper) string { return sw.Hash })
		}), utils.Map(sorted, func(s []sepalWrapper) []Sepal {
			return utils.Map(s, func(sw sepalWrapper) Sepal { return sw.S })
		})
}

// formKeyBlocks generates key blocks for each layer of the onion.
func formKeyBlocks(wrappedValue string, numBlocks int, layerKeys [][]byte) (T [][]string, err error) {
	T = make([][]string, len(layerKeys)+1) // Initialize the key blocks for each layer.
	for i := range T {
		T[i] = make([]string, numBlocks+1)
	}

	// Generate encrypted key blocks for each layer.
	for j := 1; j <= numBlocks; j++ {
		value := wrappedValue
		T[len(layerKeys)][j] = wrappedValue

		for i := len(layerKeys) - 1; i >= 1; i-- {
			k := layerKeys[i]
			saltedValue := value
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
	return T, nil // Return the generated key blocks.
}

package onion_model

import (
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/tools/keys"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"strings"
)

type Sepal struct {
	Blocks []string
}

func (s Sepal) PeelSepal(layerKey []byte) (peeledSepal Sepal, err error) {

	peeledSepal = Sepal{Blocks: make([]string, len(s.Blocks))}

	// first decrypt all non-dropped blocks with the layer key
	for j, sepalBlock := range s.Blocks {
		if sepalBlock == "" || sepalBlock == "null" {
			peeledSepal.Blocks[j] = sepalBlock
			continue
		}
		_, decryptedString, err := keys.DecryptStringWithAES(layerKey, sepalBlock)
		if err != nil {
			return Sepal{}, pl.WrapError(err, "failed to decrypt sepal block")
		} else {
			peeledSepal.Blocks[j] = decryptedString
		}
	}
	//if addBruise { // "drop" left-most sepal block that hasn't already been bruised
	//	//slog.Info("Dropping left-most sepal block")
	//	peeledSepal.Blocks = utils.DropFirstElement(peeledSepal.Blocks) //peeledSepal.Blocks[1:]
	//} else if len(peeledSepal.Blocks) > 1 { // "drop" right-most sepal block that hasn't already been dropped
	//	//slog.Info("Dropping right-most sepal block")
	//	peeledSepal.Blocks = utils.DropLastElement(peeledSepal.Blocks) // peeledSepal.Blocks[:len(peeledSepal.Blocks)-1]
	//} // else, this is a gatekeeper processing, so only one block left
	return peeledSepal, nil
}

func (s Sepal) AddBruise() {
	s.Blocks = utils.DropFirstElement(s.Blocks) //peeledSepal.Blocks[1:]
}

func (s Sepal) RemoveBlock() {
	s.Blocks = utils.DropLastElement(s.Blocks) //peeledSepal.Blocks[1:]
}

func FormSepals(masterKey string, d int, layerKeys [][]byte, l int, l1 int, l2 int, hash func(string) string) (A [][]string, S_i [][]Sepal, err error) {
	keyBlocks, err := formKeyBlocks(masterKey, d, layerKeys[:l], l1) // salted and encrypted under k_{1}...k_{l-1}
	if err != nil {
		return nil, nil, pl.WrapError(err, "failed to construct key blocks")
	}
	nullBlocks, err := formKeyBlocks("null", l1-d+1, layerKeys[:l1+2], l1) // salted and encrypted under k_{1}...k_{l1+1}
	if err != nil {
		return nil, nil, pl.WrapError(err, "failed to construct null blocks")
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

	A, S_i = generateAllPossibleSepals(l1, l2, d, T, hash)
	return A, S_i, nil
}

func generateAllPossibleSepals(l1 int, l2 int, d int, T [][]string, hash func(string) string) ([][]string, [][]Sepal) {
	type sepalWrapper struct {
		S    Sepal
		Hash string
	}
	allPossibleSepals := make([][]sepalWrapper, l1+l2)
	//hashes := make([][]string, l1+1)
	for i := 0; i < l1+l2; i++ {
		allPossibleSepals[i] = make([]sepalWrapper, 0)
		//hashes[i] = make([]string, 0)
	}
	possibleBruises := make([][]bool, 0) // utils.GenerateUniquePermutations(d, l1-d)
	for numBruises := 0; numBruises <= l1; numBruises++ {
		possibleBruises = append(possibleBruises, utils.GenerateUniquePermutations(numBruises, l1-numBruises)...)
	}
	//hashes[0] = []string{hash(strings.Join(T[1], ""))}
	//allPossibleSepals[0] = []sepalWrapper{{
	//	S:    Sepal{Blocks: T[1][1:]},
	//	Hash: hash(strings.Join(T[1][1:], "")),
	//}}

	// calulate possible sepals as received by the mixers and the first gatekeeper
	bruiseCount := make([]int, 0)

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
				//hashes[i+1] = append(hashes[i+1], h)
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

	// calculate sepals as received by the last l2 - 1 gatekeepers

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

	sorted := utils.Map(allPossibleSepals, func(s []sepalWrapper) []sepalWrapper {
		utils.Sort(s, func(a, b sepalWrapper) bool {
			return a.Hash < b.Hash
		})
		return s
	})
	return utils.Map(sorted, func(s []sepalWrapper) []string {
			return utils.Map(s, func(sw sepalWrapper) string { return sw.Hash })
		}), utils.Map(sorted, func(s []sepalWrapper) []Sepal {
			return utils.Map(s, func(sw sepalWrapper) Sepal { return sw.S })
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

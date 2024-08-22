package onion_model

// Onion represents the full onion structure used in the onion routing network.
type Onion struct {
	Header  Header  // Header of the onion, containing encryption-related metadata.
	Content Content // Content of the onion, which is encrypted.
	Sepal   Sepal   // Sepal of the onion, which handles layered encryption of blocks.
}

package models

// Onion represents an onion in the onion routing network
type Onion struct {
	ID     string
	Layers []string
}

// NewOnion creates a new onion
func NewOnion(id string, layers []string) *Onion {
	return &Onion{
		ID:     id,
		Layers: layers,
	}
}

// AddLayer adds a new layer to the onion
func (o *Onion) AddLayer(layer string) {
	o.Layers = append(o.Layers, layer)
}

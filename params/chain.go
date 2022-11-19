package params

// Chain is the blockchain chain configuration
type Chain struct {
	Name      string   `json:"name"`
	Genesis   *Genesis `json:"genesis"`
	Params    *Params  `json:"params"`
	Bootnodes []string `json:"bootnodes,omitempty"`
}

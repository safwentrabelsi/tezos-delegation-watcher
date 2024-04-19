package types

type Delegation struct {
	Id        int    `json:"-"`
	Timestamp string `json:"timestamp"`
	Amount    uint64 `json:"amount"`
	Delegator string `json:"delegator"`
	Block     uint64 `json:"block"`
}

// GetDelegationsResponse is the response from Tzkt api
// change name to RawDelegation
type FetchedDelegation struct {
	Level     uint64 `json:"level"`
	Timestamp string `json:"timestamp"`
	Sender    struct {
		Address string `json:"address"`
	} `json:"sender"`
	Amount uint64 `json:"amount"`
}

type ChanMsg struct {
	Level uint64
	Reorg bool
	Data  []FetchedDelegation
}

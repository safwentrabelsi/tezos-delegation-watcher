package types

type Delegation struct {
	Id        int    `json:"-"`
	Timestamp string `json:"timestamp"`
	Amount    uint64 `json:"amount"`
	Delegator string `json:"delegator"`
	Block     uint64 `json:"block"`
}

// Sender represents the sender of a delegation.
type Sender struct {
	Address string `json:"address"`
}

// FetchedDelegation is the response from Tzkt api
type FetchedDelegation struct {
	Level     uint64 `json:"level"`
	Timestamp string `json:"timestamp"`
	Sender    Sender `json:"sender"`
	Amount    uint64 `json:"amount"`
}

type ChanMsg struct {
	Level uint64
	Reorg bool
	Data  []FetchedDelegation
}

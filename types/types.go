package types

type Delegation struct {
	Id        int    `json:"-"`
	Timestamp string `json:"timestamp"`
	Amount    uint64 `json:"amount"`
	Delegator string `json:"delegator"`
	Block     uint64 `json:"block"`
}

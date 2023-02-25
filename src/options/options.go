package options

type ExchangeType int

const (
	LOCAL_EX ExchangeType = iota
	TCP_EX
)

type AddressPath struct {
	User     string
	Address  string
	Filepath string
}

type Options struct {
	ExType ExchangeType
	Source AddressPath
	Dest   AddressPath

	Verbose  bool
	IsServer bool
}

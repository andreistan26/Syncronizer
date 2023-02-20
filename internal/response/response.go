package response

import "fmt"

type ResponseType int

const (
	B_BLOCK ResponseType = iota
	A_BLOCK
)

type ResponsePacket struct {
	blockType ResponseType
	data      []byte
}

type Response []ResponsePacket

func (packet ResponsePacket) String() string {
	return fmt.Sprintf(
		"Block Type : %v \n "+
			"Data       : %v \n "+
			"Size       : %v \n ",
		packet.blockType, packet.data, len(packet.data),
	)
}

func (response Response) String() string {
	var responseStr string
	for idx, el := range response {
		responseStr += fmt.Sprintf(
			"\tPacket %v \n"+
				"%v\n",
			idx, el,
		)
	}
	return responseStr
}

func (resp ResponseType) String() string {
	switch resp {
	case A_BLOCK:
		return "A_BLOCK"
	default:
		return "B_BLOCK"
	}
}

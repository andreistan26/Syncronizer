package transport

// first request client ---> server
type InitialFileRequest struct {
	Filename string
	Md5sum   [16]byte
}

type PacketType int

const (
	INFO_PACKET PacketType = iota
	DATA_PACKET
	ERROR_PACKET
)

type StatusResponse int

const (
	STATUS_FILE_SYNCED StatusResponse = iota
	STATUS_FILE_EXISTS
	STATUS_REQUEST_CHUNKS
	STATUS_SENDING_CHUNKS
	STATUS_SERVER_ERROR
)

type StatusMessages struct {
	Status  StatusResponse
	Message string
}

// type SyncPacket interface {
// }

// type Packet struct {
// 	PacketType PacketType
// 	Payload    SyncPacket
// }

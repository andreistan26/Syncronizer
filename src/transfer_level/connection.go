package transport

import (
	"encoding/gob"
)

type SyncConn struct {
	Encoder *gob.Encoder
	Decoder *gob.Decoder
}

func (conn *SyncConn) Decode(e any) error {
	return conn.Decoder.Decode(e)
}

func (conn *SyncConn) Encode(e any) error {
	return conn.Encoder.Encode(e)
}

// func (conn *SyncConn) WritePacket(packet SyncPacket) error {
// 	return conn.Encode(packet)
// }

// func (conn *SyncConn) ReadPacket() (SyncPacket, error) {
// 	var packet *SyncPacket
// 	err := conn.Decode(packet)
// 	return packet, err
// }

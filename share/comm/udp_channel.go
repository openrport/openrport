package comm

import (
	"encoding/gob"
	"io"
	"net"
)

type UDPChannel struct {
	enc *gob.Encoder
	dec *gob.Decoder
}

type UDPMessage struct {
	Addr *net.UDPAddr
	Data []byte
}

func NewUDPChannel(rw io.ReadWriter) *UDPChannel {
	return &UDPChannel{
		enc: gob.NewEncoder(rw),
		dec: gob.NewDecoder(rw),
	}
}

func (c *UDPChannel) Encode(addr *net.UDPAddr, data []byte) error {
	return c.enc.Encode(UDPMessage{
		Addr: addr,
		Data: data,
	})
}

func (c *UDPChannel) Decode() (*net.UDPAddr, []byte, error) {
	var msg UDPMessage
	err := c.dec.Decode(&msg)
	if err != nil {
		return nil, nil, err
	}

	return msg.Addr, msg.Data, nil
}

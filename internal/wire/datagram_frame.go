package wire

import (
	"bytes"
	"io"

	"github.com/lucas-clemente/quic-go/internal/protocol"
	"github.com/lucas-clemente/quic-go/internal/utils"
)

// A DatagramFrame is a DATAGRAM frame
type DatagramFrame struct {
	FlowID         uint64
	DataLenPresent bool
	Data           []byte
}

func parseDatagramFrame(r *bytes.Reader, _ protocol.VersionNumber) (*DatagramFrame, error) {
	typeByte, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	f := &DatagramFrame{}
	f.DataLenPresent = typeByte&0x1 > 0

	if typeByte&0x2 > 0 {
		fid, err := utils.ReadVarInt(r)
		if err != nil {
			return nil, err
		}
		f.FlowID = fid
	}

	var length uint64
	if f.DataLenPresent {
		var err error
		len, err := utils.ReadVarInt(r)
		if err != nil {
			return nil, err
		}
		if len > uint64(r.Len()) {
			return nil, io.EOF
		}
		length = len
	} else {
		length = uint64(r.Len())
	}
	f.Data = make([]byte, length)
	if _, err := io.ReadFull(r, f.Data); err != nil {
		return nil, err
	}
	return f, nil
}

func (f *DatagramFrame) Write(b *bytes.Buffer, _ protocol.VersionNumber) error {
	typeByte := uint8(0x30)
	if f.DataLenPresent {
		typeByte ^= 0x1
	}
	if f.FlowID != 0 {
		typeByte ^= 0x2
	}
	b.WriteByte(typeByte)
	if f.FlowID != 0 {
		utils.WriteVarInt(b, f.FlowID)
	}
	if f.DataLenPresent {
		utils.WriteVarInt(b, uint64(len(f.Data)))
	}
	b.Write(f.Data)
	return nil
}

// MaxDataLen returns the maximum data length
// If 0 is returned, writing will fail (a STREAM frame must contain at least 1 byte of data).
func (f *DatagramFrame) MaxDataLen(maxSize protocol.ByteCount, version protocol.VersionNumber) protocol.ByteCount {
	headerLen := protocol.ByteCount(1)
	if f.FlowID != 0 {
		headerLen += utils.VarIntLen(f.FlowID)
	}
	if f.DataLenPresent {
		// pretend that the data size will be 1 bytes
		// if it turns out that varint encoding the length will consume 2 bytes, we need to adjust the data length afterwards
		headerLen++
	}
	if headerLen > maxSize {
		return 0
	}
	maxDataLen := maxSize - headerLen
	if f.DataLenPresent && utils.VarIntLen(uint64(maxDataLen)) != 1 {
		maxDataLen--
	}
	return maxDataLen
}

// Length of a written frame
func (f *DatagramFrame) Length(_ protocol.VersionNumber) protocol.ByteCount {
	length := 1 + protocol.ByteCount(len(f.Data))
	if f.FlowID != 0 {
		length += utils.VarIntLen(f.FlowID)
	}
	if f.DataLenPresent {
		length += utils.VarIntLen(uint64(len(f.Data)))
	}
	return length
}

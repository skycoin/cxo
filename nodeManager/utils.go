package nodeManager

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"reflect"
)

func debugf(format string, a ...interface{}) (n int, err error) {
	if Debugging {
		return fmt.Println(fmt.Sprintf(format, a...))
	}
	return 0, nil
}
func debug(a ...interface{}) (n int, err error) {
	if Debugging {
		return fmt.Println(a...)
	}
	return 0, nil
}

func readBody(reader io.Reader, length int) ([]byte, error) {
	messageBytes := make([]byte, length)

	readN, err := reader.Read(messageBytes)
	debug("messageBytes", messageBytes)
	if err != nil {
		return []byte{}, err
	}
	if readN == 0 {
		return []byte{}, fmt.Errorf("read 0 bytes")
	}
	debug("read", readN)

	return messageBytes, nil
}

func readHeader(reader io.Reader) (string, uint64, error) {

	typePrefix := make([]byte, typePrefixLength)

	lengthPrefix := make([]byte, lengthPrefixLength)

	readN, err := reader.Read(typePrefix)
	debug("typePrefix", typePrefix)
	if err != nil {
		return "", 0, fmt.Errorf("error reading")
	}
	if readN == 0 {
		return "", 0, fmt.Errorf("read 0 bytes")
	}
	debug("read", readN)

	err = validateTypePrefix(typePrefix)
	if err != nil {
		return "", 0, err
	}

	readN, err = reader.Read(lengthPrefix)
	debug("lengthPrefix", lengthPrefix)
	if err != nil {
		return "", 0, fmt.Errorf("error reading")
	}
	if readN == 0 {
		return "", 0, fmt.Errorf("read 0 bytes")
	}
	debug("read", readN)

	lp, _ := binary.Uvarint(lengthPrefix)
	return string(typePrefix), lp, nil
}

// transmission is an array of bytes: [type_prefix][length_prefix][body]
func serializeToTransmission(msg interface{}) ([]byte, error) {
	header := make([]byte, typePrefixLength)

	typeHeader := make([]byte, typePrefixLength)

	name := reflect.Indirect(reflect.ValueOf(msg)).Type().Name()

	typeHeader = []byte(name)

	if len(typeHeader) > typePrefixLength {
		typeHeader = typeHeader[:typePrefixLength]
	}

	copy(header, typeHeader)

	body := encodeMessageToBody(msg)

	lengthHeader := make([]byte, lengthPrefixLength)
	binary.PutUvarint(lengthHeader, uint64(len(body)))
	header = append(header, lengthHeader...)

	var transmission []byte
	transmission = append(transmission, header...)
	transmission = append(transmission, body...)
	debugf("header: %v, body: %v \ntransmission: %v", header, body, transmission)

	return transmission, nil
}

func closeConn(conn net.Conn) error {
	debug("closing conn")
	return conn.Close()
}

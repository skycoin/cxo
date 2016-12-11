package nodeManager

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync"

	"github.com/skycoin/skycoin/src/aether/encoder"
)

const typePrefixLength = 32
const lengthPrefixLength = 10

// DownstreamMessage is a message receivable from downstream
type DownstreamMessage interface {
	HandleFromDownstream(cc *DownstreamContext) error
}
type DownstreamContext struct {
	Remote *Subscriber
	Local  *Node
}

// UpstreamMessage is a message receivable from upstream
type UpstreamMessage interface {
	HandleFromUpstream(cc *UpstreamContext) error
}
type UpstreamContext struct {
	Remote *Subscription
	Local  *Node
}

// Wraps encoder.DeserializeRawToValue and traps panics as an error
func deserializeBody(body []byte, v reflect.Value) (int, error) {
	return encoder.DeserializeRawToValue(body, v)
}

func encodeMessageToBody(msg interface{}) []byte {
	messageBytes := encoder.Serialize(msg)

	m := make([]byte, 0, len(messageBytes))
	m = append(m, messageBytes...) // message bytes
	return m
}

// Register a message struct that can be handled when incoming from downstream
func (node *Node) RegisterDownstreamMessage(msg interface{}) {
	registerMessage("DownstreamMessage", node.mu, node.downstream.callbacks, msg)
}

// Register a message struct that can be handled when incoming from upstream
func (node *Node) RegisterUpstreamMessage(msg interface{}) {
	registerMessage("UpstreamMessage", node.mu, node.upstream.callbacks, msg)
}

func registerMessage(messageTypeName string, mu *sync.RWMutex, destination map[string]reflect.Type, msg interface{}) {
	t := reflect.TypeOf(msg)
	rcvr := reflect.ValueOf(msg)
	name := reflect.Indirect(rcvr).Type().Name()
	typ := make([]byte, typePrefixLength)
	copy(typ, []byte(name))

	debugf("%v -> %v", string(typ), typ)

	err := validateTypePrefix(typ)
	if err != nil {
		log.Panicf("Message name contains non-valid byte: %v", err)
		return
	}

	// Confirm that the message supports the message interface
	// This should only be untrue if the user modified the message map
	// directly
	switch messageTypeName {
	case "UpstreamMessage":
		mptr := reflect.PtrTo(t)
		if !mptr.Implements(reflect.TypeOf((*UpstreamMessage)(nil)).Elem()) {
			m := "Message must implement the UpstreamMessage interface"
			log.Panicf("Invalid message at id %d: %s", name, m)
		}
	case "DownstreamMessage":
		mptr := reflect.PtrTo(t)
		if !mptr.Implements(reflect.TypeOf((*DownstreamMessage)(nil)).Elem()) {
			m := "Message must implement the DownstreamMessage interface"
			log.Panicf("Invalid message at id %d: %s", name, m)
		}
	}

	_, exists := destination[string(typ)]
	if exists {
		log.Panicf("Attempted to register message prefix %s twice",
			string(typ))
	}

	mu.Lock()
	defer mu.Unlock()

	destination[string(typ)] = t
	// DEBUG
	for callbackName := range destination {
		debugf("\n'%v'\n", callbackName)
	}
}

func validateTypePrefix(name []byte) error {
	// No empty prefixes allowed
	if name[0] == 0x00 {
		return errors.New("No empty message prefixes allowed")
	}

	hasEmpty := false
	for _, b := range name {
		if b == 0x00 {
			hasEmpty = true
		} else if hasEmpty {
			return errors.New("No non-null bytes allowed after a null byte")
		}
	}

	// validate all bytes
	for _, b := range name {
		if !((b >= '0' && b <= '9') || (b >= 'A' && b <= 'Z') ||
			(b >= 'a' && b <= 'z') || b == 0x00) {
			return fmt.Errorf("Invalid byte %v", b)
		}
	}

	return nil
}

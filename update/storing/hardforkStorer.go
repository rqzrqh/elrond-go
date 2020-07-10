package storing

import (
	"fmt"
	"sync"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/batch"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/update"
)

var log = logger.GetOrCreate("update/storing")

// ArgHardforkStorer represents the argument for the hardfork storer
type ArgHardforkStorer struct {
	KeysStore   storage.Storer
	KeyValue    storage.Storer
	Marshalizer marshal.Marshalizer
}

type hardforkStorer struct {
	keysStore   storage.Storer
	keyValue    storage.Storer
	marshalizer marshal.Marshalizer

	mut  sync.Mutex
	keys map[string][][]byte
}

// NewHardforkStorer returns a new instance of a specialized storer used in the hardfork process
func NewHardforkStorer(arg ArgHardforkStorer) (*hardforkStorer, error) {
	if check.IfNil(arg.KeysStore) {
		return nil, fmt.Errorf("%w for keys", update.ErrNilStorage)
	}
	if check.IfNil(arg.KeyValue) {
		return nil, fmt.Errorf("%w for key-values", update.ErrNilStorage)
	}
	if check.IfNil(arg.Marshalizer) {
		return nil, update.ErrNilMarshalizer
	}

	return &hardforkStorer{
		keysStore:   arg.KeysStore,
		keyValue:    arg.KeyValue,
		marshalizer: arg.Marshalizer,
		keys:        make(map[string][][]byte),
	}, nil
}

// Write adds the pair (key, value) in the state storer. Also, it does record the connection between the identifier and
// the key
func (hs *hardforkStorer) Write(identifier string, key []byte, value []byte) error {
	hs.mut.Lock()
	defer hs.mut.Unlock()

	hs.keys[identifier] = append(hs.keys[identifier], key)

	log.Trace("hardforkStorer.Write",
		"key", key,
		"value", value,
	)

	return hs.keyValue.Put(key, value)
}

// FinishedIdentifier prepares and writes the identifier along with its set of keys. It does so as to
// release the memory as soon as possible.
func (hs *hardforkStorer) FinishedIdentifier(identifier string) error {
	hs.mut.Lock()
	defer hs.mut.Unlock()

	log.Trace("hardforkStorer.FinishedIdentifier", "identifier", identifier)

	vals := hs.keys[identifier]
	if len(vals) == 0 {
		return nil
	}

	b := &batch.Batch{
		Data: vals,
	}

	buff, err := hs.marshalizer.Marshal(b)
	if err != nil {
		return err
	}

	delete(hs.keys, identifier)

	return hs.keysStore.Put([]byte(identifier), buff)
}

// RangeKeys iterates over all identifiers and its set of keys. The order is not guaranteed.
func (hs *hardforkStorer) RangeKeys(handler func(identifier string, keys [][]byte)) {
	if handler == nil {
		return
	}

	chIterate := hs.keysStore.Iterate()
	for kv := range chIterate {
		b := &batch.Batch{}
		err := hs.marshalizer.Unmarshal(b, kv.Val())
		if err != nil {
			log.Warn("error reading identifiers",
				"key", string(kv.Key()),
				"error", err,
			)
			continue
		}

		handler(string(kv.Key()), b.Data)
	}
}

// Get returns the value of a provided key from the state storer
func (hs *hardforkStorer) Get(key []byte) ([]byte, error) {
	return hs.keyValue.Get(key)
}

// Close tryies to close both storers
func (hs *hardforkStorer) Close() error {
	errKeysStore := hs.keysStore.Close()
	errKeyValue := hs.keyValue.Close()

	if errKeysStore != nil {
		return errKeysStore
	}

	return errKeyValue
}

// IsInterfaceNil returns true if there is no value under the interface
func (hs *hardforkStorer) IsInterfaceNil() bool {
	return hs == nil
}

package builtInFunctions

import (
	"bytes"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/vmcommon"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/vm"
)

var _ process.BuiltinFunction = (*esdtFreezeWipe)(nil)

type esdtFreezeWipe struct {
	marshalizer marshal.Marshalizer
	keyPrefix   []byte
	wipe        bool
	freeze      bool
}

// NewESDTFreezeWipeFunc returns the esdt freeze/un-freeze/wipe built-in function component
func NewESDTFreezeWipeFunc(
	marshalizer marshal.Marshalizer,
	freeze bool,
	wipe bool,
) (*esdtFreezeWipe, error) {
	if check.IfNil(marshalizer) {
		return nil, process.ErrNilMarshalizer
	}

	e := &esdtFreezeWipe{
		marshalizer: marshalizer,
		keyPrefix:   []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier),
		freeze:      freeze,
		wipe:        wipe,
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtFreezeWipe) SetNewGasConfig(_ *process.GasCost) {
}

// ProcessBuiltinFunction resolves ESDT transfer function call
func (e *esdtFreezeWipe) ProcessBuiltinFunction(
	_, acntDst state.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	if vmInput == nil {
		return nil, process.ErrNilVmInput
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return nil, process.ErrBuiltInFunctionCalledWithValue
	}
	if len(vmInput.Arguments) != 1 {
		return nil, process.ErrInvalidArguments
	}
	if !bytes.Equal(vmInput.CallerAddr, vm.ESDTSCAddress) {
		return nil, process.ErrAddressIsNotESDTSystemSC
	}
	if check.IfNil(acntDst) {
		return nil, process.ErrNilUserAccount
	}

	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	log.Trace(vmInput.Function, "sender", vmInput.CallerAddr, "receiver", vmInput.RecipientAddr, "token", esdtTokenKey)

	if e.wipe {
		err := e.wipeIfApplicable(acntDst, esdtTokenKey)
		if err != nil {
			return nil, err
		}
	} else {
		err := e.toggleFreeze(acntDst, esdtTokenKey)
		if err != nil {
			return nil, err
		}
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
	return vmOutput, nil
}

func (e *esdtFreezeWipe) wipeIfApplicable(acntDst state.UserAccountHandler, tokenKey []byte) error {
	tokenData, err := getESDTDataFromKey(acntDst, tokenKey, e.marshalizer)
	if err != nil {
		return err
	}

	esdtUserMetadata := ESDTUserMetadataFromBytes(tokenData.Properties)
	if !esdtUserMetadata.Frozen {
		return process.ErrCannotWipeAccountNotFrozen
	}

	return acntDst.DataTrieTracker().SaveKeyValue(tokenKey, nil)
}

func (e *esdtFreezeWipe) toggleFreeze(acntDst state.UserAccountHandler, tokenKey []byte) error {
	tokenData, err := getESDTDataFromKey(acntDst, tokenKey, e.marshalizer)
	if err != nil {
		return err
	}

	esdtUserMetadata := ESDTUserMetadataFromBytes(tokenData.Properties)
	esdtUserMetadata.Frozen = e.freeze
	tokenData.Properties = esdtUserMetadata.ToBytes()

	err = saveESDTData(acntDst, tokenData, tokenKey, e.marshalizer)
	if err != nil {
		return err
	}

	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtFreezeWipe) IsInterfaceNil() bool {
	return e == nil
}

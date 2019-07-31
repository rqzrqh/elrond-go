package metablock_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	dataBlock "github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/block"
	"github.com/ElrondNetwork/elrond-go/process/metablock"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/stretchr/testify/assert"
)

var durTimeout = time.Duration(time.Second)

//------- NewShardHeaderInterceptor

func TestNewShardHeaderInterceptor_NilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	headers := &mock.CacherStub{}
	storer := &mock.StorerStub{}
	hi, err := metablock.NewShardHeaderInterceptor(
		nil,
		headers,
		&mock.Uint64SyncMapCacherStub{},
		storer,
		mock.NewMultiSigner(),
		mock.HasherMock{},
		mock.NewOneShardCoordinatorMock(),
		&mock.ChronologyValidatorStub{},
	)

	assert.Equal(t, process.ErrNilMarshalizer, err)
	assert.Nil(t, hi)
}

func TestNewShardHeaderInterceptor_NilHeadersShouldErr(t *testing.T) {
	t.Parallel()

	storer := &mock.StorerStub{}
	hi, err := metablock.NewShardHeaderInterceptor(
		&mock.MarshalizerMock{},
		nil,
		&mock.Uint64SyncMapCacherStub{},
		storer,
		mock.NewMultiSigner(),
		mock.HasherMock{},
		mock.NewOneShardCoordinatorMock(),
		&mock.ChronologyValidatorStub{},
	)

	assert.Equal(t, process.ErrNilHeadersDataPool, err)
	assert.Nil(t, hi)
}

func TestNewShardHeaderInterceptor_OkValsShouldWork(t *testing.T) {
	t.Parallel()

	headers := &mock.CacherStub{}
	storer := &mock.StorerStub{}
	hi, err := metablock.NewShardHeaderInterceptor(
		&mock.MarshalizerMock{},
		headers,
		&mock.Uint64SyncMapCacherStub{},
		storer,
		mock.NewMultiSigner(),
		mock.HasherMock{},
		mock.NewOneShardCoordinatorMock(),
		&mock.ChronologyValidatorStub{},
	)

	assert.Nil(t, err)
	assert.NotNil(t, hi)
}

//------- ProcessReceivedMessage

func TestShardHeaderInterceptor_ProcessReceivedMessageNilMessageShouldErr(t *testing.T) {
	t.Parallel()

	headers := &mock.CacherStub{}
	storer := &mock.StorerStub{}
	hi, _ := metablock.NewShardHeaderInterceptor(
		&mock.MarshalizerMock{},
		headers,
		&mock.Uint64SyncMapCacherStub{},
		storer,
		mock.NewMultiSigner(),
		mock.HasherMock{},
		mock.NewOneShardCoordinatorMock(),
		&mock.ChronologyValidatorStub{},
	)

	assert.Equal(t, process.ErrNilMessage, hi.ProcessReceivedMessage(nil))
}

func TestShardHeaderInterceptor_ProcessReceivedMessageValsOkShouldWork(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	chanDone := make(chan struct{}, 1)
	testedNonce := uint64(67)
	headers := &mock.CacherStub{}
	multisigner := mock.NewMultiSigner()
	chronologyValidator := &mock.ChronologyValidatorStub{
		ValidateReceivedBlockCalled: func(shardID uint32, epoch uint32, nonce uint64, round uint64) error {
			return nil
		},
	}
	storer := &mock.StorerStub{}
	storer.HasCalled = func(key []byte) error {
		return errors.New("key not found")
	}
	hdrsNonces := &mock.Uint64SyncMapCacherStub{
		MergeCalled: func(nonce uint64, src dataRetriever.ShardIdHashMap) {},
	}

	hi, _ := metablock.NewShardHeaderInterceptor(
		marshalizer,
		headers,
		hdrsNonces,
		storer,
		multisigner,
		mock.HasherMock{},
		mock.NewOneShardCoordinatorMock(),
		chronologyValidator,
	)

	hdr := block.NewInterceptedHeader(multisigner, chronologyValidator)
	hdr.Nonce = testedNonce
	hdr.ShardId = 0
	hdr.PrevHash = make([]byte, 0)
	hdr.PubKeysBitmap = make([]byte, 0)
	hdr.BlockBodyType = dataBlock.TxBlock
	hdr.Signature = make([]byte, 0)
	hdr.SetHash([]byte("aaa"))
	hdr.RootHash = make([]byte, 0)
	hdr.PrevRandSeed = make([]byte, 0)
	hdr.RandSeed = make([]byte, 0)
	hdr.MiniBlockHeaders = make([]dataBlock.MiniBlockHeader, 0)

	buff, _ := marshalizer.Marshal(hdr)
	msg := &mock.P2PMessageMock{
		DataField: buff,
	}

	headers.HasOrAddCalled = func(key []byte, value interface{}) (ok, evicted bool) {
		aaaHash := mock.HasherMock{}.Compute(string(buff))
		if bytes.Equal(aaaHash, key) {
			chanDone <- struct{}{}
		}
		return false, false
	}

	assert.Nil(t, hi.ProcessReceivedMessage(msg))
	select {
	case <-chanDone:
	case <-time.After(durTimeout):
		assert.Fail(t, "timeout while waiting for block to be inserted in the pool")
	}
}

func TestShardHeaderInterceptor_ProcessReceivedMessageTestHdrNonces(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	chanDone := make(chan struct{}, 1)
	testedNonce := uint64(67)
	headers := &mock.CacherStub{}
	multisigner := mock.NewMultiSigner()
	chronologyValidator := &mock.ChronologyValidatorStub{
		ValidateReceivedBlockCalled: func(shardID uint32, epoch uint32, nonce uint64, round uint64) error {
			return nil
		},
	}
	storer := &mock.StorerStub{}
	storer.HasCalled = func(key []byte) error {
		return errors.New("key not found")
	}
	hdrsNonces := &mock.Uint64SyncMapCacherStub{}

	hi, _ := metablock.NewShardHeaderInterceptor(
		marshalizer,
		headers,
		hdrsNonces,
		storer,
		multisigner,
		mock.HasherMock{},
		mock.NewMultiShardsCoordinatorMock(2),
		chronologyValidator,
	)

	hdr := block.NewInterceptedHeader(multisigner, chronologyValidator)
	hdr.Nonce = testedNonce
	hdr.ShardId = 0
	hdr.PrevHash = make([]byte, 0)
	hdr.PubKeysBitmap = make([]byte, 0)
	hdr.BlockBodyType = dataBlock.TxBlock
	hdr.Signature = make([]byte, 0)
	hdr.SetHash([]byte("aaa"))
	hdr.RootHash = make([]byte, 0)
	hdr.PrevRandSeed = make([]byte, 0)
	hdr.RandSeed = make([]byte, 0)
	hdr.MiniBlockHeaders = make([]dataBlock.MiniBlockHeader, 0)

	buff, _ := marshalizer.Marshal(hdr)
	msg := &mock.P2PMessageMock{
		DataField: buff,
	}

	headers.HasOrAddCalled = func(key []byte, value interface{}) (ok, evicted bool) {
		return false, false
	}

	hdrsNonces.MergeCalled = func(nonce uint64, src dataRetriever.ShardIdHashMap) {
		if testedNonce == nonce {
			chanDone <- struct{}{}
		}
	}

	assert.Nil(t, hi.ProcessReceivedMessage(msg))
	select {
	case <-chanDone:
	case <-time.After(durTimeout):
		assert.Fail(t, "timeout while waiting for block to be inserted in the pool")
	}
}

func TestShardHeaderInterceptor_ProcessReceivedMessageIsInStorageShouldNotAdd(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	chanDone := make(chan struct{}, 1)
	testedNonce := uint64(67)
	headers := &mock.CacherStub{}
	multisigner := mock.NewMultiSigner()
	chronologyValidator := &mock.ChronologyValidatorStub{
		ValidateReceivedBlockCalled: func(shardID uint32, epoch uint32, nonce uint64, round uint64) error {
			return nil
		},
	}
	storer := &mock.StorerStub{}
	storer.HasCalled = func(key []byte) error {
		return nil
	}
	hi, _ := metablock.NewShardHeaderInterceptor(
		marshalizer,
		headers,
		&mock.Uint64SyncMapCacherStub{},
		storer,
		multisigner,
		mock.HasherMock{},
		mock.NewOneShardCoordinatorMock(),
		chronologyValidator,
	)

	hdr := block.NewInterceptedHeader(multisigner, chronologyValidator)
	hdr.Nonce = testedNonce
	hdr.ShardId = 0
	hdr.PrevHash = make([]byte, 0)
	hdr.PubKeysBitmap = make([]byte, 0)
	hdr.BlockBodyType = dataBlock.TxBlock
	hdr.Signature = make([]byte, 0)
	hdr.RootHash = make([]byte, 0)
	hdr.PrevRandSeed = make([]byte, 0)
	hdr.RandSeed = make([]byte, 0)
	hdr.SetHash([]byte("aaa"))
	hdr.MiniBlockHeaders = make([]dataBlock.MiniBlockHeader, 0)

	buff, _ := marshalizer.Marshal(hdr)
	msg := &mock.P2PMessageMock{
		DataField: buff,
	}

	headers.HasOrAddCalled = func(key []byte, value interface{}) (ok, evicted bool) {
		aaaHash := mock.HasherMock{}.Compute(string(buff))
		if bytes.Equal(aaaHash, key) {
			chanDone <- struct{}{}
		}
		return false, false
	}

	assert.Nil(t, hi.ProcessReceivedMessage(msg))
	select {
	case <-chanDone:
		assert.Fail(t, "should have not add block in pool")
	case <-time.After(durTimeout):
	}
}

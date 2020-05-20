package processors

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type ParticipantsProcessorTestSuiteLedger struct {
	suite.Suite
	processor                        *ParticipantsProcessor
	mockQ                            *history.MockQParticipants
	mockBatchInsertBuilder           *history.MockTransactionParticipantsBatchInsertBuilder
	mockOperationsBatchInsertBuilder *history.MockOperationParticipantBatchInsertBuilder

	firstTx            io.LedgerTransaction
	secondTx           io.LedgerTransaction
	thirdTx            io.LedgerTransaction
	firstTxID          int64
	secondTxID         int64
	thirdTxID          int64
	addresses          []string
	unmuxedAddresses   []string
	addressToID        map[string]int64
	unmuxedAddressToID map[string]int64
	txs                []io.LedgerTransaction
}

func TestParticipantsProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(ParticipantsProcessorTestSuiteLedger))
}

func (s *ParticipantsProcessorTestSuiteLedger) SetupTest() {
	s.mockQ = &history.MockQParticipants{}
	s.mockBatchInsertBuilder = &history.MockTransactionParticipantsBatchInsertBuilder{}
	s.mockOperationsBatchInsertBuilder = &history.MockOperationParticipantBatchInsertBuilder{}
	sequence := uint32(20)

	s.addresses = []string{
		"MDPK3PXP32W353Z3MC7QB3RZMAIPPNDVIU5VI5YDIMPJB7YJI6U43VAPLZNCO4RC2WIDK",
		"GAXI33UCLQTCKM2NMRBS7XYBR535LLEVAHL5YBN4FTCB4HZHT7ZA5CVK",
		"GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
	}

	s.unmuxedAddresses = make([]string, len(s.addresses))
	for i := range s.addresses {
		acid := xdr.mustMuxedAccountAddress(s.addresses[i]).ToAccountId()
		s.unmuxedAddresses[i] = acid.Address()
	}

	s.firstTx = createTransaction(true, 1)
	s.firstTx.Index = 1
	s.firstTx.Envelope.Operations()[0].Body = xdr.OperationBody{
		Type: xdr.OperationTypePayment,
		PaymentOp: &xdr.PaymentOp{
			Destination: xdr.mustMuxedAccountAddress(s.addresses[0]),
			Asset:       xdr.Asset{Type: xdr.AssetTypeAssetTypeNative},
			Amount:      100,
		},
	}
	s.firstTx.Envelope.V1.Tx.SourceAccount = xdr.mustMuxedAccountAddress(s.addresses[0])
	s.firstTxID = toid.New(int32(sequence), 1, 0).ToInt64()

	s.secondTx = createTransaction(true, 1)
	s.secondTx.Index = 2
	s.secondTx.Envelope.Operations()[0].Body = xdr.OperationBody{
		Type: xdr.OperationTypeCreateAccount,
		CreateAccountOp: &xdr.CreateAccountOp{
			Destination: xdr.MustAddress(s.addresses[1]),
		},
	}
	s.secondTx.Envelope.V1.Tx.SourceAccount = xdr.mustMuxedAccountAddress(s.addresses[2])
	s.secondTxID = toid.New(int32(sequence), 2, 0).ToInt64()

	s.thirdTx = createTransaction(true, 1)
	s.thirdTx.Index = 3
	s.thirdTx.Envelope.V1.Tx.SourceAccount = xdr.mustMuxedAccountAddress(s.addresses[0])
	s.thirdTxID = toid.New(int32(sequence), 3, 0).ToInt64()

	s.addressToID = map[string]int64{
		s.addresses[0]: 2,
		s.addresses[1]: 20,
		s.addresses[2]: 200,
	}

	s.unmuxedAddressToID = map[string]int64{
		s.unmuxedAddresses[0]: 2,
		s.unmuxedAddresses[1]: 20,
		s.unmuxedAddresses[2]: 200,
	}

	s.processor = NewParticipantsProcessor(
		s.mockQ,
		sequence,
	)

	s.txs = []io.LedgerTransaction{
		s.firstTx,
		s.secondTx,
		s.thirdTx,
	}
}

func (s *ParticipantsProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockBatchInsertBuilder.AssertExpectations(s.T())
	s.mockOperationsBatchInsertBuilder.AssertExpectations(s.T())
}

func (s *ParticipantsProcessorTestSuiteLedger) mockSuccessfulTransactionBatchAdds() {
	s.mockBatchInsertBuilder.On(
		"Add", s.firstTxID, s.addressToID[s.addresses[0]],
	).Return(nil).Once()

	s.mockBatchInsertBuilder.On(
		"Add", s.secondTxID, s.addressToID[s.addresses[1]],
	).Return(nil).Once()
	s.mockBatchInsertBuilder.On(
		"Add", s.secondTxID, s.addressToID[s.addresses[2]],
	).Return(nil).Once()

	s.mockBatchInsertBuilder.On(
		"Add", s.thirdTxID, s.addressToID[s.addresses[0]],
	).Return(nil).Once()
}

func (s *ParticipantsProcessorTestSuiteLedger) mockSuccessfulOperationBatchAdds() {
	s.mockOperationsBatchInsertBuilder.On(
		"Add", s.firstTxID+1, s.addressToID[s.addresses[0]],
	).Return(nil).Once()
	s.mockOperationsBatchInsertBuilder.On(
		"Add", s.secondTxID+1, s.addressToID[s.addresses[1]],
	).Return(nil).Once()
	s.mockOperationsBatchInsertBuilder.On(
		"Add", s.secondTxID+1, s.addressToID[s.addresses[2]],
	).Return(nil).Once()
	s.mockOperationsBatchInsertBuilder.On(
		"Add", s.thirdTxID+1, s.addressToID[s.addresses[0]],
	).Return(nil).Once()
}
func (s *ParticipantsProcessorTestSuiteLedger) TestEmptyParticipants() {
	err := s.processor.Commit()
	s.Assert().NoError(err)
}

func (s *ParticipantsProcessorTestSuiteLedger) TestFeeBumptransaction() {
	feeBumpTx := createTransaction(true, 0)
	feeBumpTx.Index = 1
	feeBumpTx.Envelope.V1.Tx.SourceAccount = xdr.mustMuxedAccountAddress(s.addresses[0])
	feeBumpTx.Envelope.FeeBump = &xdr.FeeBumpTransactionEnvelope{
		Tx: xdr.FeeBumpTransaction{
			FeeSource: xdr.mustMuxedAccountAddress(s.addresses[1]),
			Fee:       100,
			InnerTx: xdr.FeeBumpTransactionInnerTx{
				Type: xdr.EnvelopeTypeEnvelopeTypeTx,
				V1:   feeBumpTx.Envelope.V1,
			},
		},
	}
	feeBumpTx.Envelope.V1 = nil
	feeBumpTx.Envelope.Type = xdr.EnvelopeTypeEnvelopeTypeTxFeeBump
	feeBumpTx.Result.Result.Result.Code = xdr.TransactionResultCodeTxFeeBumpInnerSuccess
	feeBumpTx.Result.Result.Result.InnerResultPair = &xdr.InnerTransactionResultPair{
		Result: xdr.InnerTransactionResult{
			Result: xdr.InnerTransactionResultResult{
				Code:    xdr.TransactionResultCodeTxSuccess,
				Results: &[]xdr.OperationResult{},
			},
		},
	}
	feeBumpTx.Result.Result.Result.Results = nil
	feeBumpTxID := toid.New(20, 1, 0).ToInt64()

	unmuxedAddresses := s.unmuxedAddresses[:2]
	unmuxedAddressToID := map[string]int64{
		unmuxedAddresses[0]: s.unmuxedAddressToID[unmuxedAddresses[0]],
		unmuxedAddresses[1]: s.unmuxedAddressToID[unmuxedAddresses[1]],
	}
	s.mockQ.On("CreateAccounts", mock.AnythingOfType("[]string"), maxBatchSize).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]string)
			s.Assert().ElementsMatch(
				unmuxedAddresses,
				arg,
			)
		}).Return(unmuxedAddressToID, nil).Once()
	s.mockQ.On("NewTransactionParticipantsBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()
	s.mockQ.On("NewOperationParticipantBatchInsertBuilder", maxBatchSize).
		Return(s.mockOperationsBatchInsertBuilder).Once()

	s.mockBatchInsertBuilder.On(
		"Add", feeBumpTxID, unmuxedAddressToID[unmuxedAddresses[0]],
	).Return(nil).Once()
	s.mockBatchInsertBuilder.On(
		"Add", feeBumpTxID, unmuxedAddressToID[unmuxedAddresses[1]],
	).Return(nil).Once()

	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()
	s.mockOperationsBatchInsertBuilder.On("Exec").Return(nil).Once()

	s.Assert().NoError(s.processor.ProcessTransaction(feeBumpTx))
	s.Assert().NoError(s.processor.Commit())
}

func (s *ParticipantsProcessorTestSuiteLedger) TestIngestParticipantsSucceeds() {
	s.mockQ.On("CreateAccounts", mock.AnythingOfType("[]string"), maxBatchSize).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]string)
			s.Assert().ElementsMatch(
				s.unmuxedAddresses,
				arg,
			)
		}).Return(s.unmuxedAddressToID, nil).Once()
	s.mockQ.On("NewTransactionParticipantsBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()
	s.mockQ.On("NewOperationParticipantBatchInsertBuilder", maxBatchSize).
		Return(s.mockOperationsBatchInsertBuilder).Once()

	s.mockSuccessfulTransactionBatchAdds()
	s.mockSuccessfulOperationBatchAdds()

	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()
	s.mockOperationsBatchInsertBuilder.On("Exec").Return(nil).Once()

	for _, tx := range s.txs {
		err := s.processor.ProcessTransaction(tx)
		s.Assert().NoError(err)
	}
	err := s.processor.Commit()
	s.Assert().NoError(err)
}

func (s *ParticipantsProcessorTestSuiteLedger) TestCreateAccountsFails() {
	s.mockQ.On("CreateAccounts", mock.AnythingOfType("[]string"), maxBatchSize).
		Return(s.addressToID, errors.New("transient error")).Once()
	for _, tx := range s.txs {
		err := s.processor.ProcessTransaction(tx)
		s.Assert().NoError(err)
	}
	err := s.processor.Commit()
	s.Assert().EqualError(err, "Could not create account ids: transient error")
}

func (s *ParticipantsProcessorTestSuiteLedger) TestBatchAddFails() {
	s.mockQ.On("CreateAccounts", mock.AnythingOfType("[]string"), maxBatchSize).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]string)
			s.Assert().ElementsMatch(
				s.unmuxedAddresses,
				arg,
			)
		}).Return(s.unmuxedAddressToID, nil).Once()
	s.mockQ.On("NewTransactionParticipantsBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()

	s.mockBatchInsertBuilder.On(
		"Add", s.firstTxID, s.addressToID[s.addresses[0]],
	).Return(errors.New("transient error")).Once()

	s.mockBatchInsertBuilder.On(
		"Add", s.secondTxID, s.addressToID[s.addresses[1]],
	).Return(nil).Maybe()
	s.mockBatchInsertBuilder.On(
		"Add", s.secondTxID, s.addressToID[s.addresses[2]],
	).Return(nil).Maybe()

	s.mockBatchInsertBuilder.On(
		"Add", s.thirdTxID, s.addressToID[s.addresses[0]],
	).Return(nil).Maybe()
	for _, tx := range s.txs {
		err := s.processor.ProcessTransaction(tx)
		s.Assert().NoError(err)
	}
	err := s.processor.Commit()
	s.Assert().EqualError(err, "Could not insert transaction participant in db: transient error")
}

func (s *ParticipantsProcessorTestSuiteLedger) TestOperationParticipantsBatchAddFails() {
	s.mockQ.On("CreateAccounts", mock.AnythingOfType("[]string"), maxBatchSize).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]string)
			s.Assert().ElementsMatch(
				s.unmuxedAddresses,
				arg,
			)
		}).Return(s.unmuxedAddressToID, nil).Once()
	s.mockQ.On("NewTransactionParticipantsBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()
	s.mockQ.On("NewOperationParticipantBatchInsertBuilder", maxBatchSize).
		Return(s.mockOperationsBatchInsertBuilder).Once()

	s.mockSuccessfulTransactionBatchAdds()

	s.mockOperationsBatchInsertBuilder.On(
		"Add", s.firstTxID+1, s.addressToID[s.addresses[0]],
	).Return(errors.New("transient error")).Once()
	s.mockOperationsBatchInsertBuilder.On(
		"Add", s.secondTxID+1, s.addressToID[s.addresses[1]],
	).Return(nil).Maybe()
	s.mockOperationsBatchInsertBuilder.On(
		"Add", s.secondTxID+1, s.addressToID[s.addresses[2]],
	).Return(nil).Maybe()
	s.mockOperationsBatchInsertBuilder.On(
		"Add", s.thirdTxID+1, s.addressToID[s.addresses[0]],
	).Return(nil).Maybe()

	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()

	for _, tx := range s.txs {
		err := s.processor.ProcessTransaction(tx)
		s.Assert().NoError(err)
	}
	err := s.processor.Commit()
	s.Assert().EqualError(err, "could not insert operation participant in db: transient error")
}

func (s *ParticipantsProcessorTestSuiteLedger) TestBatchAddExecFails() {
	s.mockQ.On("CreateAccounts", mock.AnythingOfType("[]string"), maxBatchSize).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]string)
			s.Assert().ElementsMatch(
				s.unmuxedAddresses,
				arg,
			)
		}).Return(s.unmuxedAddressToID, nil).Once()
	s.mockQ.On("NewTransactionParticipantsBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()

	s.mockSuccessfulTransactionBatchAdds()

	s.mockBatchInsertBuilder.On("Exec").Return(errors.New("transient error")).Once()

	for _, tx := range s.txs {
		err := s.processor.ProcessTransaction(tx)
		s.Assert().NoError(err)
	}
	err := s.processor.Commit()
	s.Assert().EqualError(err, "Could not flush transaction participants to db: transient error")
}

func (s *ParticipantsProcessorTestSuiteLedger) TestOpeartionBatchAddExecFails() {
	s.mockQ.On("CreateAccounts", mock.AnythingOfType("[]string"), maxBatchSize).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]string)
			s.Assert().ElementsMatch(
				s.unmuxedAddresses,
				arg,
			)
		}).Return(s.unmuxedAddressToID, nil).Once()
	s.mockQ.On("NewTransactionParticipantsBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()
	s.mockQ.On("NewOperationParticipantBatchInsertBuilder", maxBatchSize).
		Return(s.mockOperationsBatchInsertBuilder).Once()

	s.mockSuccessfulTransactionBatchAdds()
	s.mockSuccessfulOperationBatchAdds()

	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()
	s.mockOperationsBatchInsertBuilder.On("Exec").Return(errors.New("transient error")).Once()

	for _, tx := range s.txs {
		err := s.processor.ProcessTransaction(tx)
		s.Assert().NoError(err)
	}
	err := s.processor.Commit()
	s.Assert().EqualError(err, "could not flush operation participants to db: transient error")
}

package chaincode

import (
	"crypto/x509"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/v2/pkg/cid"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
	peer "github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ==========================================
// Mocks
// ==========================================

type MockStub struct {
	State         map[string][]byte
	Events        map[string][]byte
	TxID          string
	ChaincodeArgs [][]byte
}

func NewMockStub() *MockStub {
	return &MockStub{
		State:  make(map[string][]byte),
		Events: make(map[string][]byte),
		TxID:   "tx001",
	}
}

func (m *MockStub) GetState(key string) ([]byte, error) {
	return m.State[key], nil
}

func (m *MockStub) PutState(key string, value []byte) error {
	m.State[key] = value
	return nil
}

func (m *MockStub) DelState(key string) error {
	delete(m.State, key)
	return nil
}

func (m *MockStub) CreateCompositeKey(objectType string, attributes []string) (string, error) {
	// Mimic Fabric CompositeKey: objectType + 0x00 + attr1 + 0x00 + ...
	// But for testing, just a unique string is enough as long as consistent.
	return objectType + "\x00" + strings.Join(attributes, "\x00"), nil
}

func (m *MockStub) SetEvent(name string, payload []byte) error {
	m.Events[name] = payload
	return nil
}

func (m *MockStub) GetTxID() string {
	return m.TxID
}

// Satisfy Interface (Unimplemented methods will panic if called)
func (m *MockStub) GetArgs() [][]byte                            { return m.ChaincodeArgs }
func (m *MockStub) GetStringArgs() []string                      { return nil }
func (m *MockStub) GetFunctionAndParameters() (string, []string) { return "", nil }
func (m *MockStub) GetArgsSlice() ([]byte, error)                { return nil, nil }
func (m *MockStub) GetStateByRange(startKey, endKey string) (shim.StateQueryIteratorInterface, error) {
	return nil, nil
}
func (m *MockStub) GetStateByPartialCompositeKey(objectType string, keys []string) (shim.StateQueryIteratorInterface, error) {
	return nil, nil
}
func (m *MockStub) SplitCompositeKey(compositeKey string) (string, []string, error) {
	return "", nil, nil
}
func (m *MockStub) GetQueryResult(query string) (shim.StateQueryIteratorInterface, error) {
	return nil, nil
}
func (m *MockStub) GetHistoryForKey(key string) (shim.HistoryQueryIteratorInterface, error) {
	return nil, nil
}
func (m *MockStub) GetPrivateData(collection, key string) ([]byte, error)     { return nil, nil }
func (m *MockStub) GetPrivateDataHash(collection, key string) ([]byte, error) { return nil, nil }
func (m *MockStub) PutPrivateData(collection, key string, value []byte) error { return nil }
func (m *MockStub) DelPrivateData(collection, key string) error               { return nil }
func (m *MockStub) PurgePrivateData(collection, key string) error             { return nil }
func (m *MockStub) SetPrivateDataValidationParameter(collection, key string, ep []byte) error {
	return nil
}
func (m *MockStub) GetPrivateDataValidationParameter(collection, key string) ([]byte, error) {
	return nil, nil
}
func (m *MockStub) GetPrivateDataByRange(collection, startKey, endKey string) (shim.StateQueryIteratorInterface, error) {
	return nil, nil
}
func (m *MockStub) GetPrivateDataByPartialCompositeKey(collection, objectType string, keys []string) (shim.StateQueryIteratorInterface, error) {
	return nil, nil
}
func (m *MockStub) GetPrivateDataQueryResult(collection, query string) (shim.StateQueryIteratorInterface, error) {
	return nil, nil
}
func (m *MockStub) GetCreator() ([]byte, error)                      { return nil, nil }
func (m *MockStub) GetTransient() (map[string][]byte, error)         { return nil, nil }
func (m *MockStub) GetBinding() ([]byte, error)                      { return nil, nil }
func (m *MockStub) GetDecorations() map[string][]byte                { return nil }
func (m *MockStub) GetSignedProposal() (*peer.SignedProposal, error) { return nil, nil }
func (m *MockStub) GetChannelID() string                             { return "" }
func (m *MockStub) InvokeChaincode(chaincodeName string, args [][]byte, channel string) *peer.Response {
	return &peer.Response{}
}
func (m *MockStub) SetStateValidationParameter(key string, ep []byte) error { return nil }
func (m *MockStub) GetStateValidationParameter(key string) ([]byte, error)  { return nil, nil }
func (m *MockStub) GetQueryResultWithPagination(query string, pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *peer.QueryResponseMetadata, error) {
	return nil, nil, nil
}
func (m *MockStub) GetStateByPartialCompositeKeyWithPagination(objectType string, keys []string, pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *peer.QueryResponseMetadata, error) {
	return nil, nil, nil
}
func (m *MockStub) GetStateByRangeWithPagination(startKey, endKey string, pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *peer.QueryResponseMetadata, error) {
	return nil, nil, nil
}
func (m *MockStub) GetTxTimestamp() (*timestamppb.Timestamp, error) { return nil, nil }

type MockClientIdentity struct {
	MSPID string
	ID    string
}

func (m *MockClientIdentity) GetID() (string, error) {
	return m.ID, nil
}
func (m *MockClientIdentity) GetMSPID() (string, error) {
	return m.MSPID, nil
}

// Satisfy Interface
func (m *MockClientIdentity) GetAttributeValue(attrName string) (string, bool, error) {
	return "", false, nil
}
func (m *MockClientIdentity) AssertAttributeValue(attrName, attrValue string) error { return nil }
func (m *MockClientIdentity) GetX509Certificate() (*x509.Certificate, error)        { return nil, nil }

type MockContext struct {
	contractapi.TransactionContext
	mockStub *MockStub
	mockCID  *MockClientIdentity
}

func (m *MockContext) GetStub() shim.ChaincodeStubInterface {
	return m.mockStub
}
func (m *MockContext) GetClientIdentity() cid.ClientIdentity {
	return m.mockCID
}

// ==========================================
// Tests
// ==========================================

func TestInitialize(t *testing.T) {
	sc := new(SmartContract)
	stub := NewMockStub()
	cid := &MockClientIdentity{MSPID: "Org1MSP", ID: "user1"}
	ctx := &MockContext{mockStub: stub, mockCID: cid}

	// 1. Success
	err := stub.PutState(nameKey, nil) // Ensure empty
	success, err := sc.Initialize(ctx, "MyToken", "MTK", "18")
	if err != nil {
		t.Fatalf("Expected Initialize to succeed, got %v", err)
	}
	if !success {
		t.Errorf("Expected True")
	}

	state, _ := stub.GetState(nameKey)
	if string(state) != "MyToken" {
		t.Errorf("Expected name MyToken, got %s", string(state))
	}

	// 2. Fail if already initialized
	success, err = sc.Initialize(ctx, "NewName", "NN", "18")
	if err == nil {
		t.Errorf("Expected error for double initialization")
	}

	// 3. Fail if wrong MSP
	cid.MSPID = "Org2MSP"
	success, err = sc.Initialize(ctx, "Fail", "F", "18")
	if err == nil {
		t.Errorf("Expected error for wrong MSP")
	}
}

func TestMint(t *testing.T) {
	sc := new(SmartContract)
	stub := NewMockStub()
	cid := &MockClientIdentity{MSPID: "Org1MSP", ID: "minter1"}
	ctx := &MockContext{mockStub: stub, mockCID: cid}

	// Setup: Initialize
	stub.PutState(nameKey, []byte("Token"))

	// 1. Valid Mint
	err := sc.Mint(ctx, 1000)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}

	bal, _ := stub.GetState("minter1")
	if string(bal) != "1000" {
		t.Errorf("Expected balance 1000, got %s", string(bal))
	}

	ts, _ := stub.GetState(totalSupplyKey)
	if string(ts) != "1000" {
		t.Errorf("Expected TS 1000, got %s", string(ts))
	}

	// 2. Invalid Amount
	err = sc.Mint(ctx, -50)
	if err == nil {
		t.Errorf("Expected error for negative mint")
	}
}

func TestTransferDebit_MVCC(t *testing.T) {
	sc := new(SmartContract)
	stub := NewMockStub()
	cid := &MockClientIdentity{MSPID: "Org1MSP", ID: "sender1"}
	ctx := &MockContext{mockStub: stub, mockCID: cid}

	stub.PutState(nameKey, []byte("Token"))

	// Setup: Sender has 500
	stub.PutState("sender1", []byte("500"))

	// Setup: Recipient must exist!
	stub.PutState("recipient1", []byte("0"))

	// 1. Success TransferDebit
	err := sc.TransferDebit(ctx, "recipient1", 100)
	if err != nil {
		t.Fatalf("TransferDebit failed: %v", err)
	}

	// Verify Sender deducted
	bal, _ := stub.GetState("sender1")
	if string(bal) != "400" {
		t.Errorf("Sender balance should be 400, got %s", string(bal))
	}

	// Verify Recipient NOT changed (MVCC logic)
	rBal, _ := stub.GetState("recipient1")
	if string(rBal) != "0" {
		t.Errorf("Recipient balance should remain 0 in Debit step, got %s", string(rBal))
	}

	// Verify Request created
	// Key: transferReq + \x00 + tx001
	// (mock CreateCompositeKey simply joins with \x00)
	reqKey := "transferReq\x00tx001"
	reqBytes, _ := stub.GetState(reqKey)
	if reqBytes == nil {
		t.Fatalf("TransferRequest not found")
	}

	var req TransferRequest
	json.Unmarshal(reqBytes, &req)
	if req.Status != "PENDING" {
		t.Errorf("Expected status PENDING, got %s", req.Status)
	}
	if req.Amount != 100 {
		t.Errorf("Expected amount 100, got %d", req.Amount)
	}

	// Verify Event
	if stub.Events["DebitEvent"] == nil {
		t.Errorf("DebitEvent not emitted")
	}
	// Verify Transfer Event REMOVED
	if stub.Events["Transfer"] != nil {
		t.Errorf("Transfer event SHOULD NOT be emitted in Debit step")
	}
}

func TestTransferDebit_Failures(t *testing.T) {
	sc := new(SmartContract)
	stub := NewMockStub()
	cid := &MockClientIdentity{MSPID: "Org1MSP", ID: "sender1"}
	ctx := &MockContext{mockStub: stub, mockCID: cid}
	stub.PutState(nameKey, []byte("Token"))

	stub.PutState("sender1", []byte("500"))
	// Recipient2 does NOT exist in state

	// 1. Fail: Recipient Not Found
	err := sc.TransferDebit(ctx, "recipient2", 100)
	if err == nil {
		t.Errorf("Expected error for non-existent recipient")
	} else {
		if !strings.Contains(err.Error(), "존재하지 않습니다") {
			t.Errorf("Expected 'not exist' error, got: %v", err)
		}
	}

	// 2. Fail: Insufficient Funds
	stub.PutState("recipient1", []byte("0"))
	err = sc.TransferDebit(ctx, "recipient1", 600)
	if err == nil {
		t.Errorf("Expected error for insufficient funds")
	}
}

func TestTransferFrom_MVCC(t *testing.T) {
	sc := new(SmartContract)
	stub := NewMockStub()
	cid := &MockClientIdentity{MSPID: "Org1MSP", ID: "spender1"}
	ctx := &MockContext{mockStub: stub, mockCID: cid}
	stub.PutState(nameKey, []byte("Token"))

	// Setup
	// Owner: owner1 (Balance 1000)
	// Spender: spender1 (Allowance 500)
	// Recipient: recipient1 (Exists)
	stub.PutState("owner1", []byte("1000"))
	stub.PutState("recipient1", []byte("0"))

	allowanceKey, _ := stub.CreateCompositeKey(allowancePrefix, []string{"owner1", "spender1"})
	stub.PutState(allowanceKey, []byte("500"))

	// Invoke TransferFrom
	err := sc.TransferFrom(ctx, "owner1", "recipient1", 200)
	if err != nil {
		t.Fatalf("TransferFrom failed: %v", err)
	}

	// Verify Owner Deducted
	oBal, _ := stub.GetState("owner1")
	if string(oBal) != "800" { // 1000 - 200
		t.Errorf("Expected owner bal 800, got %s", oBal)
	}

	// Verify Allowance Deducted
	aBal, _ := stub.GetState(allowanceKey)
	if string(aBal) != "300" { // 500 - 200
		t.Errorf("Expected allowance 300, got %s", aBal)
	}

	// Verify Recipient Unchanged
	rBal, _ := stub.GetState("recipient1")
	if string(rBal) != "0" {
		t.Errorf("Recipient bal should allow be 0")
	}

	// Verify Event
	if stub.Events["DebitEvent"] == nil {
		t.Errorf("DebitEvent expected")
	}
}

func TestGetTransferStatus(t *testing.T) {
	sc := new(SmartContract)
	stub := NewMockStub()
	cid := &MockClientIdentity{MSPID: "Org1MSP", ID: "user1"}
	ctx := &MockContext{mockStub: stub, mockCID: cid}
	stub.PutState(nameKey, []byte("Token"))

	// 1. Setup Request
	txID := "tx123"
	req := TransferRequest{Status: "PENDING"}
	reqBytes, _ := json.Marshal(req)
	reqKey, _ := stub.CreateCompositeKey("transferReq", []string{txID})
	stub.PutState(reqKey, reqBytes)

	// 2. Query
	status, err := sc.GetTransferStatus(ctx, txID)
	if err != nil {
		t.Fatalf("GetTransferStatus failed: %v", err)
	}
	if status != "PENDING" {
		t.Errorf("Expected PENDING, got %s", status)
	}
}

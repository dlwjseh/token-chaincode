package chaincode

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// 옵션의 키 이름 정의
const nameKey = "name"
const symbolKey = "symbol"
const decimalsKey = "decimals"
const totalSupplyKey = "totalSupply"

const allowancePrefix = "allowance"

// SmartContract 구조체 정의
type SmartContract struct {
	contractapi.Contract
}

// TransferRequest: 출금 시 생성되는 "송금 대기표"
type TransferRequest struct {
	Sender   string `json:"sender"`   // 보낸 사람
	Receiver string `json:"receiver"` // 받을 사람
	Amount   int    `json:"amount"`   // 금액
	Status   string `json:"status"`   // 상태: "PENDING"(대기중), "COMPLETED"(완료)
}

// event 이벤트 방출을 위한 조직화된 구조체를 제공합니다.
type event struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Value int    `json:"value"`
}

// Mint 새로운 토큰을 발행하고 이를 발행인의 계정 잔액에 추가합니다.
// 이 함수는 Transfer 이벤트를 트리거합니다.
func (s *SmartContract) Mint(ctx contractapi.TransactionContextInterface, amount int) error {
	// 먼저 계약이 초기화되었는지 확인
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return fmt.Errorf("계약 초기화가 잘 되었는지 확인하지 못했습니다.: %v", err)
	}
	if !initialized {
		return errors.New("함수를 호출하기 전에 계약 옵션을 설정해야 합니다. 계약을 초기화하려면 Initialize()를 호출하세요.")
	}

	// 발행인 승인 확인 - 이 샘플에서는 Org1이 새 토큰을 발행할 권한이 있는 중앙 은행원이라고 가정합니다.
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("MSPID를 가져오지 못했습니다.: %v", err)
	}
	if clientMSPID != "Org1MSP" {
		return errors.New("클라이언트는 새 토큰을 발행할 권한이 없습니다.")
	}

	// 발행인 신원을 제출하는 ID 얻기
	minter, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("발행인 ID를 가져오지 못했습니다.: %v", err)
	}
	if amount <= 0 {
		return errors.New("생성 금액은 양의 정수여야 합니다.")
	}

	currentBalanceBytes, err := ctx.GetStub().GetState(minter)
	if err != nil {
		return fmt.Errorf("World State에서 발행인 계정 %s을(를) 읽지 못했습니다: %v", minter, err)
	}

	var currentBalance int

	// 발행인 현재 잔액이 아직 존재하지 않으면 현재 잔액을 0으로 생성합니다.
	if currentBalanceBytes == nil {
		currentBalance = 0
	} else {
		// 계정 잔액을 설정할 때 Itoa()를 사용하여 정수임을 보장하므로 오류 처리가 필요하지 않습니다.
		currentBalance, _ = strconv.Atoi(string(currentBalanceBytes))
	}

	updatedBalance, err := add(currentBalance, amount)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(minter, []byte(strconv.Itoa(updatedBalance)))
	if err != nil {
		return err
	}

	// totalSupply 업데이트
	totalSupplyBytes, err := ctx.GetStub().GetState(totalSupplyKey)
	if err != nil {
		return fmt.Errorf("총 토큰 공급량을 검색하지 못했습니다.: %v", err)
	}

	var totalSupply int

	// 토큰이 발행되지 않은 경우 totalSupply를 초기화합니다.
	if totalSupplyBytes == nil {
		totalSupply = 0
	} else {
		// totalSupply를 설정할 때 Itoa()가 사용되어 정수임을 보장하므로 오류 처리가 필요하지 않습니다.
		totalSupply, _ = strconv.Atoi(string(totalSupplyBytes))
	}

	// 총 공급량에 생성 금액을 추가하고 상태를 업데이트합니다.
	totalSupply, err = add(totalSupply, amount)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(totalSupplyKey, []byte(strconv.Itoa(totalSupply)))
	if err != nil {
		return err
	}

	// Emit the Transfer event
	transferEvent := event{"0x0", minter, amount}
	transferEventJSON, err := json.Marshal(transferEvent)
	if err != nil {
		return fmt.Errorf("JSON 인코딩을 가져오지 못했습니다.: %v", err)
	}
	err = ctx.GetStub().SetEvent("Transfer", transferEventJSON)
	if err != nil {
		return fmt.Errorf("이벤트를 설정하지 못했습니다.: %v", err)
	}

	log.Printf("발행인 계정 %s 잔액이 %d에서 %d(으)로 업데이트되었습니다.", minter, currentBalance, updatedBalance)

	return nil
}

// Burn 발행인의 계정 잔액으로 토큰을 상환합니다.
// 이 함수는 Transfer 이벤트를 트리거합니다.
func (s *SmartContract) Burn(ctx contractapi.TransactionContextInterface, amount int) error {
	// 먼저 계약이 초기화되었는지 확인
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return fmt.Errorf("계약 초기화가 잘 되었는지 확인하지 못했습니다.: %v", err)
	}
	if !initialized {
		return errors.New("함수를 호출하기 전에 계약 옵션을 설정해야 합니다. 계약을 초기화하려면 Initialize()를 호출하세요.")
	}

	// 발행인 승인 확인 - 이 샘플에서는 Org1이 새 토큰을 발행할 권한이 있는 중앙 은행원이라고 가정합니다.
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("MSPID를 가져오지 못했습니다.: %v", err)
	}
	if clientMSPID != "Org1MSP" {
		return errors.New("클라이언트는 새 토큰을 발행할 권한이 없습니다.")
	}

	// 발행인 신원을 제출하는 ID 얻기
	minter, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("발행인 ID를 가져오지 못했습니다.: %v", err)
	}
	if amount <= 0 {
		return errors.New("상환 금액은 양의 정수여야 합니다.")
	}

	currentBalanceBytes, err := ctx.GetStub().GetState(minter)
	if err != nil {
		return fmt.Errorf("World State에서 발행인 계정 %s을(를) 읽지 못했습니다: %v", minter, err)
	}

	var currentBalance int

	// 현재 발행인의 잔액이 있는지 확인.
	if currentBalanceBytes == nil {
		return errors.New("잔액이 존재하지 않습니다.")
	}

	// 계정 잔액을 설정할 때 Itoa()를 사용하여 정수임을 보장하므로 오류 처리가 필요하지 않습니다.
	currentBalance, _ = strconv.Atoi(string(currentBalanceBytes))

	updatedBalance, err := sub(currentBalance, amount)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(minter, []byte(strconv.Itoa(updatedBalance)))
	if err != nil {
		return err
	}

	// totalSupply 업데이트
	totalSupplyBytes, err := ctx.GetStub().GetState(totalSupplyKey)
	if err != nil {
		return fmt.Errorf("총 토큰 공급량을 검색하지 못했습니다: %v", err)
	}

	// 발행된 토큰이 없으면 오류가 발생합니다.
	if totalSupplyBytes == nil {
		return errors.New("총공급이 존재하지 않습니다")
	}

	// totalSupply를 설정할 때 Itoa()가 사용되어 정수임을 보장하므로 오류 처리가 필요하지 않습니다.
	totalSupply, _ := strconv.Atoi(string(totalSupplyBytes))

	// 총 공급량에서 소각량을 빼고 상태를 업데이트합니다.
	totalSupply, err = sub(totalSupply, amount)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(totalSupplyKey, []byte(strconv.Itoa(totalSupply)))
	if err != nil {
		return err
	}

	// Transfer 이벤트를 내보냅니다.
	transferEvent := event{minter, "0x0", amount}
	transferEventJSON, err := json.Marshal(transferEvent)
	if err != nil {
		return fmt.Errorf("JSON 인코딩을 가져오지 못했습니다: %v", err)
	}
	err = ctx.GetStub().SetEvent("Transfer", transferEventJSON)
	if err != nil {
		return fmt.Errorf("이벤트 설정 실패: %v", err)
	}

	log.Printf("발행인 계정 %s 잔액이 %d에서 %d(으)로 업데이트되었습니다.", minter, currentBalance, updatedBalance)

	return nil
}

// TransferDebit
// 이 함수는 Transfer 이벤트를 트리거합니다.
func (s *SmartContract) TransferDebit(ctx contractapi.TransactionContextInterface, recipient string, amount int) error {
	// 먼저 계약이 초기화되었는지 확인
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return fmt.Errorf("계약 초기화가 잘 되었는지 확인하지 못했습니다.: %v", err)
	}
	if !initialized {
		return errors.New("함수를 호출하기 전에 계약 옵션을 설정해야 합니다. 계약을 초기화하려면 Initialize()를 호출하세요.")
	}

	// 클라이언트 ID 얻기
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("클라이언트 ID를 가져오지 못했습니다: %v", err)
	}

	if clientID == recipient {
		return fmt.Errorf("동일한 고객 계정으로 이체할 수 없습니다.")
	}

	if amount < 0 { // ERC-20에서는 0의 전송이 허용되므로 음수 금액에 대해 유효성을 검사합니다.
		return fmt.Errorf("이체 금액은 음수일 수 없습니다.")
	}

	// 클라이언트 계정 잔액 조회
	clientCurrentBalanceBytes, err := ctx.GetStub().GetState(clientID)
	if err != nil {
		return fmt.Errorf("클라이언트 계정 %s을(를) World State에서 읽지 못했습니다: %v", clientID, err)
	}

	// 수신자 계정 존재 여부 확인 (Read-Only verify)
	recipientBalanceBytes, err := ctx.GetStub().GetState(recipient)
	if err != nil {
		return fmt.Errorf("수신자 계정 %s을(를) World State에서 읽지 못했습니다: %v", recipient, err)
	}
	if recipientBalanceBytes == nil {
		return fmt.Errorf("수신자 계정 %s이(가) 존재하지 않습니다.", recipient)
	}

	if clientCurrentBalanceBytes == nil {
		return fmt.Errorf("고객 계정 %s에 잔고가 없습니다", clientID)
	}

	// 계정 잔액을 설정할 때 Itoa()를 사용하여 정수임을 보장하므로 오류 처리가 필요하지 않습니다.
	clientCurrentBalance, _ := strconv.Atoi(string(clientCurrentBalanceBytes))

	if clientCurrentBalance < amount {
		return fmt.Errorf("고객 계정 %s에 자금이 부족합니다.", clientID)
	}

	// 클라이언트 잔액 차감
	clientUpdatedBalance, err := sub(clientCurrentBalance, amount)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(clientID, []byte(strconv.Itoa(clientUpdatedBalance)))
	if err != nil {
		return err
	}

	// "송금 대기표(Request)" 생성
	txID := ctx.GetStub().GetTxID() // 트랜잭션 ID를 고유 키로 사용
	transferReqKey, err := ctx.GetStub().CreateCompositeKey("transferReq", []string{txID})
	if err != nil {
		return fmt.Errorf("송금 대기표 복합 키를 생성하지 못했습니다: %v", err)
	}

	transferRequest := TransferRequest{Sender: clientID, Receiver: recipient, Amount: amount, Status: "PENDING"}
	transferReqBytes, err := json.Marshal(transferRequest)
	if err != nil {
		return fmt.Errorf("JSON 인코딩을 가져오지 못했습니다: %v", err)
	}

	err = ctx.GetStub().PutState(transferReqKey, transferReqBytes) // 대기표 저장
	if err != nil {
		return err
	}

	err = ctx.GetStub().SetEvent("DebitEvent", transferReqBytes)
	if err != nil {
		return fmt.Errorf("송금 대기표 이벤트를 설정하지 못했습니다: %v", err)
	}

	return nil
}

// BalanceOf 해당 계좌의 잔액을 반환합니다.
func (s *SmartContract) BalanceOf(ctx contractapi.TransactionContextInterface, account string) (int, error) {
	// 먼저 계약이 초기화되었는지 확인
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return 0, fmt.Errorf("계약 초기화가 잘 되었는지 확인하지 못했습니다.: %v", err)
	}
	if !initialized {
		return 0, errors.New("함수를 호출하기 전에 계약 옵션을 설정해야 합니다. 계약을 초기화하려면 Initialize()를 호출하세요.")
	}

	balanceBytes, err := ctx.GetStub().GetState(account)
	if err != nil {
		return 0, fmt.Errorf("World State에서 읽지 못했습니다: %v", err)
	}
	if balanceBytes == nil {
		return 0, fmt.Errorf("%s 계정이 존재하지 않습니다.", account)
	}

	// 계정 잔액을 설정할 때 Itoa()를 사용하여 정수임을 보장하므로 오류 처리가 필요하지 않습니다.
	balance, _ := strconv.Atoi(string(balanceBytes))

	return balance, nil
}

// ClientAccountBalance 요청한 고객 계좌의 잔액을 반환합니다.
func (s *SmartContract) ClientAccountBalance(ctx contractapi.TransactionContextInterface) (int, error) {
	// 먼저 계약이 초기화되었는지 확인
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return 0, fmt.Errorf("계약 초기화가 잘 되었는지 확인하지 못했습니다.: %v", err)
	}
	if !initialized {
		return 0, errors.New("함수를 호출하기 전에 계약 옵션을 설정해야 합니다. 계약을 초기화하려면 Initialize()를 호출하세요.")
	}

	// 클라이언트 ID 얻기
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return 0, fmt.Errorf("클라이언트 ID를 가져오지 못했습니다: %v", err)
	}

	balanceBytes, err := ctx.GetStub().GetState(clientID)
	if err != nil {
		return 0, fmt.Errorf("World State에서 읽지 못했습니다: %v", err)
	}
	if balanceBytes == nil {
		return 0, fmt.Errorf("%s 계정이 존재하지 않습니다.", clientID)
	}

	// 계정 잔액을 설정할 때 Itoa()를 사용하여 정수임을 보장하므로 오류 처리가 필요하지 않습니다.
	balance, _ := strconv.Atoi(string(balanceBytes))

	return balance, nil
}

// ClientAccountID 요청한 클라이언트 계정의 ID를 반환합니다.
// 이 구현에서 클라이언트 계정 ID는 clientId 자체입니다.
// 사용자는 이 기능을 사용하여 자신의 계정 ID를 얻을 수 있으며, 이를 다른 사람에게 지불 주소로 제공할 수 있습니다.
func (s *SmartContract) ClientAccountID(ctx contractapi.TransactionContextInterface) (string, error) {
	// 먼저 계약이 초기화되었는지 확인
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return "", fmt.Errorf("계약 초기화가 잘 되었는지 확인하지 못했습니다.: %v", err)
	}
	if !initialized {
		return "", errors.New("함수를 호출하기 전에 계약 옵션을 설정해야 합니다. 계약을 초기화하려면 Initialize()를 호출하세요.")
	}

	// 클라이언트 ID 얻기
	clientAccountID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "", fmt.Errorf("클라이언트 ID를 가져오지 못했습니다: %v", err)
	}

	return clientAccountID, nil
}

// TotalSupply 총 토큰 공급량을 반환합니다.
func (s *SmartContract) TotalSupply(ctx contractapi.TransactionContextInterface) (int, error) {
	// 먼저 계약이 초기화되었는지 확인
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return 0, fmt.Errorf("계약 초기화가 잘 되었는지 확인하지 못했습니다.: %v", err)
	}
	if !initialized {
		return 0, errors.New("함수를 호출하기 전에 계약 옵션을 설정해야 합니다. 계약을 초기화하려면 Initialize()를 호출하세요.")
	}

	// 스마트 계약 상태에서 총 토큰 공급량을 검색합니다.
	totalSupplyBytes, err := ctx.GetStub().GetState(totalSupplyKey)
	if err != nil {
		return 0, fmt.Errorf("총 토큰 공급량을 검색하지 못했습니다: %v", err)
	}

	var totalSupply int

	// 발행된 토큰이 없으면 0을 반환합니다.
	if totalSupplyBytes == nil {
		totalSupply = 0
	} else {
		// totalSupply를 설정할 때 Itoa()가 사용되어 정수임을 보장하므로 오류 처리가 필요하지 않습니다.
		totalSupply, _ = strconv.Atoi(string(totalSupplyBytes))
	}

	log.Printf("총 공급량: %d개의 토큰", totalSupply)

	return totalSupply, nil
}

// Approve 지출자가 호출 클라이언트의 토큰 계정에서 인출할 수 있도록 허용합니다.
// 지출자는 필요한 경우 가치 금액까지 여러 번 인출할 수 있습니다.
// 이 함수는 승인(Approval) 이벤트를 트리거합니다.
func (s *SmartContract) Approve(ctx contractapi.TransactionContextInterface, spender string, value int) error {
	// 먼저 계약이 초기화되었는지 확인
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return fmt.Errorf("계약 초기화가 잘 되었는지 확인하지 못했습니다.: %v", err)
	}
	if !initialized {
		return errors.New("함수를 호출하기 전에 계약 옵션을 설정해야 합니다. 계약을 초기화하려면 Initialize()를 호출하세요.")
	}

	// 클라이언트 ID 얻기
	owner, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("클라이언트 ID를 가져오지 못했습니다: %v", err)
	}

	// 허용키 생성
	allowanceKey, err := ctx.GetStub().CreateCompositeKey(allowancePrefix, []string{owner, spender})
	if err != nil {
		return fmt.Errorf("접두사 %s에 대한 복합 키를 생성하지 못했습니다: %v", allowancePrefix, err)
	}

	// 허용키와 값을 추가하여 스마트 계약 상태를 업데이트합니다.
	err = ctx.GetStub().PutState(allowanceKey, []byte(strconv.Itoa(value)))
	if err != nil {
		return fmt.Errorf("키 %s에 대한 스마트 계약 상태를 업데이트하지 못했습니다: %v", allowanceKey, err)
	}

	// 승인 이벤트 내보내기
	approvalEvent := event{owner, spender, value}
	approvalEventJSON, err := json.Marshal(approvalEvent)
	if err != nil {
		return fmt.Errorf("JSON 인코딩을 가져오지 못했습니다: %v", err)
	}
	err = ctx.GetStub().SetEvent("Approval", approvalEventJSON)
	if err != nil {
		return fmt.Errorf("이벤트를 설정하지 못했습니다: %v", err)
	}

	log.Printf("클라이언트 %s이(가) 지출자 %s에 대해 %d의 인출 허용액을 승인했습니다.", owner, spender, value)

	return nil
}

// Allowance 지출자가 소유자로부터 인출할 수 있는 금액을 반환합니다.
func (s *SmartContract) Allowance(ctx contractapi.TransactionContextInterface, owner string, spender string) (int, error) {
	// 먼저 계약이 초기화되었는지 확인
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return 0, fmt.Errorf("계약 초기화가 잘 되었는지 확인하지 못했습니다.: %v", err)
	}
	if !initialized {
		return 0, errors.New("함수를 호출하기 전에 계약 옵션을 설정해야 합니다. 계약을 초기화하려면 Initialize()를 호출하세요.")
	}

	// 허용키 생성
	allowanceKey, err := ctx.GetStub().CreateCompositeKey(allowancePrefix, []string{owner, spender})
	if err != nil {
		return 0, fmt.Errorf("접두사 %s에 대한 복합 키를 생성하지 못했습니다: %v", allowancePrefix, err)
	}

	// World State에서 허용금액 읽기
	allowanceBytes, err := ctx.GetStub().GetState(allowanceKey)
	if err != nil {
		return 0, fmt.Errorf("World State에서 %s에 대한 허용량을 읽지 못했습니다: %v", allowanceKey, err)
	}

	var allowance int

	// 현재 허용량이 없으면 허용값을 0으로 설정합니다.
	if allowanceBytes == nil {
		allowance = 0
	} else {
		// totalSupply를 설정할 때 Itoa()가 사용되어 정수임을 보장하므로 오류 처리가 필요하지 않습니다.
		allowance, err = strconv.Atoi(string(allowanceBytes))
	}

	log.Printf("지출자 %s이(가) 소유자 %s에게서 인출할 수 있도록 남은 허용량: %d", spender, owner, allowance)

	return allowance, nil
}

// TransferFrom "from" 주소에서 "to" 주소로 금액을 전송합니다.
// 이 함수는 Transfer 이벤트를 트리거합니다.
func (s *SmartContract) TransferFrom(ctx contractapi.TransactionContextInterface, from string, to string, value int) error {
	// 먼저 계약이 초기화되었는지 확인
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return fmt.Errorf("계약 초기화가 잘 되었는지 확인하지 못했습니다.: %v", err)
	}
	if !initialized {
		return errors.New("함수를 호출하기 전에 계약 옵션을 설정해야 합니다. 계약을 초기화하려면 Initialize()를 호출하세요.")
	}

	// 클라이언트 ID 얻기
	spender, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("클라이언트 ID를 가져오지 못했습니다: %v", err)
	}

	if from == to {
		return fmt.Errorf("동일한 고객 계정으로 이체할 수 없습니다.")
	}

	if value < 0 {
		return fmt.Errorf("이체 금액은 음수일 수 없습니다.")
	}

	// 허용키 생성
	allowanceKey, err := ctx.GetStub().CreateCompositeKey(allowancePrefix, []string{from, spender})
	if err != nil {
		return fmt.Errorf("접두사 %s에 대한 복합 키를 생성하지 못했습니다: %v", allowancePrefix, err)
	}

	// 지출자의 허용량을 검색합니다.
	currentAllowanceBytes, err := ctx.GetStub().GetState(allowanceKey)
	if err != nil {
		return fmt.Errorf("World State에서 %s에 대한 허용량을 검색하지 못했습니다: %v", allowanceKey, err)
	}

	var currentAllowance int
	if currentAllowanceBytes == nil {
		currentAllowance = 0
	} else {
		// totalSupply를 설정할 때 Itoa()가 사용되어 정수임을 보장하므로 오류 처리가 필요하지 않습니다.
		currentAllowance, _ = strconv.Atoi(string(currentAllowanceBytes))
	}

	// 이체금액이 허용량보다 적은지 확인
	if currentAllowance < value {
		return fmt.Errorf("지출자에게는 이체할 수 있는 여유가 충분하지 않습니다.")
	}

	// 수신자 계정 존재 여부 확인 (Read-Only verify)
	toBalanceBytes, err := ctx.GetStub().GetState(to)
	if err != nil {
		return fmt.Errorf("수신자 계정 %s을(를) World State에서 읽지 못했습니다: %v", to, err)
	}
	if toBalanceBytes == nil {
		return fmt.Errorf("수신자 계정 %s이(가) 존재하지 않습니다.", to)
	}

	// sender(from) 잔액 조회
	fromCurrentBalanceBytes, err := ctx.GetStub().GetState(from)
	if err != nil {
		return fmt.Errorf("클라이언트 계정 %s을(를) World State에서 읽지 못했습니다: %v", from, err)
	}

	if fromCurrentBalanceBytes == nil {
		return fmt.Errorf("고객 계정 %s에 잔고가 없습니다", from)
	}

	// 계정 잔액을 설정할 때 Itoa()를 사용하여 정수임을 보장하므로 오류 처리가 필요하지 않습니다.
	fromCurrentBalance, _ := strconv.Atoi(string(fromCurrentBalanceBytes))

	if fromCurrentBalance < value {
		return fmt.Errorf("고객 계정 %s에 자금이 부족합니다.", from)
	}

	// sender 잔액 차감
	fromUpdatedBalance, err := sub(fromCurrentBalance, value)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(from, []byte(strconv.Itoa(fromUpdatedBalance)))
	if err != nil {
		return err
	}

	// 허용량 감소
	updatedAllowance, err := sub(currentAllowance, value)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(allowanceKey, []byte(strconv.Itoa(updatedAllowance)))
	if err != nil {
		return err
	}

	// "송금 대기표(Request)" 생성
	txID := ctx.GetStub().GetTxID() // 트랜잭션 ID를 고유 키로 사용
	transferReqKey, err := ctx.GetStub().CreateCompositeKey("transferReq", []string{txID})
	if err != nil {
		return fmt.Errorf("송금 대기표 복합 키를 생성하지 못했습니다: %v", err)
	}

	transferRequest := TransferRequest{Sender: from, Receiver: to, Amount: value, Status: "PENDING"}
	transferReqBytes, err := json.Marshal(transferRequest)
	if err != nil {
		return fmt.Errorf("JSON 인코딩을 가져오지 못했습니다: %v", err)
	}

	err = ctx.GetStub().PutState(transferReqKey, transferReqBytes) // 대기표 저장
	if err != nil {
		return err
	}

	err = ctx.GetStub().SetEvent("DebitEvent", transferReqBytes)
	if err != nil {
		return fmt.Errorf("송금 대기표 이벤트를 설정하지 못했습니다: %v", err)
	}

	log.Printf("소비자 %s 허용량이 %d에서 %d(으)로 업데이트되었습니다.", spender, currentAllowance, updatedAllowance)
	log.Printf("송금자 %s 잔액이 %d에서 %d(으)로 업데이트되었습니다.", from, fromCurrentBalance, fromUpdatedBalance)

	return nil
}

// Name 이 계약에서 대체 가능한 토큰을 설명하는 이름을 반환합니다.
// returns {String} 토큰 이름을 반환합니다.
func (s *SmartContract) Name(ctx contractapi.TransactionContextInterface) (string, error) {
	// 먼저 계약이 초기화되었는지 확인
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return "", fmt.Errorf("계약 초기화가 잘 되었는지 확인하지 못했습니다.: %v", err)
	}
	if !initialized {
		return "", errors.New("함수를 호출하기 전에 계약 옵션을 설정해야 합니다. 계약을 초기화하려면 Initialize()를 호출하세요.")
	}

	bytes, err := ctx.GetStub().GetState(nameKey)
	if err != nil {
		return "", fmt.Errorf("이름 바이트를 가져오지 못했습니다: %s", err)
	}

	return string(bytes), nil
}

// Symbol 이 계약에서 대체 가능한 토큰의 약식 이름을 반환합니다.
// returns {String} 토큰의 약식 이름을 반환합니다.
func (s *SmartContract) Symbol(ctx contractapi.TransactionContextInterface) (string, error) {
	// 먼저 계약이 초기화되었는지 확인
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return "", fmt.Errorf("계약 초기화가 잘 되었는지 확인하지 못했습니다.: %v", err)
	}
	if !initialized {
		return "", errors.New("함수를 호출하기 전에 계약 옵션을 설정해야 합니다. 계약을 초기화하려면 Initialize()를 호출하세요.")
	}

	bytes, err := ctx.GetStub().GetState(symbolKey)
	if err != nil {
		return "", fmt.Errorf("약식 이름을 가져오지 못했습니다: %v", err)
	}

	return string(bytes), nil
}

// 토큰에 대한 정보를 설정하고 계약을 초기화합니다.
// param {String} name 토큰 이름
// param {String} symbol 토큰의 상징
// param {String} decimals 토큰 작업에 사용되는 소수점
func (s *SmartContract) Initialize(ctx contractapi.TransactionContextInterface, name string, symbol string, decimals string) (bool, error) {
	// 발행인 승인 확인 - 이 샘플에서는 Org1이 계약을 초기화할 권한이 있는 중앙 은행원이라고 가정합니다.
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return false, fmt.Errorf("MSPID를 가져오지 못했습니다.: %v", err)
	}
	if clientMSPID != "Org1MSP" {
		return false, fmt.Errorf("클라이언트는 계약을 초기화할 권한이 없습니다.")
	}

	// 계약 옵션이 아직 설정되지 않았는지 확인하세요. 클라이언트는 초기화된 후에 해당 옵션을 변경할 수 있는 권한이 없습니다.
	bytes, err := ctx.GetStub().GetState(nameKey)
	if err != nil {
		return false, fmt.Errorf("이름을 가져오지 못했습니다.: %v", err)
	}
	if bytes != nil {
		return false, fmt.Errorf("계약 옵션이 이미 설정되어 있으며 고객이 이를 변경할 권한이 없습니다.")
	}

	err = ctx.GetStub().PutState(nameKey, []byte(name))
	if err != nil {
		return false, fmt.Errorf("토큰 이름을 설정하지 못했습니다.: %v", err)
	}

	err = ctx.GetStub().PutState(symbolKey, []byte(symbol))
	if err != nil {
		return false, fmt.Errorf("기호를 설정하지 못했습니다.: %v", err)
	}

	err = ctx.GetStub().PutState(decimalsKey, []byte(decimals))
	if err != nil {
		return false, fmt.Errorf("소수점 키를 설정하지 못했습니다.: %v", err)
	}

	return true, nil
}

// GetTransferStatus 트랜잭션 ID를 사용하여 송금 요청의 상태를 반환합니다.
// param {String} txID 트랜잭션 ID
// returns {String} 송금 상태 ("PENDING" 또는 "COMPLETED")를 반환합니다.
func (s *SmartContract) GetTransferStatus(ctx contractapi.TransactionContextInterface, txID string) (string, error) {
	reqKey, _ := ctx.GetStub().CreateCompositeKey("transferReq", []string{txID})
	reqBytes, _ := ctx.GetStub().GetState(reqKey)

	if reqBytes == nil {
		return "", fmt.Errorf("transaction not found")
	}

	var request TransferRequest
	json.Unmarshal(reqBytes, &request)

	return request.Status, nil // "PENDING" or "COMPLETED"
}

// Helper Functions

// 계약 옵션 초기화가 잘 되었는지 확인합니다.
func checkInitialized(ctx contractapi.TransactionContextInterface) (bool, error) {
	tokenName, err := ctx.GetStub().GetState(nameKey)
	if err != nil {
		return false, fmt.Errorf("토큰 이름을 가져오지 못했습니다.: %v", err)
	}

	if tokenName == nil {
		return false, nil
	}

	return true, nil
}

// add 두 개의 숫자를 더합니다.(오버플로우 확인)
func add(b int, q int) (int, error) {
	var sum int
	sum = q + b
	if (sum < q || sum < b) == (b >= 0 && q >= 0) {
		return 0, fmt.Errorf("Math: 오버플로가 %d + %d 발생했습니다.", b, q)
	}
	return sum, nil
}

// sub 두 개의 숫자를 뺍니다.(b-q)
func sub(b int, q int) (int, error) {
	// 빼는 두 숫자 확인
	if q <= 0 {
		return 0, fmt.Errorf("오류: 뺄셈 숫자가 %d입니다. 0보다 커야 합니다.", q)
	}
	if b < q {
		return 0, fmt.Errorf("오류: 숫자 %d은(는) %d에서 빼기에는 충분하지 않습니다.", b, q)
	}
	var diff int
	diff = b - q
	return diff, nil
}

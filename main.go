package main

import (
	"log"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
	"github.com/jd/token-chaincode/chaincode"
)

func main() {
	tokenChaincode, err := contractapi.NewChaincode(&chaincode.SmartContract{})
	if err != nil {
		log.Panicf("자산 전송 기본 체인코드 생성 중 오류 발생: %v", err)
	}

	if err := tokenChaincode.Start(); err != nil {
		log.Panicf("자산 전송 기본 체인코드 시작 중 오류 발생: %v", err)
	}
}

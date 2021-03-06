/**
@author: Arshad Sarfarz
@version: 1.0.0
@date: 10/04/2017
@Description: MedLab-Pharma chaincode v1
**/

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

const STATUS_SHIPPED = "shipped"
const UNIQUE_ID_COUNTER string = "UniqueIDCounter"
const CONTAINER_OWNER = "ContainerOwner"

type MedLabPharmaChaincode struct {
}

type UniqueIDCounter struct {
	ContainerMaxID int `json:"ContainerMaxID"`
	PalletMaxID    int `json:"PalletMaxID"`
}
type Shipment struct{
	ContainerList []Container `json:"container_list"`

}

type Container struct {
	ContainerId       string              `json:"container_id"`
	ParentContainerId string              `json:"parent_container_id"`
	ChildContainerId  []string            `json:"child_container_id"`
	Recipient         string              `json:"logistics_id"`
	Elements          ContainerElements   `json:"elements"`
	Provenance        ContainerProvenance `json:"provenance"`
	CertifiedBy       string              `json:"certified_by"`   ///New fields
	Address           string              `json:"address"`        ///New fields
	USN               string              `json:"usn"`            ///New fields
	ShipmentDate      string              `json:"shipment_date"`  ///New fields
	InvoiceNumber     string              `json:"invoice_number"` ///New fields
	Remarks           string              `json:"remarks"`        ///New fields
}

type ContainerElements struct {
	Pallets []Pallet `json:"pallets"`
}

type Pallet struct {
	PalletId string `json:"pallet_id"`
	Cases    []Case `json:"cases"`
}

type Case struct {
	CaseId string `json:"case_id"`
	Units  []Unit `json:"units"`
}

type Unit struct {
	DrugId       string `json:"drug_id"`
	DrugName     string `json:"drug_name"` ////New Fields
	UnitId       string `json:"unit_id"`
	ExpiryDate   string `json:"expiry_date"`
	HealthStatus string `json:"health_status"`
	BatchNumber  string `json:"batch_number"`
	LotNumber    string `json:"lot_number"`
	SaleStatus   string `json:"sale_status"`
	ConsumerName string `json:"consumer_name"`
}

type ContainerProvenance struct {
	TransitStatus string          `json:transit_status`
	Sender        string          `json:sender`
	Receiver      string          `json:receiver`
	Supplychain   []ChainActivity `json:supplychain`
}

type ChainActivity struct {
	Sender   string `json:sender`
	Receiver string `json:receiver`
	Status   string `json:transit_status`
}

type ContainerOwners struct {
	Owners []Owner `json:owners`
}

type Owner struct {
	OwnerId       string   `json:owner_id`
	ContainerList []string `json:container_id`
}

func main() {
	fmt.Println("Inside MedLabPharmaChaincode main function")
	err := shim.Start(new(MedLabPharmaChaincode))
	if err != nil {
		fmt.Printf("Error starting MedLabPharma chaincode: %s", err)
	}
}

// Init resets all the things
func (t *MedLabPharmaChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	// Handle different functions
	if function == "init" {
		return t.init(stub, args)
	}
	fmt.Println("invoke did not find func: " + function)

	return nil, errors.New("Received unknown function invocation: " + function)
}

// Invoke isur entry point to invoke a chaincode function
func (t *MedLabPharmaChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "ShipContainerUsingLogistics" {
		return t.ShipContainerUsingLogistics(stub, args[0], args[1], args[2], args[3], args[4])
	} else if function == "SetCurrentOwner"{
		return t.SetCurrentOwnerTest(stub, args[0], args[1])
	}
	 
	fmt.Println("invoke did not find func: " + function)

	return nil, errors.New("Received unknown function invocation: " + function)
}

// Query is our entry point for queries
func (t *MedLabPharmaChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

	// Handle different functions
	if function == "GetContainerDetails" { //read a variable
		return t.GetContainerDetails(stub, args[0])
	} else if function == "GetMaxIDValue" {
		return t.GetMaxIDValue(stub)
	} else if function == "GetEmptyContainer" {
		return t.GetEmptyContainer(stub)
	}  else if function == "getContainerDetailsforowner" {
		return t.getContainerDetailsforowner(stub, args[0])
	}
	
	fmt.Println("query did not find func: " + function)
	return nil, errors.New("Received unknown function query: " + function)
}

func (t *MedLabPharmaChaincode) init(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	maxIDCounter := UniqueIDCounter{
		ContainerMaxID: 0,
		PalletMaxID:    0}
	jsonVal, _ := json.Marshal(maxIDCounter)
	err := stub.PutState(UNIQUE_ID_COUNTER, []byte(jsonVal))
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// write - invoke function to write key/value pair
func (t *MedLabPharmaChaincode) ShipContainerUsingLogistics(stub shim.ChaincodeStubInterface,
	senderID string, logisticsID string, receiverID string, remarks string, elementsJSON string) ([]byte, error) {
	var err error

	containerID, jsonValue := ShipContainerUsingLogistics_Internal(senderID, logisticsID, receiverID, remarks, elementsJSON)
	fmt.Println("running ShipContainerUsingLogistics.key:" + containerID)
	fmt.Println(jsonValue)
	err = stub.PutState(containerID, jsonValue) //write the variable into the chaincode state

	incrementCounter(stub) //increment the unique ids for container and Pallet

	setCurrentOwner(stub, senderID, containerID)

	if err != nil {
		return nil, err
	}
	return nil, nil
}

// read - query function to read key/value pair
func (t *MedLabPharmaChaincode) GetContainerDetails(stub shim.ChaincodeStubInterface, container_id string) ([]byte, error) {
	fmt.Println("runnin GetContainerDetails ")
	var key, jsonResp string
	var err error

	if container_id == "" {
		return nil, errors.New("Incorrect number of arguments. Expecting name of the key to query")
	}

	fmt.Println("key:" + container_id)
	valAsbytes, err := stub.GetState(container_id)
	fmt.Println(valAsbytes)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + key + "\"}"
		return nil, errors.New(jsonResp)
	}

	return valAsbytes, nil
}

//Returns the maximum number used for ContainerID and PalletID in the format "ContainerMaxNumber, PalletMaxNumber"
func (t *MedLabPharmaChaincode) GetMaxIDValue(stub shim.ChaincodeStubInterface) ([]byte, error) {
	var jsonResp string
	var err error
	ConMaxAsbytes, err := stub.GetState(UNIQUE_ID_COUNTER)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for ContainerMaxNumber \"}"
		return nil, errors.New(jsonResp)
	}
	return ConMaxAsbytes, nil
}

func (t *MedLabPharmaChaincode) GetEmptyContainer(stub shim.ChaincodeStubInterface) ([]byte, error) {
	ConMaxAsbytes, err := stub.GetState(UNIQUE_ID_COUNTER)
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for ContainerMaxNumber \"}"
		return nil, errors.New(jsonResp)
	}

	counter := UniqueIDCounter{}
	json.Unmarshal([]byte(ConMaxAsbytes), &counter)
	containerID := "CON" + strconv.Itoa(counter.ContainerMaxID+1)
	pallets := createPallet(containerID, counter.PalletMaxID+1)
	conelement := ContainerElements{Pallets: pallets}
	container := Container{
		ContainerId: containerID,
		Elements:    conelement}
	jsonVal, _ := json.Marshal(container)
	return jsonVal, nil
}

func ShipContainerUsingLogistics_Internal(senderID string,
	logisticsID string, receiverID string, remarks string, elementsJSON string) (string, []byte) {
	chainActivity := ChainActivity{
		Sender:   senderID,
		Receiver: logisticsID,
		Status:   STATUS_SHIPPED}
	var supplyChain []ChainActivity
	supplyChain = append(supplyChain, chainActivity)
	conprov := ContainerProvenance{
		TransitStatus: STATUS_SHIPPED,
		Sender:        senderID,
		Receiver:      logisticsID,
		Supplychain:   supplyChain}
	shipment := Container{}
	json.Unmarshal([]byte(elementsJSON), &shipment)
	shipment.Recipient = receiverID
	shipment.Provenance = conprov
	jsonVal, _ := json.Marshal(shipment)
	return shipment.ContainerId, jsonVal
}

func createUnit(caseID string) []Unit {
	units := make([]Unit, 3)

	for index := 0; index < 3; index++ {
		strIndex := strconv.Itoa(index + 1)
		unitid := caseID + "-UNIT" + strIndex
		units[index].UnitId = unitid
	}
	return units
}

func createCase(palletID string) []Case {
	cases := make([]Case, 3)

	for index := 0; index < 3; index++ {
		strIndex := strconv.Itoa(index + 1)
		caseid := palletID + "-CASE" + strIndex
		cases[index].CaseId = caseid
		cases[index].Units = createUnit(caseid)
	}
	return cases
}

func createPallet(containerID string, palletMaxID int) []Pallet {
	pallets := make([]Pallet, 3)
	for index := 0; index < 3; index++ {
		strMaxID := strconv.Itoa(palletMaxID)
		palletid := containerID + "-PAL" + strMaxID
		pallets[index].PalletId = palletid
		pallets[index].Cases = createCase(palletid)
		palletMaxID++
	}
	return pallets
}

func incrementCounter(stub shim.ChaincodeStubInterface) error {
	ConMaxAsbytes, err := stub.GetState(UNIQUE_ID_COUNTER)
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for ContainerMaxNumber \"}"
		return errors.New(jsonResp)
	}
	counter := UniqueIDCounter{}
	json.Unmarshal([]byte(ConMaxAsbytes), &counter)
	counter.ContainerMaxID = counter.ContainerMaxID + 1
	counter.PalletMaxID = counter.PalletMaxID + 3
	jsonVal, _ := json.Marshal(counter)
	err = stub.PutState(UNIQUE_ID_COUNTER, []byte(jsonVal))
	if err != nil {
		return err
	}
	return nil
}

func (t *MedLabPharmaChaincode) SetCurrentOwnerTest(stub shim.ChaincodeStubInterface, ownerID string, containerID string) ([]byte, error) {
	err := setCurrentOwner(stub, ownerID, containerID)
	return []byte("success"), err
}

func (t *MedLabPharmaChaincode) getContainerDetailsforowner(stub shim.ChaincodeStubInterface, ownerID string) ([]byte, error) {
	ConMaxAsbytes, err := stub.GetState(CONTAINER_OWNER)
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for ContainerMaxNumber \"}"
		return nil, errors.New(jsonResp)
	}
	ConOwners := ContainerOwners{}
	json.Unmarshal([]byte(ConMaxAsbytes), &ConOwners)

    fmt.Println("owne1r= "+ConOwners.Owners[0].OwnerId)
	var containerList []string
	//var result [][]byte
	var matchFound bool
    fmt.Println("owne2r= "+ConOwners.Owners[0].OwnerId)
	 
	for index := range ConOwners.Owners {
          fmt.Println("owne3r= "+ConOwners.Owners[0].OwnerId + " inp="+ownerID)
	if ConOwners.Owners[index].OwnerId == ownerID {
	        fmt.Println("owne4r= "+ConOwners.Owners[index].OwnerId)
	       containerList = ConOwners.Owners[index].ContainerList
			
	        matchFound = true
			break
		}
	}
    fmt.Println("Container List " + containerList[0])
	 if matchFound {
     fmt.Println("match found ")
     shipment :=Shipment{}
	 for _,containerID := range containerList{
		fmt.Println(containerID)
		byteVal,_ :=  t.GetContainerDetails(stub,  containerID)
	    container := Container{}
	    json.Unmarshal([]byte(byteVal), &container)
        fmt.Println(container)
        fmt.Println("shipment= "+container.ContainerId)
        shipment.ContainerList = append(shipment.ContainerList, container)
         // fmt.Println("shipment after appending= "+shipment)
         // result= append(result,byteVal)
	 }
	    jsonVal, _:= json.Marshal(shipment)
	    return jsonVal, nil
		
	}   
    return nil, nil
	
	//result = []{"CON01","CON02","CON03"}
		//jsonVal, _ = json.Marshal(result)
	
}

func setCurrentOwner(stub shim.ChaincodeStubInterface, ownerID string, containerID string) error {
	ConMaxAsbytes, err := stub.GetState(CONTAINER_OWNER)
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for ContainerMaxNumber \"}"
		return errors.New(jsonResp)
	}
	ConOwners := ContainerOwners{}
	json.Unmarshal([]byte(ConMaxAsbytes), &ConOwners)

	var containerList []string
	var ownerIndex int
	var matchFound bool
	for index := range ConOwners.Owners {
		if ConOwners.Owners[index].OwnerId == ownerID {
			ownerIndex = index
			containerList = ConOwners.Owners[index].ContainerList
			matchFound = true
			break
		}
	}
	containerFound := false
	if matchFound {
		for index := range containerList {
			if containerList[index] == containerID {
				containerFound = true
				break
			}
		}
		if !containerFound {
			containerList = append(containerList, containerID)
			ConOwners.Owners[ownerIndex].ContainerList = containerList
		}
	} else {
		containerList := make([]string, 1)
		containerList[0] = containerID
		owner := Owner{OwnerId: ownerID, ContainerList: containerList}
		ConOwners.Owners = append(ConOwners.Owners, owner)
	}

	jsonVal, _ := json.Marshal(ConOwners)
	err = stub.PutState(CONTAINER_OWNER, []byte(jsonVal))
	if err != nil {
		return err
	}

	return nil
}

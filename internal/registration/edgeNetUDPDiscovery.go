package registration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/node"
	"github.com/grussorusso/serverledge/utils"
)

// UDPStatusServer listen for incoming request from other edge-nodes which want to retrieve the status of this server
// this listener should be called asynchronously in the main function
func UDPStatusServer() {
	hostname := config.GetString(config.API_IP, utils.GetIpAddress().String())
	port := config.GetInt(config.LISTEN_UDP_PORT, 9876)
	address := fmt.Sprintf("%s:%d", hostname, port)
	udpAddr, err := net.ResolveUDPAddr("udp", address)

	if err != nil {
		log.Fatal(err)
	}
	// setup listener for incoming UDP connection
	udpConn, err := net.ListenUDP("udp", udpAddr)

	if err != nil {
		log.Fatal(err)
	}
	log.Printf("UDP server up and listening on port %d", port)
	defer udpConn.Close()

	for {
		// wait for UDP client to connect
		handleUDPConnection(udpConn)
	}

}

func handleUDPConnection(conn *net.UDPConn) {
	buffer := make([]byte, 1024)

	_, addr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		return
	}
	//retrieve the current status
	msgStatus, err := getCurrentStatusInformation()
	if err != nil {
		log.Println(err)
		msgStatus = []byte("")
	}
	//send the infos back to the client edge-node
	_, err = conn.WriteToUDP(msgStatus, addr)
	if err != nil {
		log.Println(err)
	}
}

func getCurrentStatusInformation() (status []byte, err error) {
	portNumberApi := config.GetInt("api.port", 1323)
	portNumberReg := config.GetInt("registry.udp.port", 9876)
	hostname := config.GetString(config.API_IP, utils.GetIpAddress().String())
	urlApi := fmt.Sprintf("http://%s:%d", hostname, portNumberApi)
	urlReg := fmt.Sprintf("http://%s:%d", hostname, portNumberReg)

	addr := NodeInterfaces{
		NodeAddress:     urlApi,
		RegistryAddress: urlReg,
	}

	response := StatusInformation{
		Addresses:               addr,
		AvailableWarmContainers: node.WarmStatus(),
		AvailableMemMB:          node.Resources.AvailableMemMB,
		AvailableCPUs:           node.Resources.AvailableCPUs,
		DropCount:               node.Resources.DropRequestsCount,
		Coordinates:             *Reg.Client.GetCoordinate(),
	}

	return json.Marshal(response)

}

// statusInfoRequest sends a request to the local registry of a node
func statusInfoRequest(hostname string, port string) (info *StatusInformation, duration time.Duration) {
	// Construct the address of the local registry of the target node
	address := fmt.Sprintf("%s:%s", hostname, port)

	remoteAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		log.Printf("Unreachable server %s", address)
		return nil, 0
	}

	udpConn, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		log.Println(err)
		return nil, 0
	}
	defer udpConn.Close()

	// write a message to server, here 1 byte is enough
	message := []byte("A")
	sendingTime := time.Now()
	_, err = udpConn.Write(message)
	if err != nil {
		log.Println(err)
		return nil, 0
	}

	// receive message from server
	buffer := make([]byte, 1024)
	_, _, err = udpConn.ReadFromUDP(buffer)
	if err != nil {
		log.Println(err)
		return nil, 0
	}

	rtt := time.Now().Sub(sendingTime)
	buffer = bytes.Trim(buffer, "\x00")
	//unmarshal result
	var result StatusInformation
	err = json.Unmarshal(buffer, &result)
	if err != nil {
		fmt.Println("Can not unmarshal JSON")
		return nil, 0
	}

	return &result, rtt
}

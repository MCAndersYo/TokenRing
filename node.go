package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	gRPC "github.com/MCAndersYo/TokenRing/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	Wanted   string = "wanted"
	Held            = "held"
	Released        = "released"
)

type node struct {
	gRPC.UnimplementedTemplateServer
	listenPort      string
	nodeID          int32
	channels        map[string]chan gRPC.Reply
	lamport         int32
	nodeSlice       []nodeConnection
	mutex           sync.Mutex
	state           string
	requests        []string
	lamportRequest  int32
	repliesReceived chan int
}
type nodeConnection struct {
	node     gRPC.TemplateClient
	nodeConn *grpc.ClientConn
}

var nodeName = flag.Int("name", 0, "Senders name")
var port = flag.String("port", "5400", "Listen port")

func (n *node) ConnectToNode(port string) {

	n.channels[port] = make(chan gRPC.Reply)
	//dial options
	//the server is not using TLS, so we use insecure credentials
	//(should be fine for local testing but not in the real world)
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))

	//use context for timeout on the connection
	timeContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel() //cancel the connection when we are done

	//dial the server to get a connection to it
	log.Printf("client %v: Attempts to dial on port %v\n", n.nodeID, port)
	// Insert your device's IP before the colon in the print statement
	conn, err := grpc.DialContext(timeContext, fmt.Sprintf(":%s", port), opts...)
	if err != nil {
		log.Printf("Fail to Dial : %v", err)
		return
	}

	// makes a client from the server connection and saves the connection
	// and prints rather or not the connection was is READY
	nodeConnection := nodeConnection{
		node:     gRPC.NewTemplateClient(conn),
		nodeConn: conn,
	}
	n.nodeSlice = append(n.nodeSlice, nodeConnection)
	log.Println("the connection is: ", conn.GetState().String())
}

func (n *node) launchNode() {
	log.Printf("Server %v: Attempts to create listener on port %v\n", n.nodeID, n.listenPort)

	// Create listener tcp on given port or default port 5400
	// Insert your device's IP before the colon in the print statement
	list, err := net.Listen("tcp", fmt.Sprintf(":%s", n.listenPort))
	if err != nil {
		log.Printf("Server %v: Failed to listen on port %v: %v", n.nodeID, n.listenPort, err) //If it fails to listen on the port, run launchServer method again with the next value/port in ports array
		return
	}

	// makes gRPC server using the options
	// you can add options here if you want or remove the options part entirely
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	gRPC.RegisterTemplateServer(grpcServer, n) //Registers the server to the gRPC server.

	log.Printf("Server %v: Listening on port %v\n", n.nodeID, n.listenPort)

	if err := grpcServer.Serve(list); err != nil {
		log.Fatalf("failed to serve %v", err)
	}
	// code here is unreachable because grpcServer.Serve occupies the current thread.
}

func (n *node) parseInput() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("--------------------")

	//Infinite loop to listen for clients input.
	for {
		fmt.Print("-> ")

		//Read input into var input and any errors into err
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		input = strings.TrimSpace(input) //Trim input
		if strings.Contains(input, "connect") {
			portString := input[8:12]
			if err != nil {
				// ... handle error
				panic(err)
			}
			n.ConnectToNode(portString)
		}
		continue
	}
}

func main() {
	//parse flag/arguments
	flag.Parse()

	fmt.Println("--- CLIENT APP ---")

	//log to file instead of console
	//setLog()

	//connect to server and close the connection when program closes
	fmt.Println("--- join Server ---")

	node := node{
		listenPort:      *port,
		nodeID:          int32(*nodeName),
		nodeSlice:       make([]nodeConnection, 0),
		channels:        make(map[string]chan gRPC.Reply),
		state:           Released,
		lamport:         1,
		lamportRequest:  1,
		repliesReceived: make(chan int),
	}
	go node.launchNode()
	go node.parseInput()
	for {
		time.Sleep(5 * time.Second)
	}
}

func (n *node) incrementLamport() {
	n.mutex.Lock()
	n.lamport++
	n.mutex.Unlock()
}
func (n *node) setlamportMax(lamport int32) {
	n.mutex.Lock()
	if n.lamport < lamport {
		n.lamport = lamport
	}
	n.mutex.Unlock()

}

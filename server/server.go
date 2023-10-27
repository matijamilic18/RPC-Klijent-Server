package main

import (
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"sync"
	"time"
)

//struct nad kojim cemo pozivati metode
/*
type API struct{
	Values map[string] int;
}


type SetRequest struct {
	Str string
	Val int
}


func (api *API) Set (request SetRequest, reply *int) error{
	key := request.Str
	val := request.Val

	api.Values[key] = val
	*reply= val

	return nil
}



func (api *API) Get (key string, reply *int) error{

	val, exists := api.Values[key]

	if !exists{
		fmt.Println("Kljuc ne postoji")
	}
	*reply=val
	return nil
}
*/

type Server struct{
	ID int
	RPCServer *rpc.Server
	listener net.Listener
	peers []*rpc.Client
	wg *sync.WaitGroup
	mu sync.Mutex
}

func (s *Server) Call(id int, serviceMethod string, args interface{}, reply interface{}) error {
	s.mu.Lock()
	peer := s.peers[id]
	s.mu.Unlock()

	// If this is called after shutdown (where client.Close is called), it will
	// return an error.
	if peer == nil {
		return fmt.Errorf("call client %d after it's closed", id)
	} else {
		return peer.Call(serviceMethod, args, reply)
	}
}


type HeartbeatArgs struct{
	ID int
}


type HeartbeatReply struct{
	Reply string
}

func (s* Server) Heartbeat (args* HeartbeatArgs, reply* HeartbeatReply) error{
	fmt.Printf("Server %d salje poruku serveru %d\n", args.ID,s.ID)
	reply.Reply = "Dobijena poruka"
	return nil
}

func (s *Server) SendHeartbeats (){
	for{
		for targetID:= 0; targetID<5;targetID++{
			if s.peers[targetID] == nil || targetID==s.ID{
				continue
			}
			arguments := &HeartbeatArgs{s.ID}
			var reply HeartbeatReply

			err := s.peers[targetID].Call("Server.ReceiveHeartbeat", arguments,&reply)
			if err!=nil{
				fmt.Printf("Greska pri slanju")
			}

		}
		//povecano vreme timeout-a da bi se lepse videlo u terminalu
		time.Sleep(time.Duration(1500+rand.Intn(1510)) * time.Millisecond)

	}
}

func (s *Server) ReceiveHeartbeat(message HeartbeatArgs,  reply* HeartbeatReply) error {
	
	reply.Reply = fmt.Sprintf("Server %d je dobio poruku od Servera %d\n", s.ID, message.ID)
	fmt.Printf(reply.Reply)


	return nil
}

func (s *Server) run (wg *sync.WaitGroup){
	s.wg=wg
	wg.Add(1)
	s.peers = make ([]*rpc.Client, 5)
	s.RPCServer = rpc.NewServer()
	s.RPCServer.Register(s)
	listener , err := net.Listen("tcp" , fmt.Sprintf("localhost:1234%d", s.ID))
	
	if err!= nil{
		fmt.Printf("Greska pri slusanju: %s", err)
	}
	
	s.listener=listener
	


	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Greska pri slusanju: %s", err)
			continue
		}
		go s.RPCServer.ServeConn(conn)
	}

}


func main(){

	var waitG sync.WaitGroup
	
	serverList := make([]*Server, 5)

	//pokretanje servera
	for i := 0; i < 5; i++ {
		serverList[i] = &Server{ID: i}
		go serverList[i].run(&waitG)
	}

	//povezivanje servera medjusobno
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			if i != j {
				client, err := rpc.Dial("tcp", fmt.Sprintf("localhost:1234%d", j))
				if err != nil {
					fmt.Printf("Server %d nije uspeo da se poveze sa serverom %d: %s\n", i, j, err)
				} else {
					serverList[i].peers[j] = client
				}
			}
		}
	}

	//slanje keep-alive poruka
	for i := 0; i < 5; i++ {
		go serverList[i].SendHeartbeats()
	}

	waitG.Wait()

	/*service := &API{
		Values: make(map[string]int),
	}
	
	rpc.Register(service)


	listener, err :=net.Listen ("tcp", ":1234")
	if err!=nil{
		log.Fatal("Greska pri pokretanju servera:", err)
	}

	fmt.Println("Server pokrenut")
	for {
		conn, err:= listener.Accept()
		if err!=nil{
			log.Fatal("Accept error:", err)
		}
		go rpc.ServeConn(conn)
	}
	*/
}
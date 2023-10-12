package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
)

//struct nad kojim cemo pozivati metodu sabiranja
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


func main(){
	service := &API{
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
	
}
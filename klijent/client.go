package main

import (
	"fmt"
	"net/rpc"
)

type SetRequest struct {
	Str string
	Val int
}


func main() {
	client, err := rpc.Dial("tcp", "localhost:1234")
	if err != nil {
		fmt.Println("Dial error:", err)
		return
	}
	for {
		var input1 int
		fmt.Printf("\nMENI \n 1.Dodavanje (azuriranje) vrednosti\n 2.Uzimanje vrednosti\n 0.Izlaz\n")
		fmt.Scanln(&input1);
		switch(input1){
		case 1:
			var str string
			var broj int
			fmt.Printf("Vrednost koju ubacujete (int):")
			fmt.Scanln(&broj);
			fmt.Printf("Njen kljuc(string):")
			fmt.Scanln(&str);
			request := SetRequest{Str:str,Val:broj};
			var reply int
			err = client.Call("API.Set",request,&reply)
			if err!=nil{
				fmt.Println("Call error:", err)
				return
			}
			fmt.Printf("Uspesno ste dodali vrednost %d", reply);
		case 2:
			var str string
			fmt.Printf("kljuc po kome trazite(string):")
			fmt.Scanln(&str);
			var reply int 
			err = client.Call("API.Get",str,&reply)
			if err!=nil{
				fmt.Println("Call error:", err)
				return
			}
			fmt.Printf("Vrednost koju ste trazili je %d", reply)
		case 0:
			return
		default:
			continue
		}
		
	}
	
}
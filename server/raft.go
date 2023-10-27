package main

import (
	"math/rand"
	"sync"
	"time"
)


const (
	Follower int= 1
	Candidate int= 2
	Leader int= 3
	Dead int= 0
)

type LogEntry struct{
	Command interface{}
	Term    int
}

type RAFTModule struct{
	//ID servera koji poseduje modul
	id int
	//ID ostalih servera u clusteru
	peerIDs []int
	//server koji poseduje modul
	server *Server

	mu sync.Mutex

	// stanje modula
	currentTerm int
	votedFor    int
	log         []LogEntry


	state int
	electionResetEvent time.Time

}


func NewRAFTModule(id int, peerIds []int, server *Server, ready <-chan interface{}) *RAFTModule {
	rm := new(RAFTModule)
	rm.id = id
	rm.peerIDs = peerIds
	rm.server = server
	rm.state = Follower
	rm.votedFor = -1

	go func() {
		
		<-ready
		rm.mu.Lock()
		rm.electionResetEvent = time.Now()
		rm.mu.Unlock()
		rm.runElectionTimer()
	}()

	return rm
}

func (rm *RAFTModule) electionTimeout() time.Duration {
	
	return time.Duration(150+rand.Intn(150)) * time.Millisecond
	
}

func (rm* RAFTModule) runElectionTimer(){
	timeoutDuration := rm.electionTimeout()
  	rm.mu.Lock()
  	termStarted := rm.currentTerm
  	rm.mu.Unlock()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		<-ticker.C
	
		rm.mu.Lock()
		if rm.state != Candidate && rm.state != Follower {
		  rm.mu.Unlock()
		  return
		}
		//da ne bi bilo vise followera sa jedne masine
		if termStarted != rm.currentTerm {
		  rm.mu.Unlock()
		  return
		}
	
		// pocinjemo izbore ako nismo culi od lidera za vreme celog timeout-a
		if elapsed := time.Since(rm.electionResetEvent); elapsed >= timeoutDuration {
		  rm.startElection()
		  rm.mu.Unlock()
		  return
		}
		rm.mu.Unlock()
	}

}

type RequestVoteArgs struct {
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

func (rm *RAFTModule) startElection(){
	rm.state = Candidate
  	rm.currentTerm += 1
  	savedCurrentTerm := rm.currentTerm
  	rm.electionResetEvent = time.Now()
  	rm.votedFor = rm.id
  

  	votesReceived := 1


	for _, peerId := range rm.peerIDs{
		go func (peerId int){
			args:= RequestVoteArgs{
				Term: savedCurrentTerm,
				CandidateId: rm.id,
			}
			var reply RequestVoteReply

			if err:=rm.server.Call(peerId,"RAFTModule.RequestVote",args,&reply);err == nil{
				rm.mu.Lock()
        		defer rm.mu.Unlock()
        		
				//u slucaju da modul prestane da bude kandidat dok ceka odgovor
        		if rm.state != Candidate {
					return
        		}

				//u slucaju da kandidat ima zastareli (outdated) term 
				if reply.Term > savedCurrentTerm {
					rm.becomeFollower(reply.Term)
					return
				}else if reply.Term == savedCurrentTerm{
					if reply.VoteGranted == true{
						votesReceived++
						if votesReceived*2 > len(rm.peerIDs)+1 {
						  // Dobio izbore
						  
						  rm.startLeader()
						  return
						}
					  }
				}

			}

		}(peerId)

		
	}

	//u slucaju da su izbori bili neuspesni
	go rm.runElectionTimer()
}

func (rm *RAFTModule) becomeFollower(term int) {
	rm.state = Follower
	rm.currentTerm = term
	rm.votedFor = -1
	rm.electionResetEvent = time.Now()

	go rm.runElectionTimer()
}

func (rm *RAFTModule) startLeader(){
	rm.state = Leader
	
  	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		// Salje heartbeat-e dok je leader
		for {
		<-ticker.C
		rm.leaderSendHeartbeats()

		rm.mu.Lock()
		if rm.state != Leader {
			rm.mu.Unlock()
			return
		}
		rm.mu.Unlock()
		}
  	}()
}

type AppendEntriesArgs struct {
	Term     int
	LeaderID int

	
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}


func (rm *RAFTModule) leaderSendHeartbeats (){
	rm.mu.Lock()
	savedCurrentTerm := rm.currentTerm
	rm.mu.Unlock()

	for _, peerID := range rm.peerIDs{

		args := AppendEntriesArgs{
			Term: savedCurrentTerm,
			LeaderID:	rm.id,
		}

		go func (peerID int){
			var reply AppendEntriesReply

			if err:= rm.server.Call(peerID, "RAFTModule.AppendEntries", args, &reply); err==nil{

				rm.mu.Lock()
				defer rm.mu.Unlock()

				if reply.Term > savedCurrentTerm{
					rm.becomeFollower(reply.Term)
				}
			}

		}(peerID)
	}

}

func (rm *RAFTModule) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) error{
	rm.mu.Lock()
	defer rm.mu.Unlock()
	// Mrtvi cvorovi ne mogu da glasaju
	if rm.state == Dead{
		return nil
	}
	//provera za term
	if args.Term > rm.currentTerm{
		rm.becomeFollower(args.Term)
	}

	if rm.currentTerm == args.Term && rm.votedFor == -1{
		reply.VoteGranted = true
		rm.votedFor = args.CandidateId
		rm.electionResetEvent = time.Now()
	}else{
		reply.VoteGranted = false
	}
	reply.Term = rm.currentTerm
	return nil
}


func (rm *RAFTModule) AppendEntries (args AppendEntriesArgs, reply *AppendEntriesReply) error{
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	if rm.state == Dead{
		return nil
	}

	if args.Term > rm.currentTerm{
		rm.becomeFollower(args.Term)
	}

	reply.Success= false
	if args.Term == rm.currentTerm {
		if rm.state != Follower{
			rm.becomeFollower(args.Term)
		}
		rm.electionResetEvent = time.Now()
		reply.Success = true
	}
	reply.Term = rm.currentTerm
	return nil
}



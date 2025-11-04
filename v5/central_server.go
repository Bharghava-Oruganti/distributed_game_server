package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"
)

var (
	zone        map[ChunkID]string
	zoneMu      sync.Mutex
	serversList = []string{"172.16.118.72:9000", "172.16.118.120:9000", "172.16.118.112:9000"}
)

func randomServer(id string) string {
	var key int
	if id == "1" {
		key = 1
	} else if id == "2" {
		key = 2
	} else {
		key = 3
	}
	// return serversList[rand.Intn(len(serversList))]
	return serversList[key-1]
}

func handleJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req PlayerJoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	log.Printf("Player %s joined !", req.PlayerID)
	assigned := randomServer(req.PlayerID)
	//res := PlayerJoinResponse{AssignedServer: "127.0.0.1" + assigned, Message: fmt.Sprintf("Player %s assigned to %s", req.PlayerID, assigned)}
	res := Response{Success: true, Message: assigned}
	//log.Println("Assigned:", req.PlayerID, "->", assigned)
	json.NewEncoder(w).Encode(res)
}

func handleFetchChunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	zoneMu.Lock()
	defer zoneMu.Unlock()

	// Normalize chunk coordinates
	chunkID := ChunkID{
		IDX: req.ChunkID.IDX / 32,
		IDY: req.ChunkID.IDY / 32,
	}

	owner, ok := zone[chunkID]
	var res Response

	if ok {
		// Chunk already assigned
		res = Response{Success: false, Message: owner}
	} else {
		// Assign chunk to requesting server
		res = Response{Success: true, Message: "assigned"}
		log.Printf("Assigned chunk (%d,%d) to server %s", chunkID.IDX, chunkID.IDY, req.CallerIP)
	}

	zone[chunkID] = req.CallerIP
	fmt.Printf("Chunk map: %+v\n", zone)
	json.NewEncoder(w).Encode(res)
}

func handleSentChunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	chunk_id := req.ChunkID

	zone[chunk_id] = req.CallerIP

}

func handlePeerChunk(w http.ResponseWriter, r *http.Request) {
	// Check if zone map is initialized
	if zone == nil {
		log.Println("ERROR: zone map is nil")
		http.Error(w, "Server not properly initialized", http.StatusInternalServerError)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Check if request body is nil
	if r.Body == nil {
		http.Error(w, "Request body is empty", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Validate required fields
	// if req.ChunkID == "" {
	// 	http.Error(w, "Missing chunk_id", http.StatusBadRequest)
	// 	return
	// }
	if req.CallerIP == "" {
		http.Error(w, "Missing caller_ip", http.StatusBadRequest)
		return
	}

	chunk_id := req.ChunkID
	caller_load := req.PlayerCount

	owner, ok := zone[chunk_id]

	if !ok {
		res := Response{Success: false}
		zone[chunk_id] = req.CallerIP
		json.NewEncoder(w).Encode(res)
		log.Println("the zone map is ", zone)
		return
	}

	// Resolve UDP addresses with error handling
	peer_addr, err := net.ResolveUDPAddr("udp", owner)
	if err != nil {
		log.Printf("ERROR: Failed to resolve peer address %s: %v", owner, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	local_addr, err := net.ResolveUDPAddr("udp", "172.16.118.72:8080")
	if err != nil {
		log.Printf("ERROR: Failed to resolve local address: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	conn, err := net.DialUDP("udp", local_addr, peer_addr)
	if err != nil {
		log.Printf("ERROR: Failed to dial UDP: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	req_from_central := Request{
		Type:        "FROM_CENTRAL",
		ChunkID:     chunk_id,
		CallerIP:    req.CallerIP,
		PlayerCount: caller_load,
	}

	data, err := json.Marshal(req_from_central)
	if err != nil {
		log.Printf("ERROR: Failed to marshal request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	_, err = conn.Write(data)
	if err != nil {
		log.Printf("ERROR: Failed to write to UDP connection: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("ERROR: Failed to read from UDP connection: %v", err)
		// Continue processing even if read fails, but with default values
		var final_res Response
		if caller_load > 0 { // If we have caller load, assume we should take ownership
			zone[chunk_id] = req.CallerIP
			final_res = Response{Success: true, Message: req.CallerIP, NewIP: req.CallerIP}
		} else {
			final_res = Response{Success: true, Message: owner, NewIP: owner}
		}
		json.NewEncoder(w).Encode(final_res)
		return
	}

	var res Response
	if err := json.Unmarshal(buffer[:n], &res); err != nil {
		log.Println("WARNING: Invalid data from peer, using fallback logic")
		// Fallback logic when unmarshaling fails
		var final_res Response
		if caller_load > 0 {
			zone[chunk_id] = req.CallerIP
			final_res = Response{Success: true, Message: req.CallerIP, NewIP: req.CallerIP}
		} else {
			final_res = Response{Success: true, Message: owner, NewIP: owner}
		}
		json.NewEncoder(w).Encode(final_res)
		return
	}

	var final_res Response
	callee_load := res.PlayerCount
	peer_chunk := res.Chunk

	log.Printf("Processing chunk transfer decision")

	if callee_load < caller_load {
		zone[chunk_id] = req.CallerIP
		final_res = Response{Success: true, Message: req.CallerIP, NewIP: req.CallerIP, Chunk: peer_chunk}
	} else {
		final_res = Response{Success: true, Message: owner, NewIP: owner}
	}

	log.Println("Central map is", zone)
	log.Println("Owner is : ", final_res.Message)
	if err := json.NewEncoder(w).Encode(final_res); err != nil {
		log.Printf("ERROR: Failed to encode response: %v", err)
	}
	log.Println("the zone map is ", zone)
}

// func enableCORS(next http.HandlerFunc) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		w.Header().Set("Access-Control-Allow-Origin", "*")
// 		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
// 		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

// 		if r.Method == "OPTIONS" {
// 			w.WriteHeader(http.StatusOK)
// 			return
// 		}

// 		next(w, r)
// 	}
// }

func main() {
	rand.Seed(time.Now().UnixNano())
	zone = make(map[ChunkID]string)
	http.HandleFunc("/join", enableCORS(handleJoin))
	http.HandleFunc("/chunk", handlePeerChunk)
	http.HandleFunc("/sentchunk", handleSentChunk)
	http.HandleFunc("/peer_chunk", handlePeerChunk)
	log.Println("Central Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

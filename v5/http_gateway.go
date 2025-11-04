package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

// HTTP request structures
type HTTPAddCubeRequest struct {
	Cube    Cube    `json:"cube"`
	ChunkID ChunkID `json:"chunk_id"`
}

type HTTPDltCubeRequest struct {
	CubeID  string  `json:"cube_id"`
	ChunkID ChunkID `json:"chunk_id"`
}

type HTTPMoveRequest struct {
	PlayerID string  `json:"player_id"`
	X        int     `json:"x"`
	Y        int     `json:"y"`
	ChunkID  ChunkID `json:"chunk_id"`
}

type HTTPGetDataRequest struct {
	PlayerID string  `json:"player_id"`
	ChunkID  ChunkID `json:"chunk_id"`
	Player   Player  `json:"player"`
}

type HTTPGetUpdatesRequest struct {
	PlayerID string  `json:"player_id"`
	ChunkID  ChunkID `json:"chunk_id"`
}

type HTTPDeletePlayerRequest struct {
	PlayerID string `json:"player_id"`
}

type HTTPResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

var (
	udpConn   *net.UDPConn
	httpMutex sync.Mutex
)

// sendUDPRequest sends a request to UDP server and waits for response
func sendUDPRequest(req Request, timeout time.Duration) (Response, error) {
	httpMutex.Lock()
	defer httpMutex.Unlock()

	// Marshal request
	data, err := json.Marshal(req)
	if err != nil {
		return Response{}, err
	}

	// Send to UDP server
	udpAddr, err := net.ResolveUDPAddr("udp", "172.16.118.72:9000")
	if err != nil {
		return Response{}, err
	}

	_, err = udpConn.WriteToUDP(data, udpAddr)
	if err != nil {
		return Response{}, err
	}

	// Wait for response with timeout
	buffer := make([]byte, 2048)
	udpConn.SetReadDeadline(time.Now().Add(timeout))
	n, _, err := udpConn.ReadFromUDP(buffer)
	if err != nil {
		return Response{}, err
	}

	var response Response
	if err := json.Unmarshal(buffer[:n], &response); err != nil {
		return Response{}, err
	}

	return response, nil
}

func handleMovePlayerHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var moveReq HTTPMoveRequest
	if err := json.NewDecoder(r.Body).Decode(&moveReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create UDP request
	udpReq := Request{
		Type:    "MOVE_PLAYER",
		Player:  Player{ID: moveReq.PlayerID, PosX: moveReq.X, PosY: moveReq.Y},
		ChunkID: moveReq.ChunkID,
	}

	response, err := sendUDPRequest(udpReq, 5*time.Second)
	if err != nil {
		log.Printf("❌ Error communicating with UDP server: %v", err)
		http.Error(w, "Failed to communicate with game server", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HTTPResponse{
		Success: response.Success,
		Message: response.Message,
		Data:    response.GameData,
	})
}

func handleAddCubeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var dataReq HTTPAddCubeRequest
	if err := json.NewDecoder(r.Body).Decode(&dataReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	udpReq := Request{
		Type:    "ADD_CUBE",
		ChunkID: dataReq.ChunkID,
		Cube:    dataReq.Cube,
	}

	log.Printf("Request chunk id is", dataReq)
	//log.Printf("Request from client is,", udpReq.Player)
	response, err := sendUDPRequest(udpReq, 5*time.Second)
	if err != nil {
		log.Printf("❌ Error communicating with UDP server: %v", err)
		http.Error(w, "Failed to communicate with game server", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HTTPResponse{
		Success: response.Success,
		Message: response.Message,
	})
}

func handleDltCubeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var dataReq HTTPDltCubeRequest
	if err := json.NewDecoder(r.Body).Decode(&dataReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	udpReq := Request{
		Type:    "DLT_CUBE",
		ChunkID: dataReq.ChunkID,
		CubeID:  dataReq.CubeID,
	}

	log.Printf("Request chunk id is", dataReq)
	//log.Printf("Request from client is,", udpReq.Player)
	response, err := sendUDPRequest(udpReq, 5*time.Second)
	if err != nil {
		log.Printf("❌ Error communicating with UDP server: %v", err)
		http.Error(w, "Failed to communicate with game server", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HTTPResponse{
		Success: response.Success,
		Message: response.Message,
	})
}

func handleGetDataHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var dataReq HTTPGetDataRequest
	if err := json.NewDecoder(r.Body).Decode(&dataReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	udpReq := Request{
		Type:    "GET_DATA",
		Player:  dataReq.Player,
		ChunkID: dataReq.ChunkID,
	}

	log.Printf("Request chunk id is", dataReq)
	//log.Printf("Request from client is,", udpReq.Player)
	response, err := sendUDPRequest(udpReq, 5*time.Second)
	if err != nil {
		log.Printf("❌ Error communicating with UDP server: %v", err)
		http.Error(w, "Failed to communicate with game server", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HTTPResponse{
		Success: response.Success,
		Message: response.Message,
		Data:    response.Chunk,
	})
}

func handleGetUpdatesHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var dataReq HTTPGetUpdatesRequest
	if err := json.NewDecoder(r.Body).Decode(&dataReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	udpReq := Request{
		Type:    "GET_UPDATES",
		Player:  Player{ID: dataReq.PlayerID},
		ChunkID: dataReq.ChunkID,
	}

	response, err := sendUDPRequest(udpReq, 5*time.Second)
	if err != nil {
		log.Printf("❌ Error communicating with UDP server: %v", err)
		http.Error(w, "Failed to communicate with game server", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HTTPResponse{
		Success: response.Success,
		Message: response.Message,
		Data:    response.GameData,
	})
}

func handleDeletePlayerHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var dataReq HTTPDeletePlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&dataReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	udpReq := Request{
		Type:   "DLT_PLAYER",
		Player: Player{ID: dataReq.PlayerID},
	}

	response, err := sendUDPRequest(udpReq, 5*time.Second)
	if err != nil {
		log.Printf("❌ Error communicating with UDP server: %v", err)
		http.Error(w, "Failed to communicate with game server", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HTTPResponse{
		Success: response.Success,
		Message: response.Message,
	})
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HTTPResponse{
		Success: true,
		Message: "HTTP Gateway is running",
	})
}

func startHTTPServer() {
	// Register HTTP routes with CORS
	http.HandleFunc("/api/player/move", enableCORS(handleMovePlayerHTTP))
	http.HandleFunc("/api/player/data", enableCORS(handleGetDataHTTP))
	http.HandleFunc("/api/player/updates", enableCORS(handleGetUpdatesHTTP))
	http.HandleFunc("/api/player/delete", enableCORS(handleDeletePlayerHTTP))
	http.HandleFunc("/api/health", enableCORS(handleHealthCheck))
	http.HandleFunc("/api/player/addcube", enableCORS(handleAddCubeHTTP))
	http.HandleFunc("/api/player/dltcube", enableCORS(handleDltCubeHTTP))

	log.Println("🌐 HTTP API Gateway starting on :8081")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal("HTTP server failed:", err)
	}
}

func main() {
	// Initialize UDP connection for HTTP gateway
	var err error
	udpAddr, err := net.ResolveUDPAddr("udp", ":0") // Use any available port
	if err != nil {
		log.Fatal("Failed to resolve UDP addr:", err)
	}

	udpConn, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal("Failed to create UDP connection:", err)
	}
	defer udpConn.Close()

	log.Printf("🔗 HTTP Gateway UDP listener on %s", udpConn.LocalAddr().String())

	// Start HTTP server
	startHTTPServer()
}

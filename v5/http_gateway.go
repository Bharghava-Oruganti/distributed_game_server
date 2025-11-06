package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"time"
)

// ===================== Config =====================

const (
	gameServerUDP = "172.16.118.72:9000" // your game server UDP address
	udpTimeout    = 5 * time.Second      // per request timeout
	udpBufSize    = 65535                // max safe UDP datagram size
)

// ===================== HTTP request structures =====================

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

// ===================== UDP bridge =====================

// sendUDPRequest opens a dedicated UDP socket for this request.
// This avoids race conditions when multiple clients hit the same chunk.
func sendUDPRequest(req Request, timeout time.Duration) (Response, error) {
	// local ephemeral UDP socket
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return Response{}, err
	}
	defer conn.Close()

	// marshal request
	data, err := json.Marshal(req)
	if err != nil {
		return Response{}, err
	}

	// resolve server
	udpAddr, err := net.ResolveUDPAddr("udp", gameServerUDP)
	if err != nil {
		return Response{}, err
	}

	// send
	if _, err := conn.WriteToUDP(data, udpAddr); err != nil {
		return Response{}, err
	}

	// receive
	buf := make([]byte, udpBufSize)
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		return Response{}, err
	}

	var resp Response
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		log.Printf("❌ JSON unmarshal failed. Raw=%q err=%v", string(buf[:n]), err)
		return Response{}, err
	}

	return resp, nil
}

// ===================== HTTP handlers =====================

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

	udpReq := Request{
		Type:    "MOVE_PLAYER",
		Player:  Player{ID: moveReq.PlayerID, PosX: moveReq.X, PosY: moveReq.Y},
		ChunkID: moveReq.ChunkID,
	}

	resp, err := sendUDPRequest(udpReq, udpTimeout)
	if err != nil {
		log.Printf("❌ UDP MOVE_PLAYER error: %v", err)
		http.Error(w, "Failed to communicate with game server", http.StatusInternalServerError)
		return
	}

	writeJSON(w, HTTPResponse{Success: resp.Success, Message: resp.Message, Data: resp.GameData})
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

	log.Printf("ADD_CUBE req: %+v", dataReq)

	resp, err := sendUDPRequest(udpReq, udpTimeout)
	if err != nil {
		log.Printf("❌ UDP ADD_CUBE error: %v", err)
		http.Error(w, "Failed to communicate with game server", http.StatusInternalServerError)
		return
	}

	writeJSON(w, HTTPResponse{Success: resp.Success, Message: resp.Message})
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

	log.Printf("DLT_CUBE req: %+v", dataReq)

	resp, err := sendUDPRequest(udpReq, udpTimeout)
	if err != nil {
		log.Printf("❌ UDP DLT_CUBE error: %v", err)
		http.Error(w, "Failed to communicate with game server", http.StatusInternalServerError)
		return
	}

	writeJSON(w, HTTPResponse{Success: resp.Success, Message: resp.Message})
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

	log.Printf("GET_DATA req: %+v", dataReq)

	resp, err := sendUDPRequest(udpReq, udpTimeout)
	if err != nil {
		log.Printf("❌ UDP GET_DATA error: %v", err)
		http.Error(w, "Failed to communicate with game server", http.StatusInternalServerError)
		return
	}

	writeJSON(w, HTTPResponse{Success: resp.Success, Message: resp.Message, Data: resp.Chunk})
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

	resp, err := sendUDPRequest(udpReq, udpTimeout)
	if err != nil {
		log.Printf("❌ UDP GET_UPDATES error: %v", err)
		http.Error(w, "Failed to communicate with game server", http.StatusInternalServerError)
		return
	}

	writeJSON(w, HTTPResponse{Success: resp.Success, Message: resp.Message, Data: resp.GameData})
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

	resp, err := sendUDPRequest(udpReq, udpTimeout)
	if err != nil {
		log.Printf("❌ UDP DLT_PLAYER error: %v", err)
		http.Error(w, "Failed to communicate with game server", http.StatusInternalServerError)
		return
	}

	writeJSON(w, HTTPResponse{Success: resp.Success, Message: resp.Message})
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, HTTPResponse{Success: true, Message: "HTTP Gateway is running"})
}

// ===================== HTTP bootstrap =====================

func startHTTPServer() {
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
	// no shared UDP socket needed anymore
	startHTTPServer()
}

// ===================== Helpers =====================

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}



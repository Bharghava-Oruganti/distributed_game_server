package main

import (
	"bytes"
	"encoding/json"
	"log"
	"math/rand"
	"net"
	"net/http"
	"time"
)

type PlayerState struct {
	conn         *net.UDPConn
	serverAddr   *net.UDPAddr
	player       Player
	currentChunk ChunkID
	serverIP     string
}

func NewPlayerState(playerID string) *PlayerState {
	serverAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:9000")
	if err != nil {
		log.Fatal("ResolveUDPAddr failed:", err)
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		log.Fatal("DialUDP failed:", err)
	}

	return &PlayerState{
		conn:       conn,
		serverAddr: serverAddr,
		player:     Player{ID: playerID, PosX: 0, PosY: 0},
		serverIP:   "127.0.0.1:9000",
	}
}

func (ps *PlayerState) CalculateChunkID() ChunkID {
	chunkSize := 32
	return ChunkID{
		IDX: int(ps.player.PosX / chunkSize),
		IDY: int(ps.player.PosY / chunkSize),
	}
}

func (ps *PlayerState) SendRequest(req Request) (*Response, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Send request
	_, err = ps.conn.Write(data)
	if err != nil {
		return nil, err
	}

	// Wait for response
	buf := make([]byte, 4096)
	ps.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _, err := ps.conn.ReadFromUDP(buf)
	if err != nil {
		return nil, err
	}

	var res Response
	if err := json.Unmarshal(buf[:n], &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (ps *PlayerState) Initialize() {
	log.Printf("🎮 Player %s initializing...", ps.player.ID)

	// Get initial chunk
	chunkID := ps.CalculateChunkID()
	req := Request{
		Type:    "GET_DATA",
		Player:  ps.player,
		ChunkID: chunkID,
	}

	res, err := ps.SendRequest(req)
	if err != nil {
		log.Printf("❌ Initialization failed: %v", err)
		return
	}

	if res.Success {
		ps.currentChunk = chunkID
		log.Printf("✅ Joined chunk [%d,%d] - %s", chunkID.IDX, chunkID.IDY, res.Message)
	} else {
		log.Printf("⚠️  Server message: %s and changing to :", ps.serverIP, res.Message)
		ps.ChangeServerIP(res.Message)

	}
}

func (ps *PlayerState) MoveRandomly() {
	//Random movement within bounds
	ps.player.PosX += 1 // -3 to +3
	ps.player.PosY += 1

	// Keep within reasonable bounds
	if ps.player.PosX < 0 {
		ps.player.PosX = 0
	}
	if ps.player.PosY < 0 {
		ps.player.PosY = 0
	}
	if ps.player.PosX > 500 {
		ps.player.PosX = 500
	}
	if ps.player.PosY > 500 {
		ps.player.PosY = 500
	}
}

func (ps *PlayerState) HandleChunkTransition() bool {
	newChunk := ps.CalculateChunkID()

	// Check if chunk changed
	if newChunk != ps.currentChunk {
		log.Printf("🔄 Chunk transition: [%d,%d] → [%d,%d]",
			ps.currentChunk.IDX, ps.currentChunk.IDY,
			newChunk.IDX, newChunk.IDY)

		// Get data for new chunk
		req := Request{
			Type:    "GET_DATA",
			Player:  ps.player,
			ChunkID: newChunk,
		}

		res, err := ps.SendRequest(req)
		if err != nil {
			log.Printf("❌ Failed to get new chunk: %v", err)
			return false
		}

		if res.Success {
			ps.currentChunk = newChunk
			log.Printf("✅ Entered new chunk [%d,%d]", newChunk.IDX, newChunk.IDY)
			return true
		} else {
			log.Printf("⚠️  Cannot enter chunk: %s", res.Message)
			return false
		}
	}
	return true
}

func (ps *PlayerState) UpdatePosition() {
	// Send move request
	moveReq := Request{
		Type:    "MOVE_PLAYER",
		Player:  ps.player,
		ChunkID: ps.currentChunk,
	}

	_, err := ps.SendRequest(moveReq)
	if err != nil {
		log.Printf("❌ Move update failed: %v", err)
	} else {
		log.Printf("📍 Position updated: (%d, %d)", ps.player.PosX, ps.player.PosY)
	}
}

func (ps *PlayerState) GetNearbyPlayers() {
	// Request updates about nearby players
	updateReq := Request{
		Type:    "GET_UPDATES",
		Player:  ps.player,
		ChunkID: ps.currentChunk,
	}

	res, err := ps.SendRequest(updateReq)
	if err != nil {
		log.Printf("❌ Failed to get updates: %v", err)
		return
	}

	if res.Success {
		log.Printf("👥 Received chunk updates")
		log.Printf("Gamedata is : ", res.GameData)
	}
}

func (ps *PlayerState) GameLoop() {
	log.Printf("🎯 Starting game loop for player %s", ps.player.ID)

	ticker := time.NewTicker(2000 * time.Millisecond) // 2 seconds per game tick
	defer ticker.Stop()

	frame := 0
	for range ticker.C {
		frame++
		log.Printf("\n--- Frame %d ---", frame)

		// 1. Move player randomly
		ps.MoveRandomly()

		// 2. Handle chunk transitions
		if !ps.HandleChunkTransition() {
			continue // Skip this frame if chunk transition failed
		}

		ps.Initialize()

		// 3. Update position on server
		ps.UpdatePosition()

		// 4. Get nearby players and updates (every 3 frames)
		if frame%3 == 0 {
			ps.GetNearbyPlayers()
		}

		// 5. Log current state - FIXED FORMATTING
		log.Printf("🎮 Player %s at (%d, %d) in chunk [%d,%d]",
			ps.player.ID, ps.player.PosX, ps.player.PosY,
			ps.currentChunk.IDX, ps.currentChunk.IDY)
	}
}

func (ps *PlayerState) Cleanup() {
	log.Printf("🧹 Cleaning up player %s", ps.player.ID)

	// Notify server about player departure
	req := Request{
		Type:    "DLT_PLAYER",
		Player:  ps.player,
		ChunkID: ps.currentChunk,
	}

	ps.SendRequest(req) // Best effort cleanup
	ps.conn.Close()
}

func (ps *PlayerState) ChangeServerIP(new_IP string) {
	log.Printf("Changing server ip")
	log.Printf("The new ip of %s", ps.player.ID, "is ", new_IP)

	ps.serverIP = new_IP

	serverAddr, err := net.ResolveUDPAddr("udp", ps.serverIP)
	if err != nil {
		log.Fatal("ResolveUDPAddr failed:", err)
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		log.Fatal("DialUDP failed:", err)
	}

	ps.conn = conn
	ps.serverAddr = serverAddr
}

func (ps *PlayerState) join(playerID string) {

	//centralReq := Request{Type: "GET_CHUNK", ChunkID: chunk_id, CallerIP: serverIP}
	req := Request{Type: "JOIN", PlayerID: playerID}
	b, _ := json.Marshal(req)
	httpResp, _ := http.Post("http://127.0.0.1:8080/join", "application/json", bytes.NewReader(b))
	var res Response
	json.NewDecoder(httpResp.Body).Decode(&res)

	ps.ChangeServerIP(res.Message)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// Create player with unique ID
	//playerID := "player_" + time.Now().Format("150405")
	playerID := "1"
	player := NewPlayerState(playerID)
	defer player.Cleanup()

	// Initialize and start game loop
	player.join(playerID)
	player.Initialize()
	player.GameLoop()
}

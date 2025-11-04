package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

var (
	zone_map    = make(map[ChunkID]Chunk)
	zone_map_Mu sync.Mutex
	serverIP    = "172.16.118.72:9000" // Set your actual server IP
	players     = make(map[string]ChunkID)
	player_map  = make(map[string]Player)
)

// Represents a simple player event (e.g., move, shoot, jump, etc.)
type PlayerEvent struct {
	PlayerID string  `json:"player_id"`
	Action   string  `json:"action"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
}

type ZoneMap struct {
	sync.Mutex
	ZoneMap map[ChunkID]Chunk
}

func (r *ZoneMap) AddPlayer(chunk_id ChunkID, chunk *Chunk) {
	r.Lock()
	defer r.Unlock()
	r.ZoneMap[chunk_id] = *chunk
}

func (r *ZoneMap) RemovePlayer(chunk_id ChunkID) {
	r.Lock()
	defer r.Unlock()
	delete(r.ZoneMap, chunk_id)
}

func sendUDP(conn *net.UDPConn, addr *net.UDPAddr, data []byte) {
	_, err := conn.WriteToUDP(data, addr)
	if err != nil {
		log.Printf("❌ Error sending to %s: %v", addr.String(), err)
	}
}

func sendJSON(conn *net.UDPConn, addr *net.UDPAddr, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Println("JSON marshal error:", err)
		return
	}
	sendUDP(conn, addr, data)
}

func main() {
	port := "172.16.118.72:9000"
	addr, err := net.ResolveUDPAddr("udp", port)
	if err != nil {
		log.Fatal("ResolveUDPAddr failed:", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal("ListenUDP failed:", err)
	}
	defer conn.Close()

	log.Printf("🎮 Game server listening on %s", port)

	buf := make([]byte, 2048)
	for {
		n, playerAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Println("ReadFromUDP error:", err)
			continue
		}

		// Decode event
		var req Request
		if err := json.Unmarshal(buf[:n], &req); err != nil {
			log.Println("Invalid data from", playerAddr, ":", err)
			continue
		}

		log.Printf("📩 Received request from %s of type : %s", req.Player.ID, req.Type)

		switch req.Type {
		case "GET_DATA":
			handleGetData(conn, playerAddr, req)
		case "FROM_CENTRAL":
			handleCentralPeerReq(req, conn, playerAddr)
		case "UPDATE_DATA":
			handleUpdateData(req, conn, playerAddr) // Added conn and addr
		case "MOVE_PLAYER":
			handleMovePlayer(req, conn, playerAddr) // Added conn and addr
		case "GET_UPDATES":
			handleGetUpdates(conn, playerAddr, req)
		case "DLT_PLAYER":
			handleDeletePlayer(req, conn, playerAddr) // Added conn and addr
		case "READ_ONLY":
			handleReadOnly(req, conn, playerAddr)
		case "MERGE":
			handleMergeChunk(req, conn, playerAddr)
		case "ADD_CUBE":
			handleAddCube(req, conn, playerAddr)
		case "DLT_CUBE":
			handleDltCube(req, conn, playerAddr)
		default:
			log.Printf("❌ Unknown request type: %s", req.Type)
			// Send error response
			errorRes := Response{Success: false, Message: "Unknown request type"}
			sendJSON(conn, playerAddr, errorRes)
		}

	}
}

func deleteFromList(s []Cube, idx int) []Cube {
	s[idx] = s[len(s)-1]
	return s[:len(s)-1]
}

func handleDltCube(req Request, conn *net.UDPConn, addr *net.UDPAddr) {
	chunk_id := req.ChunkID
	chunk, _ := zone_map[chunk_id]

	for cell_no, cell := range chunk.Cells {
		if cell.ID == req.CubeID {
			chunk.Cells = deleteFromList(chunk.Cells, cell_no)
			break
		}
	}

	chunk.IsDirty = true
	zone_map[chunk_id] = chunk

	res := Response{Success: true, Message: "Deleted Cube"}
	sendJSON(conn, addr, res)

	log.Printf("Deleted Cube !")
	log.Printf("The updated zone map is ", zone_map)
}

func handleAddCube(req Request, conn *net.UDPConn, addr *net.UDPAddr) {
	chunk_id := req.ChunkID
	// chunk is owned by this server
	chunk, _ := zone_map[chunk_id]

	chunk.Cells = append(chunk.Cells, req.Cube)

	chunk.IsDirty = true

	zone_map[chunk_id] = chunk

	res := Response{Success: true, Message: "Added Cube"}
	sendJSON(conn, addr, res)

	log.Printf("Added cube : ", req.Cube.ID)
	log.Printf("Updated zone map is : ", zone_map)
}

func handleMergeChunk(req Request, conn *net.UDPConn, addr *net.UDPAddr) {
	chunk_id := req.ChunkID
	chunk, ok := zone_map[chunk_id]
	req_chunk := req.Chunk

	if !ok {
		zone_map[chunk_id] = req_chunk
	} else {
		for _, player := range req_chunk.PlayerList {
			chunk.PlayerList = append(chunk.PlayerList, player)
		}

		zone_map[chunk_id] = chunk
	}

	res := Response{Success: true, Message: "Merged Chunk"}
	sendJSON(conn, addr, res)

	log.Printf("Merged Chunk")

}

func handleReadOnly(req Request, conn *net.UDPConn, addr *net.UDPAddr) {

	chunk_id := req.ChunkID

	chunk, _ := zone_map[chunk_id]

	var res Response
	if req.IsChunkNew || chunk.IsDirty || len(chunk.PlayerList) > 0 {
		res = Response{Success: true, Chunk: chunk, Message: "Sending the chunk"}
	} else {
		res = Response{Success: false, Message: "Use your local copy"}
	}

	sendJSON(conn, addr, res)

	log.Printf("Handled P2P conn")
}

func handleDeletePlayer(req Request, conn *net.UDPConn, addr *net.UDPAddr) {
	player_id := req.Player.ID
	delete(players, player_id)
	delete(player_map, player_id)

	// Send response
	res := Response{Success: true, Message: "Player deleted"}
	sendJSON(conn, addr, res)

	log.Printf("🗑️ Player %s deleted", player_id)
}
func handleGetUpdates(conn *net.UDPConn, addr *net.UDPAddr, req Request) {

	//player_id := req.Player.ID
	chunk_id := req.ChunkID
	chunk := zone_map[chunk_id]
	var players_in_chunk []Player

	for player, id := range players {
		if id == chunk_id {
			players_in_chunk = append(players_in_chunk, player_map[player])
		}
	}

	// send the update response via udp
	data := GameData{Chunk: chunk}
	res := Response{Success: true, GameData: data} //
	sendJSON(conn, addr, res)

	log.Printf("📊 Sent updates for chunk [%d,%d] with %d players",
		chunk_id.IDX, chunk_id.IDY, len(players_in_chunk))
}

func handleMovePlayer(req Request, conn *net.UDPConn, addr *net.UDPAddr) {
	player_id := req.Player.ID
	chunk_id := req.ChunkID
	player := req.Player

	players[player_id] = chunk_id
	player_map[player_id] = player

	// Send response back to client
	res := Response{
		Success: true,
		Message: "Player position updated",
	}
	sendJSON(conn, addr, res)

	log.Printf("✅ Player %s moved to (%d, %d) in chunk [%d,%d]",
		player_id, player.PosX, player.PosY, chunk_id.IDX, chunk_id.IDY)
}

func handleCentralPeerReq(req Request, conn *net.UDPConn, addr *net.UDPAddr) {
	chunk_id := req.ChunkID
	chunk, _ := zone_map[chunk_id]

	// var ok bool
	// ok = true
	// for _, id := range players {
	// 	if id == chunk_id {
	// 		ok = false
	// 		break
	// 	}
	// }

	caller_player_count := req.PlayerCount
	my_player_count := len(chunk.PlayerList)

	var res Response
	//res = Response{Success: true, PlayerCount: my_player_count}

	if caller_player_count >= my_player_count {
		chunk.ServerIP = req.CallerIP
		for _, player := range chunk.PlayerList {
			player.ServerIP = req.CallerIP
		}
		chunk.IsDirty = true
		zone_map[chunk_id] = chunk
		res = Response{Success: true, Chunk: chunk, PlayerCount: my_player_count}
		merge_req := Request{Type: "MERGE", ChunkID: chunk_id, Chunk: chunk}
		merge_res, _ := merge(merge_req, req.CallerIP)
		log.Printf(merge_res.Message)
	} else {
		res = Response{Success: true, PlayerCount: my_player_count, Chunk: chunk}
	}
	// if ok {
	// 	// transfer chunk
	// 	res = Response{Success: true, Chunk: chunk}
	// 	chunk.ServerIP = req.CallerIP
	// 	zone_map[chunk_id] = chunk
	// } else {
	// 	res = Response{Success: false, Message: serverIP}
	// }

	sendJSON(conn, addr, res)
}

func handleUpdateData(req Request, conn *net.UDPConn, addr *net.UDPAddr) {
	chunk_id := req.ChunkID
	chunk := req.Chunk
	zone_map[chunk_id] = chunk

	// Send response
	res := Response{Success: true, Message: "Chunk data updated"}
	sendJSON(conn, addr, res)

	log.Printf("🔄 Chunk [%d,%d] data updated", chunk_id.IDX, chunk_id.IDY)
}
func handleGetData(conn *net.UDPConn, addr *net.UDPAddr, req Request) {
	//log.Println("Welcome to ")
	// creating chunk id
	chunk_id := req.ChunkID

	log.Printf("Request chunk id is", chunk_id)
	player_id := req.Player.ID
	player := req.Player
	//writeAccess := req.WriteAccess
	val, ok := zone_map[chunk_id]
	var res Response
	var player_count int
	if ok {
		player_count = len(val.PlayerList)
	} else {
		player_count = 0
	}
	if ok && val.ServerIP == serverIP {
		res = Response{Success: true, Chunk: val, Message: serverIP}
		players[player_id] = chunk_id
	} else {

		centralReq := Request{Type: "GET_CHUNK", ChunkID: chunk_id, CallerIP: serverIP, PlayerCount: player_count}
		b, _ := json.Marshal(centralReq)
		httpResp, _ := http.Post("http://172.16.118.72:8080/chunk", "application/json", bytes.NewReader(b))
		var central_response Response
		json.NewDecoder(httpResp.Body).Decode(&central_response)

		if !central_response.Success {
			log.Printf("New chunk ! first operation !")
			new_chunk := Chunk{IDX: chunk_id.IDX, IDY: chunk_id.IDY, Data: "new chunk", ServerIP: serverIP, Cells: make([]Cube, 0)}

			players[player_id] = chunk_id
			player_map[player_id] = player
			new_chunk.PlayerList = append(new_chunk.PlayerList, player)
			zone_map[chunk_id] = new_chunk
			res = Response{Success: true, Chunk: new_chunk, Message: serverIP}
		} else {
			// make the call to owner just to get the updated data
			// make a peer to peer connection with owner and also state wheter u have the chunk
			// if the owner donot send any data it means the data is not dirty and u can use your local copy
			// central server will decide who get write access and owner is the final owner of chunk
			// players of non-owners gets reconnects
			owner := central_response.Message
			//new_ip := central_response.NewIP
			//chunk, ok := zone_map[chunk_id]
			// req := Request{Type: "READ_ONLY", ChunkID: chunk_id, IsChunkNew: ok}
			// peer_res, _ := p2p(req, owner)

			if ok && owner != serverIP {
				//if ok {
				for _, player := range val.PlayerList {
					player.ServerIP = owner
				}
				val.ServerIP = owner
				val.IsDirty = true
				zone_map[chunk_id] = val
				//}

				merge_req := Request{Type: "MERGE", ChunkID: chunk_id, Chunk: val}
				merge_res, _ := merge(merge_req, owner)
				log.Printf(merge_res.Message)
				res = Response{Success: true, Message: owner}
			} else if !ok && owner != serverIP {
				temp_chunk := Chunk{}
				temp_chunk.PlayerList = append(temp_chunk.PlayerList, player)
				merge_req := Request{Type: "MERGE", ChunkID: chunk_id, Chunk: temp_chunk}
				merge_res, _ := merge(merge_req, owner)
				log.Printf(merge_res.Message)
				res = Response{Success: true, Message: owner}
			} else if ok {
				updated_chunk := zone_map[chunk_id]
				res = Response{Success: true, Chunk: updated_chunk, Message: owner}
			} else {
				updated_chunk := central_response.Chunk
				updated_chunk.PlayerList = append(updated_chunk.PlayerList, player)
				res = Response{Success: true, Chunk: updated_chunk, Message: owner}
			}

			zone_map[chunk_id] = res.Chunk
			//sendJSON(res,)
		}
		// } else {
		// 	new_owner := central_response.Message
		// 	if ok && new_owner != serverIP {
		// 		// update the server ip of players in chunks to this new ip
		// 	} else if new_owner != serverIP {
		// 		// change the player ip
		// 	}
		// }
		// centralReq := Request{Type: "GET_CHUNK", ChunkID: chunk_id, CallerIP: serverIP}
		// b, _ := json.Marshal(centralReq)
		// httpResp, _ := http.Post("http://127.0.0.1:8080/chunk", "application/json", bytes.NewReader(b))
		// var central_response Response
		// json.NewDecoder(httpResp.Body).Decode(&central_response)

		// if !central_response.Success {
		// 	log.Printf("New chunk ! first operation !")
		// 	new_chunk := Chunk{IDX: chunk_id.IDX, IDY: chunk_id.IDY, Data: "new chunk", ServerIP: serverIP}

		// 	zone_map[chunk_id] = new_chunk
		// 	players[player_id] = chunk_id
		// 	player_map[player_id] = player
		// 	res = Response{Success: true, Chunk: new_chunk, Message: "New chunk created"}
		// } else {
		// 	// in this case we have bring chunk from different server
		// 	// we have server ip in central_response
		// 	// create a peer request
		// 	current_chunk_owner := central_response.Message
		// 	if current_chunk_owner == serverIP {
		// 		// sending the player a message to change his ip in Message

		// 		res = Response{Success: false, Message: central_response.Message}
		// 	} else {
		// 		p2p()
		// 		chunk := central_response.Chunk
		// 		zone_map[chunk_id] = chunk
		// 		players[player_id] = chunk_id
		// 		res = Response{Success: true, Chunk: chunk, Message: "Chunk fetched from peer"}
		// 	}
		//}
	}

	sendJSON(conn, addr, res)
}

func merge(req Request, peer_ip string) (*Response, error) {
	peerAddr, err := net.ResolveUDPAddr("udp", peer_ip)
	if err != nil {
		log.Fatal("ResolveUDPAddr failed:", err)
	}

	conn, err := net.DialUDP("udp", nil, peerAddr)
	if err != nil {
		log.Fatal("DialUDP failed:", err)
	}
	defer conn.Close()
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Send request
	_, err = conn.Write(data)
	if err != nil {
		return nil, err
	}

	// Wait for response
	buf := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		return nil, err
	}

	var res Response
	if err := json.Unmarshal(buf[:n], &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func p2p(req Request, peer_ip string) (*Response, error) {

	peerAddr, err := net.ResolveUDPAddr("udp", peer_ip)
	if err != nil {
		log.Fatal("ResolveUDPAddr failed:", err)
	}

	conn, err := net.DialUDP("udp", nil, peerAddr)
	if err != nil {
		log.Fatal("DialUDP failed:", err)
	}
	defer conn.Close()
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Send request
	_, err = conn.Write(data)
	if err != nil {
		return nil, err
	}

	// Wait for response
	buf := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		return nil, err
	}

	var res Response
	if err := json.Unmarshal(buf[:n], &res); err != nil {
		return nil, err
	}

	return &res, nil
}

package main

import "net/http"

type GameData struct {
	Chunk Chunk `json:"chunk"`
}
type Player struct {
	ID        string  `json:"id"`
	PosX      int     `json:"posx"`
	PosY      int     `json:"posy"`
	ServerIP  string  `json:"server_ip"`
	AOIRadius int     `json:"aoi_radius"`
	ChunkID   ChunkID `json:"chunk_id"`
}

type Cube struct {
	ID     string `json:"cube_id"`
	X      int    `json:"x"`
	Z      int    `json:"z"`
	Height int    `json:"height"`
	Color  string `json:"color"`
}

type Chunk struct {
	IDX        int      `json:"id_x"`
	IDY        int      `json:"id_y"`
	ServerIP   string   `json:"server_ip"`
	Data       string   `json:"data"`
	PlayerList []Player `json:"player_list"`
	IsDirty    bool     `json:"is_dirty"`
	Cells      []Cube   `json:"cells"`
}

type ChunkID struct {
	IDX int `json:"id_x"`
	IDY int `json:"id_y"`
}

type Request struct {
	Type        string  `json:"type"`
	ChunkID     ChunkID `json:"chunk_id"`
	CallerIP    string  `json:"caller_ip"`
	Player      Player  `json:"player"`
	IsPeerReq   bool    `json:"is_peer_req"`
	Chunk       Chunk   `json:"chunk"`
	IsChunkNew  bool    `json:"is_chunk_new"`
	PlayerCount int     `json:"player_count"`
	PlayerID    string  `json:"player_id"`
	Cube        Cube    `json:"cube"`
	CubeID      string  `json:"cube_id"`
}

type Response struct {
	Success     bool     `json:"success"`
	Chunk       Chunk    `json:"chunk"`
	Message     string   `json:"message"`
	GameData    GameData `json:"game_data"`
	NewIP       string   `json:"new_ip"`
	PlayerCount int      `json:"player_count"`
}

type PlayerJoinRequest struct {
	PlayerID string `json:"player_id"`
	PosX     int    `json:"pos_x"`
	PosY     int    `json:"pos_y"`
}

type PlayerJoinResponse struct {
	AssignedServer string `json:"assigned_server"`
	Message        string `json:"message"`
}

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// api.js
const API_BASE_URL = 'http://172.16.118.72:8081/api'; // Your HTTP gateway

// Helper function to make API calls
async function apiCall(endpoint, data) {
  try {
    const response = await fetch(`${API_BASE_URL}${endpoint}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error('API call failed:', error);
    throw error;
  }
}

// Get chunk data for a specific chunk ID
export async function getChunkData(chunkId, player) {
  return await apiCall('/player/data', {
    chunk_id: chunkId,
    player: player
  });
}

// Move player to new position
export async function movePlayer(playerId, x, y, chunkId) {
  return await apiCall('/player/move', {
    player_id: playerId,
    x: x,
    y: y,
    chunk_id: chunkId
  });
}

// Get updates for a chunk
export async function getChunkUpdates(playerId, chunkId) {
  return await apiCall('/player/updates', {
    player_id: playerId,
    chunk_id: chunkId
  });
}

// Delete player
export async function deletePlayer(playerId) {
  return await apiCall('/player/delete', {
    player_id: playerId
  });
}

// Health check
export async function healthCheck() {
  const response = await fetch(`${API_BASE_URL}/health`);
  return await response.json();
}
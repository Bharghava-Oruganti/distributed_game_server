// useChunkManager.js
import { useState, useEffect, useCallback } from 'react';
import { getChunkData, getChunkUpdates, movePlayer } from './api';

// Generate player ID (you might want to persist this)
const generatePlayerId = () => `player_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

export function useChunkManager() {
  const [currentChunk, setCurrentChunk] = useState(null);
  const [playerId] = useState(generatePlayerId());
  const [playerPosition, setPlayerPosition] = useState({ x: 0, z: 0 });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  // Convert 3D position to chunk coordinates
  const getChunkIdFromPosition = (x, z, chunkSize = 10) => {
    const chunkX = Math.floor(x / chunkSize);
    const chunkZ = Math.floor(z / chunkSize);
    return { IDX: chunkX, IDY: chunkZ };
  };

  // Get current player chunk ID
  const getCurrentChunkId = useCallback(() => {
    return getChunkIdFromPosition(playerPosition.x, playerPosition.z);
  }, [playerPosition]);

  // Fetch chunk data
  const fetchChunk = useCallback(async (chunkId) => {
    setLoading(true);
    setError(null);
    
    try {
      const playerData = {
        ID: playerId,
        PosX: Math.round(playerPosition.x),
        PosY: Math.round(playerPosition.z) // Using z as Y for 2D position
      };

      const response = await getChunkData(chunkId, playerData);
      
      if (response.success) {
        setCurrentChunk(response.data);
        return response.data;
      } else {
        throw new Error(response.message || 'Failed to fetch chunk');
      }
    } catch (err) {
      setError(err.message);
      console.error('Error fetching chunk:', err);
      return null;
    } finally {
      setLoading(false);
    }
  }, [playerId, playerPosition]);

  // Fetch current chunk
  const fetchCurrentChunk = useCallback(async () => {
    const chunkId = getCurrentChunkId();
    return await fetchChunk(chunkId);
  }, [getCurrentChunkId, fetchChunk]);

  // Update player position and handle chunk changes
  const updatePlayerPosition = useCallback(async (newX, newZ) => {
    const oldChunkId = getCurrentChunkId();
    setPlayerPosition({ x: newX, z: newZ });
    
    const newChunkId = getChunkIdFromPosition(newX, newZ);
    
    // Check if player changed chunks
    if (oldChunkId.IDX !== newChunkId.IDX || oldChunkId.IDY !== newChunkId.IDY) {
      console.log('Player moved to new chunk:', newChunkId);
      await fetchChunk(newChunkId);
    }

    // Update server with new position
    try {
      await movePlayer(playerId, Math.round(newX), Math.round(newZ), newChunkId);
    } catch (err) {
      console.error('Failed to update player position on server:', err);
    }
  }, [playerId, getCurrentChunkId, fetchChunk]);

  // Convert chunk data to cubes for your game
  const getCubesFromChunk = useCallback(() => {
    if (!currentChunk || !currentChunk.Cells) return [];

    return currentChunk.Cells.map(cell => ({
      id: `cube_${cell.x}_${cell.z}_${cell.height}`,
      position: [cell.x, cell.height, cell.z],
      color: cell.color || '#ff0000'
    }));
  }, [currentChunk]);

  // Periodically fetch chunk updates
  useEffect(() => {
    if (!currentChunk) return;

    const interval = setInterval(async () => {
      try {
        const response = await getChunkUpdates(playerId, getCurrentChunkId());
        if (response.success && response.data) {
          setCurrentChunk(response.data.Chunk);
        }
      } catch (err) {
        console.error('Error fetching chunk updates:', err);
      }
    }, 2000); // Update every 2 seconds

    return () => clearInterval(interval);
  }, [currentChunk, playerId, getCurrentChunkId]);

  return {
    currentChunk,
    playerId,
    playerPosition,
    loading,
    error,
    fetchChunk,
    fetchCurrentChunk,
    updatePlayerPosition,
    getCubesFromChunk,
    getCurrentChunkId
  };
}
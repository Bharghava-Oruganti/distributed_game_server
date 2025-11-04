import create from "zustand";

export const useGameStore = create((set) => ({
  playerId: null,
  playerPos: { x: 0, y: 0 },
  chunkId: { x: 0, y: 0 },
  world: {}, // keyed by chunk + tile
  setPlayer: (data) => set({ ...data }),
  updateWorld: (updates) =>
    set((state) => ({
      world: { ...state.world, ...updates }
    })),
}));

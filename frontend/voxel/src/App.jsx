// Game.js (with dynamic chunk loading)
import React, { useState, useRef, useCallback, useEffect } from 'react';
import { Canvas, useFrame, useThree } from '@react-three/fiber';
import { PointerLockControls } from '@react-three/drei';
import * as THREE from 'three';

// Grid Cell component
function GridCell({ x, z, gridX, gridZ, onLeftClick, onRightClick }) {
  return (
    <mesh
      position={[gridX, -0.5, gridZ]}
      rotation={[-Math.PI / 2, 0, 0]}
      onPointerDown={(e) => {
        e.stopPropagation();
        if (e.button === 0) onLeftClick(x, z);
        if (e.button === 2) onRightClick(x, z);
      }}
    >
      <planeGeometry args={[0.95, 0.95]} />
      <meshBasicMaterial color="#666" transparent opacity={0.6} />
    </mesh>
  );
}

// Grid component - now handles multiple chunks
function Grid({ chunks, onCellClick, onCellRightClick, playerPosition }) {
  const visibleRange = 25;

  return (
    <group>
      {Object.entries(chunks).map(([chunkKey, chunkData]) => 
        chunkData.cells.map((row, x) =>
          row.map((cell, z) => {
            const gridX = x + chunkData.offsetX;
            const gridZ = z + chunkData.offsetZ;

            const distance = Math.sqrt(
              Math.pow(gridX - playerPosition[0], 2) +
                Math.pow(gridZ - playerPosition[2], 2)
            );

            if (distance > visibleRange) return null;

            return (
              <GridCell
                key={`${chunkKey}-${x}-${z}`}
                x={x}
                z={z}
                gridX={gridX}
                gridZ={gridZ}
                onLeftClick={(cellX, cellZ) => onCellClick(cellX, cellZ, chunkData.offsetX, chunkData.offsetZ)}
                onRightClick={(cellX, cellZ) => onCellRightClick(cellX, cellZ, chunkData.offsetX, chunkData.offsetZ)}
              />
            );
          })
        )
      )}
    </group>
  );
}

// Cube component with visibility based on player distance
function Cube({ position, color, onRightClick, playerPosition }) {
  const distance = Math.sqrt(
    Math.pow(position[0] - playerPosition[0], 2) +
    Math.pow(position[2] - playerPosition[2], 2)
  );
  
  const visibleRange = 25;

  if (distance > visibleRange) return null;

  return (
    <mesh
      position={position}
      onPointerDown={(e) => {
        e.stopPropagation();
        if (e.button === 2) onRightClick();
      }}
    >
      <boxGeometry args={[0.8, 0.8, 0.8]} />
      <meshStandardMaterial color={color} />
    </mesh>
  );
}

// Player component
function Player({ onPositionUpdate }) {
  const { camera } = useThree();
  const previousPosition = useRef([Infinity, Infinity, Infinity]);

  useFrame(() => {
    const currentPosition = [camera.position.x, camera.position.y, camera.position.z];

    const distance = Math.sqrt(
      Math.pow(currentPosition[0] - previousPosition.current[0], 2) +
        Math.pow(currentPosition[1] - previousPosition.current[1], 2) +
        Math.pow(currentPosition[2] - previousPosition.current[2], 2)
    );

    if (distance > 0.05) {
      onPositionUpdate(currentPosition);
      previousPosition.current = currentPosition;
    }
  });

  return null;
}

// Crosshair component
function Crosshair() {
  return (
    <div
      style={{
        position: 'absolute',
        top: '50%',
        left: '50%',
        transform: 'translate(-50%, -50%)',
        width: '20px',
        height: '20px',
        pointerEvents: 'none',
        zIndex: 1000
      }}
    >
      <div
        style={{
          position: 'absolute',
          top: '50%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
          width: '2px',
          height: '20px',
          backgroundColor: 'white'
        }}
      />
      <div
        style={{
          position: 'absolute',
          top: '50%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
          width: '20px',
          height: '2px',
          backgroundColor: 'white'
        }}
      />
    </div>
  );
}

// First Person Controller
function FirstPersonController({ onPositionUpdate }) {
  const moveForward = useRef(false);
  const moveBackward = useRef(false);
  const moveLeft = useRef(false);
  const moveRight = useRef(false);
  const canJump = useRef(false);
  const velocity = useRef(new THREE.Vector3());
  const direction = useRef(new THREE.Vector3());
  const { camera } = useThree();

  const speed = 10;
  const jumpStrength = 4.5;
  const gravity = 8;

  const handleKeyDown = useCallback((event) => {
    switch (event.code) {
      case 'KeyW':
      case 'ArrowUp':
        moveForward.current = true;
        break;
      case 'KeyA':
      case 'ArrowLeft':
        moveLeft.current = true;
        break;
      case 'KeyS':
      case 'ArrowDown':
        moveBackward.current = true;
        break;
      case 'KeyD':
      case 'ArrowRight':
        moveRight.current = true;
        break;
      case 'Space':
        if (canJump.current === true) {
          velocity.current.y = jumpStrength;
          canJump.current = false;
        }
        break;
      default:
        break;
    }
  }, []);

  const handleKeyUp = useCallback((event) => {
    switch (event.code) {
      case 'KeyW':
      case 'ArrowUp':
        moveForward.current = false;
        break;
      case 'KeyA':
      case 'ArrowLeft':
        moveLeft.current = false;
        break;
      case 'KeyS':
      case 'ArrowDown':
        moveBackward.current = false;
        break;
      case 'KeyD':
      case 'ArrowRight':
        moveRight.current = false;
        break;
      default:
        break;
    }
  }, []);

  useFrame((state, delta) => {
    direction.current.set(0, 0, 0);
    
    const forward = new THREE.Vector3();
    state.camera.getWorldDirection(forward);
    forward.y = 0;
    forward.normalize();

    const right = new THREE.Vector3();
    right.crossVectors(forward, new THREE.Vector3(0, 1, 0));

    if (moveForward.current) {
      direction.current.add(forward);
    }
    if (moveBackward.current) {
      direction.current.sub(forward);
    }
    if (moveLeft.current) {
      direction.current.sub(right);
    }
    if (moveRight.current) {
      direction.current.add(right);
    }

    if (direction.current.length() > 0) {
      direction.current.normalize();
      direction.current.multiplyScalar(speed * delta);
    }

    velocity.current.x = direction.current.x;
    velocity.current.z = direction.current.z;
    velocity.current.y -= gravity * delta;

    state.camera.position.x += velocity.current.x;
    state.camera.position.y += velocity.current.y;
    state.camera.position.z += velocity.current.z;

    if (state.camera.position.y < 1) {
      state.camera.position.y = 1;
      velocity.current.y = 0;
      canJump.current = true;
    }
  });

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    document.addEventListener('keyup', handleKeyUp);

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.removeEventListener('keyup', handleKeyUp);
    };
  }, [handleKeyDown, handleKeyUp]);

  return <Player onPositionUpdate={onPositionUpdate} />;
}

// API Service Functions
//const API_BASE_URL = 'http://172.16.118.72:8081/api';

async function apiCall(endpoint, data, serverAddr) {
  var temp = serverAddr
  const transformed = 'http://' + temp.replace(':9000', ':8081/api');
  try {
    const response = await fetch(`${transformed}${endpoint}`, {
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

// Generate player ID
const generatePlayerId = () => {
  // let playerId = localStorage.getItem('playerId');
  // if (!playerId) {
  //   playerId = "1"
  //   localStorage.setItem('playerId', playerId);
  // }
  return "1";
};

// Generate empty chunk data
const generateEmptyChunk = (chunkX, chunkZ, chunkSize = 10) => {
  return {
    cells: Array(chunkSize).fill().map(() => Array(chunkSize).fill(false)),
    offsetX: chunkX * chunkSize,
    offsetZ: chunkZ * chunkSize,
    chunkX,
    chunkZ,
    isEmpty: true
  };
};



 

// Main Game
export default function Game() {
  const [cubes, setCubes] = useState([]);
  const [selectedColor, setSelectedColor] = useState('#ff0000');
  const [chunks, setChunks] = useState({});
  const [playerPosition, setPlayerPosition] = useState([0, 1, 0]);
  const [gameStarted, setGameStarted] = useState(false);
  const [playerId] = useState(generatePlayerId());
  const [serverStatus, setServerStatus] = useState('Disconnected');
  const [serverAddr, setServerAddr] = useState('')
  const [lastUpdate, setLastUpdate] = useState(null);
  const [currentChunk, setCurrentChunk] = useState({ IDX: 0, IDY: 0 });
  const controlsRef = useRef();

  const chunkSize = 32;




  const initializeServerConnection = useCallback(async () => {
    try {
      const data = {
        player_id: playerId
      };
      
      const response = await fetch('http://172.16.118.72:8080/join', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(data),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const result = await response.json();
      console.log(result.message);
      setServerAddr(result.message);
      
    } catch (error) {
      console.error('Failed to connect to central server:', error);
      setServerStatus('Connection Failed - Using Local');
    }
  }, [playerId,serverAddr]);

  // Initialize server connection only once when component mounts
  useEffect(() => {
    initializeServerConnection();
  }, [initializeServerConnection]);
  // Prevent default browser context menu
  useEffect(() => {
    const handleContextMenu = (e) => {
      if (gameStarted) e.preventDefault();
    };
    document.addEventListener('contextmenu', handleContextMenu);
    return () => document.removeEventListener('contextmenu', handleContextMenu);
  }, [gameStarted]);

  // Convert 3D position to chunk coordinates
  const getChunkIdFromPosition = (x, z, size = chunkSize) => {
    const chunkX = Math.floor(x / size);
    const chunkZ = Math.floor(z / size);
    return { IDX: chunkX, IDY: chunkZ };
  };

  // Get chunk key for object storage
  const getChunkKey = (chunkId) => `${chunkId.IDX},${chunkId.IDY}`;

  // Get current player chunk ID
  const getCurrentChunkId = useCallback(() => {
    return getChunkIdFromPosition(playerPosition[0], playerPosition[2]);
  }, [playerPosition]);

  // Load or create chunk
  const loadChunk = useCallback(async (chunkId) => {
    const chunkKey = getChunkKey(chunkId);
    //console.log(chunks)
    // If chunk already exists, return it
    // if (chunks[chunkKey]) {
    //   return chunks[chunkKey];
    // }

    var currentChunkID = getChunkIdFromPosition(playerPosition[0],playerPosition[2])

    console.log(chunkId)
    var chunkX = Math.round(chunkId.IDX)
    var chunkY = Math.round(chunkId.IDY) 

    console.log(chunkX,chunkY)
    var chunk_id = {
      id_x : chunkId.IDX,
      id_y : chunkId.IDY
    }
    try {
      // Try to get chunk data from server
      const response = await apiCall('/player/data', {
        player_id : playerId,
        chunk_id :chunk_id,
        player: {
          ID: playerId,
          PosX: Math.round(playerPosition[0]),
          PosY: Math.round(playerPosition[2]),
          chunk_id : chunk_id
        }
      },serverAddr);
      console.log(response)
      if (response.success) {
        // Server returned chunk data
        setServerStatus('Connected');
        if(response.message != serverAddr){
          setServerAddr(response.message)

           const new_response = await apiCall('/player/data', {
            player_id : playerId,
            chunk_id :chunk_id,
            player: {
              ID: playerId,
              PosX: Math.round(playerPosition[0]),
              PosY: Math.round(playerPosition[2]),
              chunk_id : chunk_id
            }
          },response.message);
          console.log(new_response)

           var updated_cubes = []
        response.data.cells.forEach((cube) => {
          const position = [cube.x, cube.height, cube.z]
          var new_cube = {
            cube_id : cube.cube_id, 
            position : position, 
            color : cube.color
          }

          updated_cubes.push(new_cube)
        })
    
          const serverChunk = {
          cells:  Array(chunkSize).fill().map(() => Array(chunkSize).fill(false)),
          offsetX: chunkId.IDX * chunkSize,
          offsetZ: chunkId.IDY * chunkSize,
          chunkX: chunkId.IDX,
          chunkZ: chunkId.IDY,
          isEmpty: false,
          cubes : updated_cubes
        };

       
        //setCubes(prev => [...prev, ...updated_cubes])
        
        setChunks(prev => ({
          ...prev,
          [chunkKey]: serverChunk
        }));
        
        if(currentChunkID.IDX === chunkId.IDX  && currentChunkID.IDY === chunkId.IDY){
          setCubes(updated_cubes)
        }

      //  setCubes((prev) => [...prev, ...updated_cubes])
       // setCubes(updated_cubes)
       // setServerAddr(response.message)
      //    if (new_response.data.cubes) {
      //   updateCubesFromChunk(response.data.cubes, chunkId);
      // }
        
        return serverChunk;

        }

        var updated_cubes = []
        response.data.cells.forEach((cube) => {
          const position = [cube.x, cube.height, cube.z]
          var new_cube = {
            cube_id : cube.cube_id, 
            position : position, 
            color : cube.color
          }

          updated_cubes.push(new_cube)
        })
        
        const serverChunk = {
          cells: Array(chunkSize).fill().map(() => Array(chunkSize).fill(false)),
          offsetX: chunkId.IDX * chunkSize,
          offsetZ: chunkId.IDY * chunkSize,
          chunkX: chunkId.IDX,
          chunkZ: chunkId.IDY,
          isEmpty: false,
          cubes : updated_cubes
        };
        
       
       // setCubes(prev => [...prev, ...updated_cubes])
        setChunks(prev => ({
          ...prev,
          [chunkKey]: serverChunk
        }));

        if(currentChunkID.IDX === chunkId.IDX  && currentChunkID.IDY === chunkId.IDY){
          setCubes(updated_cubes)
        }

    //    setCubes((prev) => [...prev, ...updated_cubes])
      //  setCubes(updated_cubes)
       // setServerAddr(response.message)

      //    if (response.data.cubes) {
      //   updateCubesFromChunk(response.data.cubes, chunkId);
      // }
        
        return serverChunk;
      } else {
        // Server returned empty or no data, create empty chunk
        throw new Error('No chunk data from server');
      }
    } catch (error) {
      // Create empty chunk if server fails or returns no data
      console.log(`Creating empty chunk at ${chunkKey}`);
      const emptyChunk = generateEmptyChunk(chunkId.IDX, chunkId.IDY, chunkSize);
      
      setChunks(prev => ({
        ...prev,
        [chunkKey]: emptyChunk
      }));
      
      return emptyChunk;
    }
  }, [chunks, playerId, playerPosition,serverAddr]);

  // Load surrounding chunks when player moves
  // change this according to specs
  const loadSurroundingChunks = useCallback(async (centerChunkId) => {
    const loadRadius = 1; // Load chunks within 1 chunk distance


    
    for (let x = -loadRadius; x <= loadRadius; x++) {
      for (let z = -loadRadius; z <= loadRadius; z++) {
        const chunkId = {
          IDX: centerChunkId.IDX + x,
          IDY: centerChunkId.IDY + z
        };
        await loadChunk(chunkId);
      }
    }
  }, [loadChunk]);

  // Poll server for updates and handle chunk loading
  const pollServerForUpdates = useCallback(async () => {
    if (!gameStarted) return;

  //  try {
      const currentChunkId = getCurrentChunkId();
      setCurrentChunk(currentChunkId);
      

      await loadSurroundingChunks(currentChunkId);

      // Update server with player position
//       const response = await apiCall('/player/move', {
//         player_id: playerId,
//         x: Math.round(playerPosition[0]),
//         y: Math.round(playerPosition[2]),
//         chunk_id: currentChunkId
//       },serverAddr);
//       console.log(response)
// if (response.success) {
//       setServerStatus('Connected');
//       setLastUpdate(new Date().toLocaleTimeString());

//       if (response.data){

//         const chunkKey = getChunkKey(currentChunkId)

//         var focus_chunk = chunks[chunkKey]
//         var updated_cubes = []
//         // see for this optimization 2 loops inducing lag hence maybe increase the poll interval 
//         for( var cube of response.data.cells ){
//             var ok = true 

//             for (var compare_cube of focus_chunk.cubes){
//               if( cube.cube_id === compare_cube.cube_id){
//                   ok = false
//                   break
//               }
//             }

//             if(ok){
//               var new_cube = {
//                 cube_id : cube.cube_id, 
//                 position : [cube.x, cube.height, cube.z],
//                 color : cube.color 
//               }
//               updated_cubes.push(new_cube)
//             }
//         }

//         focus_chunk.cubes = [...focus_chunk.cubes, ...updated_cubes]
//         chunks[chunkKey] = focus_chunk
//         setChunks(chunks)
//       }
    
//       //  loadChunk(currentChunkId)
//     //  if (response.data) {
//     //     const updatedChunks = {};
//     //     const updatedCubes = [];

//     //     // Iterate through all chunks sent from server
//     //     //for (const chunk of response.data.chunks) {
//     //     var chunk = response.data
//     //       const chunkKey = `${chunk.id_x},${chunk.id_y}`;

//     //       // Update chunk info
//     //       updatedChunks[chunkKey] = {
//     //         cells: Array(chunkSize).fill().map(() => Array(chunkSize).fill(false)),
//     //         offsetX: chunk.id_x * chunkSize,
//     //         offsetZ: chunk.id_y * chunkSize,
//     //         chunkX: chunk.id_x,
//     //         chunkZ: chunk.id_y,
//     //         isEmpty: false,
//     //         cubes : chunk.cells
//     //       };

//     //       // Extract cubes for this chunk
//     //       if (chunk.cells && Array.isArray(chunk.cells)) {
//     //         chunk.cells.forEach((cube) => {
//     //           updatedCubes.push({
//     //             id: `server_cube_${cube.x}_${cube.z}_${cube.height}`,
//     //             position: [cube.x, cube.height, cube.z],
//     //             color: cube.color || '#ff0000'
//     //           });
//     //         });
//     //       }
//     // //    }


//     //     setChunks((prev) => ({
//     //       ...prev,
//     //       ...updatedChunks
//     //     }));

//         // setCubes(updatedCubes)
//         // setCubes((prev) => {
//         //   const local = prev.filter((cube) => !cube.id.startsWith('server_cube_'));
//         //   const seen = new Set();
//         //   const merged = [];

//         //   for (const cube of [...local, ...updatedCubes]) {
//         //     const key = `${cube.position[0]}_${cube.position[1]}_${cube.position[2]}`;
//         //     if (!seen.has(key)) {
//         //       merged.push(cube);
//         //       seen.add(key);
//         //     }
//         //   }

//         //   return merged;
//         // });
//      // }
//   }
//     else {
//       setServerStatus('Error: ' + response.message);
//     }
//   } catch (error) {
//     console.error('Polling error:', error);
//     setServerStatus('Connection Failed - Using Local');
//   }

}, [gameStarted, playerId, playerPosition, getCurrentChunkId, loadSurroundingChunks, serverAddr]);

  // Setup polling interval
  useEffect(() => {
    if (!gameStarted) return;

    const interval = setInterval(pollServerForUpdates, 5000); // Poll every 5 seconds
    //const chunkUpdateInterval = setInterval(checkChunkUpdates, 1000); // Check every second
    return () => {
      clearInterval(interval);
   //   clearInterval(chunkUpdateInterval);
    }
  }, [gameStarted, pollServerForUpdates]);

  // Handle cell clicks with chunk awareness
  const handleCellLeftClick = async (x, z, offsetX, offsetZ) => {
    if (!gameStarted) return;

    const worldX = x + offsetX;
    const worldZ = z + offsetZ;
    const chunkId = getChunkIdFromPosition(playerPosition[0],playerPosition[2])
    const chunkKey = getChunkKey(chunkId)

    var focus_chunk = chunks[chunkKey]
    console.log(focus_chunk)
    //var cubes = focus_chunk.cubes
    // Find the highest cube at this position
    const existingCubes = focus_chunk.cubes.filter(cube => 
      Math.round(cube.position[0]) === worldX && 
      Math.round(cube.position[2]) === worldZ
    );
    
    const height = existingCubes.length > 0 
      ? Math.max(...existingCubes.map(cube => cube.position[1])) + 1 
      : 0;

    const position = [worldX, height, worldZ];

    const newCube = {
      cube_id: `cube_${worldX}_${worldZ}_${height}`,
      position,
      color: selectedColor
    };

    focus_chunk.cubes.push(newCube)
    chunks[chunkKey] = focus_chunk
    setChunks(chunks)

    setCubes((prev) => [...prev, newCube]);

    
    console.log('Placed cube at:', position);

    // send server the update about cube placed in this chunk 

    var new_cube = {
      cube_id : newCube.cube_id, 
      x : worldX, 
      z : worldZ,
      height : height, 
      color : selectedColor
    }

    var chunkid = getCurrentChunkId()
    var current_chunk_id = {
      id_x : chunkid.IDX,
      id_y : chunkid.IDY
    }

    const add_req = {
        cube : new_cube, 
        chunk_id : current_chunk_id
   }

   // try {
      // Try to get chunk data from server
      const response = await apiCall('/player/addcube', add_req,serverAddr);
      console.log(response)
      if (response.success) {
        console.log(response.message)
      } else {
        // Server returned empty or no data, create empty chunk
        throw new Error('Cube not added in server');
      }
    //}

  };

  const handleCellRightClick = async (x, z, offsetX, offsetZ) => {
    if (!gameStarted) return;

    const worldX = x + offsetX;
    const worldZ = z + offsetZ;

    // Find all cubes at this world position
    const cubesAtPosition = cubes.filter(
      (cube) =>
        Math.round(cube.position[0]) === worldX &&
        Math.round(cube.position[2]) === worldZ
    );

    if (cubesAtPosition.length > 0) {
      // Remove the top cube
      const topCube = cubesAtPosition.reduce((highest, cube) =>
        cube.position[1] > highest.position[1] ? cube : highest
      );
      setCubes((prev) => prev.filter((cube) => cube.cube_id !== topCube.cube_id));
      console.log('Removed cube at:', worldX, worldZ);

      // send server the update to server that cube got removed 

      var chunkid = getCurrentChunkId()
      var current_chunk_id = {
        id_x : chunkid.IDX,
        id_y : chunkid.IDY
      }

      const dlt_req = {
            cube_id : topCube.cube_id, 
            chunk_id : current_chunk_id
      }

      const response = await apiCall('/player/dltcube', dlt_req,serverAddr);
      console.log(response)
      if (response.success) {
        console.log(response.message)
      } else {
        // Server returned empty or no data, create empty chunk
        throw new Error('Cube not added in server');
      }
    }
  };

  const handleCubeRightClick = async (cubeId) => {
    if (!gameStarted) return;
    setCubes((prev) => prev.filter((cube) => cube.cube_id !== cubeId));
    console.log('Removed cube:', cubeId);

     var chunkid = getCurrentChunkId()
      var current_chunk_id = {
        id_x : chunkid.IDX,
        id_y : chunkid.IDY
      }

      const dlt_req = {
            cube_id : cubeId, 
            chunk_id : current_chunk_id
      }

    const response = await apiCall('/player/dltcube', dlt_req,serverAddr);
      console.log(response)
      if (response.success) {
        console.log(response.message)
      } else {
        // Server returned empty or no data, create empty chunk
        throw new Error('Cube not added in server');
      }


  };

  const handlePositionUpdate = (newPosition) => {
    setPlayerPosition(newPosition);
  };

  const handleGameStart = async () => {
    setGameStarted(true);
    if (controlsRef.current) controlsRef.current.lock();



    
    // Load initial chunk when game starts
    const initialChunkId = getChunkIdFromPosition(0, 0);
    await loadSurroundingChunks(initialChunkId);
    
    // Start polling
    pollServerForUpdates();
  };

  const colors = [
    '#ff0000',
    '#00ff00',
    '#0000ff',
    '#ffff00',
    '#ff00ff',
    '#00ffff',
    '#ffffff',
    '#888888'
  ];

  return (
    <div style={{ width: '100vw', height: '100vh', position: 'relative' }}>
      <Canvas
        camera={{ position: [0, 1, 5], fov: 75 }}
        style={{ background: 'linear-gradient(to bottom, #87CEEB, #98FB98)' }}
        onContextMenu={(e) => e.preventDefault()}
      >
        <ambientLight intensity={0.6} />
        <pointLight position={[10, 10, 10]} intensity={1} />
        <directionalLight position={[-10, 10, -10]} intensity={0.5} />

        <Grid
          chunks={chunks}
          onCellClick={handleCellLeftClick}
          onCellRightClick={handleCellRightClick}
          playerPosition={playerPosition}
        />

        {cubes.map((cube) => (
          <Cube
            key={cube.id}
            position={cube.position}
            color={cube.color}
            onRightClick={() => handleCubeRightClick(cube.cube_id)}
            playerPosition={playerPosition}
          />
        ))}

        <FirstPersonController onPositionUpdate={handlePositionUpdate} />
        <PointerLockControls ref={controlsRef} />
      </Canvas>

      <Crosshair />

      {/* Control Panel */}
      {gameStarted && (
        <div
          style={{
            position: 'absolute',
            top: '20px',
            left: '20px',
            background: 'rgba(0,0,0,0.7)',
            padding: '20px',
            borderRadius: '10px',
            color: 'white',
            fontFamily: 'Arial, sans-serif',
            maxWidth: '300px'
          }}
        >
          <h3>Controls</h3>
          <p>WASD: Move | Space: Jump</p>
          <p>Mouse: Look around</p>
          <p>Left Click: Place cube</p>
          <p style={{ color: '#ff4444' }}>Right Click: Remove cube</p>

          {/* Server Status */}
          <div style={{ marginTop: '20px', padding: '10px', background: 'rgba(255,255,255,0.1)', borderRadius: '5px' }}>
            <h4>Server Status</h4>
            <p>Status: <span style={{ 
              color: serverStatus === 'Connected' ? '#4CAF50' : 
                     serverStatus.includes('Using Local') ? '#FF9800' : '#f44336'
            }}>{serverStatus}</span></p>
            <p>Player ID: {playerId}</p>
            <p>Current Chunk: [{currentChunk.IDX}, {currentChunk.IDY}]</p>
            <p>Loaded Chunks: {Object.keys(chunks).length}</p>
            {lastUpdate && <p>Last update: {lastUpdate}</p>}
          </div>

          <div style={{ marginTop: '20px' }}>
            <h4>Cube Color</h4>
            <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap', maxWidth: '200px' }}>
              {colors.map((color) => (
                <div
                  key={color}
                  onClick={() => setSelectedColor(color)}
                  style={{
                    width: '25px',
                    height: '25px',
                    backgroundColor: color,
                    border: selectedColor === color ? '3px solid white' : '1px solid #ccc',
                    cursor: 'pointer',
                    borderRadius: '4px'
                  }}
                />
              ))}
            </div>
            <p style={{ marginTop: '10px', fontSize: '12px' }}>
              Selected:{' '}
              <span style={{ color: selectedColor, fontWeight: 'bold' }}>{selectedColor}</span>
            </p>
          </div>

          <div style={{ marginTop: '20px' }}>
            <p>Cubes placed: {cubes.length}</p>
            <p>Position: {playerPosition.map((p) => p.toFixed(1)).join(', ')}</p>
            <button
              onClick={() => setCubes([])}
              style={{
                padding: '8px 16px',
                background: '#ff4444',
                color: 'white',
                border: 'none',
                borderRadius: '4px',
                cursor: 'pointer',
                marginTop: '10px'
              }}
            >
              Clear All Cubes
            </button>
          </div>
        </div>
      )}

      {/* Click to start overlay */}
      {!gameStarted && (
        <div
          style={{
            position: 'absolute',
            top: 0,
            left: 0,
            width: '100%',
            height: '100%',
            background: 'rgba(0,0,0,0.9)',
            display: 'flex',
            flexDirection: 'column',
            justifyContent: 'center',
            alignItems: 'center',
            color: 'white',
            fontSize: '24px',
            cursor: 'pointer',
            zIndex: 1000
          }}
          onClick={handleGameStart}
        >
          <h1>3D Cube Builder</h1>
          <p style={{ fontSize: '18px', marginTop: '20px' }}>Click anywhere to start game</p>
          <div style={{ marginTop: '40px', textAlign: 'left', fontSize: '16px' }}>
            <p>🎮 <strong>Controls:</strong></p>
            <p>• WASD - Move around</p>
            <p>• Space - Jump</p>
            <p>• Mouse - Look around</p>
            <p>• <strong style={{ color: '#4CAF50' }}>Left Click</strong> - Place cube</p>
            <p>• <strong style={{ color: '#ff4444' }}>Right Click</strong> - Remove cube</p>
            <p style={{ marginTop: '20px', color: '#4CAF50' }}>
              🔄 <strong>Dynamic World:</strong> New chunks load as you explore
            </p>
            <p style={{ color: '#FF9800' }}>
              🌐 <strong>Multiplayer:</strong> Server syncs your position and chunk data
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
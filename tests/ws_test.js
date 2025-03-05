import http from 'k6/http';
import { check, sleep, group } from 'k6';
import ws from 'k6/ws';

const BASE_URL = 'http://172.18.0.4:8080/api';

// User details
const user1 = { username: 'user1', email: 'user1@test.com', password: 'password123' };
const user2 = { username: 'user2', email: 'user2@test.com', password: 'password123' };

export default function () {
    group('User Signup', () => {
        const user1Token = signup(user1);
        const user2Token = signup(user2);

        // User 1 creates a room
        const roomId = createRoom(user1Token, 'TestRoom');

        // Ensure room creation is successful before proceeding
        if (roomId) {
            // Both users join the room in parallel
            group('User1 and User2 Join Room', () => {
                let user1WebSocket = joinRoom(user1Token, roomId, 'Hello from User 1');
                let user2WebSocket = joinRoom(user2Token, roomId, 'Hello from User 2');
                
                // Wait for WebSocket connections to complete
                user1WebSocket.wait();
                user2WebSocket.wait();
            });
        }
    });

    sleep(5); // Wait before finishing the test
}

function signup(user) {
    const res = http.post(`${BASE_URL}/users/signup`, JSON.stringify(user), {
        headers: { 'Content-Type': 'application/json' },
    });

    check(res, {
        'Signup successful': (r) => r.status === 201,
    });

    if (res.status !== 201) {
        console.error(`Signup failed for ${user.username}:`, res.body);
        return null;
    }

    return JSON.parse(res.body).token;
}

function createRoom(token, roomName) {
    const res = http.post(`${BASE_URL}/ws/create-room`, JSON.stringify({ name: roomName }), {
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    });

    check(res, {
        'Room created successfully': (r) => r.status === 201,
    });

    if (res.status !== 201) {
        console.error('Room creation failed:', res.body);
        return null;
    }

    return JSON.parse(res.body).id;
}

function joinRoom(token, roomId, message) {
    const url = `ws://172.18.0.4:8080/api/ws/join-room/${roomId}`;

    // Establish WebSocket connection
    let res = ws.connect(url, { headers: { Authorization: `Bearer ${token}` } }, function (socket) {
        socket.on('open', function () {
            console.log(`User joined room ${roomId}`);
            socket.send(message);
        });

        socket.on('message', function (msg) {
            console.log(`Received: ${msg}`);
        });

        socket.on('close', function () {
            console.log(`Disconnected from room ${roomId}`);
        });

        socket.setTimeout(function () {
            socket.close();
        }, 5000);
    });

    check(res, {
        'WebSocket connection successful': (r) => r && r.status === 101,
    });

    return res;
}

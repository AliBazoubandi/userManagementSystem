import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
    vus: 1000,  // 1000 virtual users
    duration: '10s',  // Test for 10 seconds
};

let BASE_URL = 'http://172.18.0.4:8080/api';

export default function () {
    // 1. Sign Up
    let signupRes = http.post(`${BASE_URL}/users/signup`, JSON.stringify({
        username: `user_${__VU}`,  // Unique per virtual user
        email: `user_${__VU}@example.com`,
        password: "password123",
        age: 23,
        room_id: null
    }), { headers: { 'Content-Type': 'application/json' } });

    console.log(`Signup response: ${signupRes.status} ${signupRes.body}`);
    check(signupRes, { "Signup successful": (res) => res.status === 201 });

    // 2. Log In
    let loginRes = http.post(`${BASE_URL}/users/login`, JSON.stringify({
        username: `user_${__VU}`,
        password: "password123",
    }), { headers: { 'Content-Type': 'application/json' } });

    console.log(`Login response: ${loginRes.status} ${loginRes.body}`);
    check(loginRes, { "Login successful": (res) => res.status === 200 });

    let authToken = loginRes.json().token;
    let authHeaders = { headers: { 'Authorization': `Bearer ${authToken}` } };

    // 3. Fetch User Data
    let userRes = http.get(`${BASE_URL}/users`, authHeaders);
    console.log(`Get response: ${userRes.status} ${userRes.body}`);
    check(userRes, { "Fetched users": (res) => res.status === 200 });

    sleep(1);
}

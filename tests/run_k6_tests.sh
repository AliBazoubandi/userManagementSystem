#!/bin/sh

# Run HTTP test first
k6 run /k6-tests/tests/http_test.js

# Run WebSocket test next
k6 run /k6-tests/tests/ws_test.js

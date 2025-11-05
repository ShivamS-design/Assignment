#!/usr/bin/env python3
import http.server
import socketserver
import json
import os
from urllib.parse import urlparse, parse_qs

class WASMHandler(http.server.SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory="frontend", **kwargs)
    
    def do_GET(self):
        if self.path.startswith('/api/'):
            self.handle_api()
        elif self.path == '/health':
            self.send_response(200)
            self.send_header('Content-type', 'text/plain')
            self.end_headers()
            self.wfile.write(b'OK')
        else:
            super().do_GET()
    
    def do_POST(self):
        if self.path.startswith('/api/'):
            self.handle_api()
        else:
            self.send_error(404)
    
    def handle_api(self):
        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.send_header('Access-Control-Allow-Origin', '*')
        self.send_header('Access-Control-Allow-Methods', 'GET, POST, OPTIONS')
        self.send_header('Access-Control-Allow-Headers', 'Content-Type, Authorization')
        self.end_headers()
        
        if self.path == '/api/v1/auth/login':
            response = {
                "token": "demo-token-123",
                "expires_at": "2024-12-31T23:59:59Z",
                "user": {"id": "admin", "username": "admin", "roles": ["admin"]}
            }
        elif self.path == '/api/v1/modules':
            if self.command == 'GET':
                response = {"modules": [], "total": 0, "page": 1, "limit": 20}
            else:
                response = {"module_id": "demo-module-123", "name": "uploaded-module", "size": 1024}
        else:
            response = {"status": "success", "message": "Demo API response"}
        
        self.wfile.write(json.dumps(response).encode())

if __name__ == "__main__":
    PORT = 8080
    with socketserver.TCPServer(("", PORT), WASMHandler) as httpd:
        print(f"🚀 WASM-as-OS Demo Server running at:")
        print(f"   Frontend: http://localhost:{PORT}")
        print(f"   API: http://localhost:{PORT}/api/v1/")
        print(f"   Health: http://localhost:{PORT}/health")
        print(f"\nPress Ctrl+C to stop")
        httpd.serve_forever()
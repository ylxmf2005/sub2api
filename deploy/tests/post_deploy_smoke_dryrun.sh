#!/usr/bin/env bash

set -euo pipefail

temp_dir="$(mktemp -d)"
trap 'rm -rf "${temp_dir}"' EXIT

port=18080
cat > "${temp_dir}/server.py" <<'EOF'
from http.server import BaseHTTPRequestHandler, HTTPServer

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/health":
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b"ok")
            return
        self.send_response(404)
        self.end_headers()

    def log_message(self, format, *args):
        pass

HTTPServer(("127.0.0.1", 18080), Handler).serve_forever()
EOF

python3 "${temp_dir}/server.py" &
server_pid=$!
trap 'kill ${server_pid} 2>/dev/null || true; rm -rf "${temp_dir}"' EXIT
sleep 1

./deploy/scripts/post_deploy_smoke.sh --base-url "http://127.0.0.1:${port}" >/dev/null

kill "${server_pid}" 2>/dev/null || true
wait "${server_pid}" 2>/dev/null || true

echo "post deploy smoke dry-run passed"

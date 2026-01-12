import http.server
import socketserver
import urllib.request
import urllib.error
import sys
import os

# usage: python3 cors_proxy.py https://my-resource.openai.azure.com

PORT = 9000

if len(sys.argv) < 2:
    print("Error: Missing Azure Endpoint argument.")
    print("Usage: python3 cors_proxy.py <AZURE_ENDPOINT_URL>")
    print("Example: python3 cors_proxy.py https://my-resource.openai.azure.com")
    sys.exit(1)

TARGET_BASE_URL = sys.argv[1]
if TARGET_BASE_URL.endswith('/'):
    TARGET_BASE_URL = TARGET_BASE_URL[:-1]

class ProxyHandler(http.server.SimpleHTTPRequestHandler):
    def do_OPTIONS(self):
        self.send_response(200, "ok")
        self.send_header('Access-Control-Allow-Origin', '*')
        self.send_header('Access-Control-Allow-Methods', 'GET, POST, OPTIONS')
        self.send_header('Access-Control-Allow-Headers', 'content-type, api-key')
        self.end_headers()

    def do_POST(self):
        content_length = int(self.headers.get('Content-Length', 0))
        post_data = self.rfile.read(content_length)

        target_url = f"{TARGET_BASE_URL}{self.path}"
        print(f"Proxying POST to: {target_url}")

        # Create Request
        req = urllib.request.Request(target_url, data=post_data, method='POST')

        # Forward Headers
        if 'Content-Type' in self.headers:
            req.add_header('Content-Type', self.headers['Content-Type'])
        if 'api-key' in self.headers:
            req.add_header('api-key', self.headers['api-key'])

        try:
            with urllib.request.urlopen(req) as response:
                resp_data = response.read()
                self.send_response(response.status)
                self.send_header('Access-Control-Allow-Origin', '*')
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(resp_data)
        except urllib.error.HTTPError as e:
            print(f"Azure Error: {e.code} - {e.reason}")
            self.send_response(e.code)
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            self.wfile.write(e.read())
        except Exception as e:
            print(f"Proxy Error: {e}")
            self.send_response(500)
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            self.wfile.write(str(e).encode('utf-8'))

print(f"Starting CORS Proxy on port {PORT}")
print(f"Target: {TARGET_BASE_URL}")
print("----------------------------------------------------------------")
print(f"1. Keep this terminal open.")
print(f"2. In the App, set Endpoint URL to: http://localhost:{PORT}")
print("----------------------------------------------------------------")

with socketserver.TCPServer(("", PORT), ProxyHandler) as httpd:
    httpd.serve_forever()

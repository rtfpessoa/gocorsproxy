package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var allowedOrigins = []string{
	// "*", // allow all (not recommended)
	"https://debridui-alt.vercel.app",
}

func originAllowed(origin string) bool {
	if origin == "" {
		return true
	}
	for _, o := range allowedOrigins {
		if o == "*" || o == origin {
			return true
		}
	}
	return false
}

func allowOriginValue(origin string) string {
	for _, o := range allowedOrigins {
		if o == "*" {
			return "*"
		}
	}
	return origin
}

func corsProxyHandler(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")

	// Handle OPTIONS preflight
	if r.Method == http.MethodOptions {
		if !originAllowed(origin) {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", allowOriginValue(origin))
		w.Header().Set("Access-Control-Allow-Methods", "GET,HEAD,POST,PUT,DELETE,PATCH,OPTIONS")
		w.Header().Set(
			"Access-Control-Allow-Headers",
			r.Header.Get("Access-Control-Request-Headers"),
		)
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	target := r.URL.Query().Get("url")
	if target == "" {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, demoHTML)
		return
	}

	if !originAllowed(origin) {
		http.Error(w, "Origin not allowed", http.StatusForbidden)
		return
	}

	parsedTarget, err := url.Parse(target)
	if err != nil {
		http.Error(w, "Invalid target URL", http.StatusBadRequest)
		return
	}

	// Create upstream request
	req, err := http.NewRequest(r.Method, parsedTarget.String(), r.Body)
	if err != nil {
		http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
		return
	}

	// Copy headers
	for k, vv := range r.Header {
		if strings.ToLower(k) == "host" {
			continue
		}
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		w.Header().Set("Access-Control-Allow-Origin", allowOriginValue(origin))
		http.Error(w, "Proxy error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Copy response headers (minus CORS headers)
	for k, vv := range resp.Header {
		lk := strings.ToLower(k)
		if strings.HasPrefix(lk, "access-control-") {
			continue
		}
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	// Set our CORS headers
	w.Header().Set("Access-Control-Allow-Origin", allowOriginValue(origin))
	w.Header().Set("Access-Control-Allow-Methods", "GET,HEAD,POST,PUT,DELETE,PATCH,OPTIONS")
	w.Header().Set("Vary", "Origin")

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func main() {
	http.HandleFunc("/", corsProxyHandler)

	log.Println("CORS proxy running on http://localhost:8111")
	log.Fatal(http.ListenAndServe(":8111", nil))
}

const demoHTML = `<!DOCTYPE html>
<html>
<head>
<title>Universal CORS Proxy</title>
<style>
body{font-family:Arial,sans-serif;max-width:800px;margin:0 auto;padding:20px}
code{background:#f4f4f4;padding:10px;display:block;margin:10px 0}
.status{font-weight:bold}
.error{color:red}
.success{color:green}
</style>
</head>
<body>
<h1>Universal CORS Proxy</h1>
<p>Make cross-origin requests to any API.</p>
<h2>Usage</h2>
<code>http://localhost:8080/?url=https://api.example.com/endpoint</code>
<h2>Test</h2>
<input type="url" id="u" placeholder="API URL" style="width:400px;padding:5px">
<button onclick="t()">GET</button>
<button onclick="p()">POST</button>
<h3>Result:</h3>
<p class="status" id="s">Ready</p>
<code id="r">No requests made</code>
<script>
async function t(){
        const u=document.getElementById('u').value,s=document.getElementById('s'),r=document.getElementById('r');
        if(!u){s.textContent='Enter URL';s.className='status error';return}
        try{
                s.textContent='Loading...';s.className='status';
                const x=await fetch('/?url='+encodeURIComponent(u)),d=await x.text();
                s.textContent='Success: '+x.status;s.className='status success';r.textContent=d
        }catch(e){
                s.textContent='Error: '+e.message;s.className='status error';r.textContent=e
        }
}
async function p(){
        const u=document.getElementById('u').value,s=document.getElementById('s'),r=document.getElementById('r');
        if(!u){s.textContent='Enter URL';s.className='status error';return}
        try{
                s.textContent='Loading...';s.className='status';
                const x=await fetch('/?url='+encodeURIComponent(u),{
                        method:'POST',
                        headers:{'Content-Type':'application/json'},
                        body:JSON.stringify({msg:'Hello',ts:Date.now()})
                }),d=await x.text();
                s.textContent='Success: '+x.status;s.className='status success';r.textContent=d
        }catch(e){
                s.textContent='Error: '+e.message;s.className='status error';r.textContent=e
        }
}
</script>
</body>
</html>`

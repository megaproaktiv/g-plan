// Command mcpdrive is a tiny JSON-RPC 2.0 client for the Wails v3 MCP server.
//
// The Wails v3 app, when built with the `mcp` build tag, exposes an MCP server
// over streamable HTTP at http://127.0.0.1:9099/mcp. That transport is plain
// JSON-RPC 2.0, so it can be driven from Go with nothing but net/http and
// encoding/json — no Python needed.
//
// This replaces the throwaway /tmp Python helper used for live GUI checks.
//
// Usage (after starting the app: `go build -tags mcp -o bin/ptplan-mcp . && ./bin/ptplan-mcp &`):
//
//	go run ./cmd/mcpdrive call main.App.AddWorkPackage '["Spec","2026-01-01","2026-01-20"]'
//	go run ./cmd/mcpdrive js 'return document.querySelectorAll("svg.gantt text").length;'
//	go run ./cmd/mcpdrive tools            # list available MCP tools
//
// Flags:
//
//	-url   MCP endpoint (default http://127.0.0.1:9099/mcp)
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type rpcReq struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResp struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// client is a minimal MCP/JSON-RPC client. It performs the initialize
// handshake lazily on first use.
type client struct {
	url  string
	http *http.Client
	id   int
	init bool
}

func newClient(url string) *client {
	return &client{url: url, http: &http.Client{Timeout: 120 * time.Second}}
}

// post sends one JSON-RPC request. When wantResult is false (notifications) it
// ignores the body.
func (c *client) post(method string, params any, wantResult bool) (json.RawMessage, error) {
	req := rpcReq{JSONRPC: "2.0", Method: method, Params: params}
	if wantResult {
		c.id++
		req.ID = c.id
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest(http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	// The server may answer as a Server-Sent-Events stream or plain JSON.
	httpReq.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if !wantResult {
		return nil, nil
	}
	// Strip a leading SSE "data:" frame if present.
	text := strings.TrimSpace(string(raw))
	if strings.HasPrefix(text, "data:") {
		text = strings.TrimSpace(strings.TrimPrefix(text, "data:"))
	}
	var r rpcResp
	if err := json.Unmarshal([]byte(text), &r); err != nil {
		return nil, fmt.Errorf("decode %q: %w", text, err)
	}
	if r.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", r.Error.Code, r.Error.Message)
	}
	return r.Result, nil
}

// handshake runs initialize + the initialized notification once.
func (c *client) handshake() error {
	if c.init {
		return nil
	}
	_, err := c.post("initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "mcpdrive", "version": "1.0"},
	}, true)
	if err != nil {
		return err
	}
	if _, err := c.post("notifications/initialized", nil, false); err != nil {
		return err
	}
	c.init = true
	return nil
}

// callTool invokes an MCP tool by name with the given arguments.
func (c *client) callTool(name string, args map[string]any) (json.RawMessage, error) {
	if err := c.handshake(); err != nil {
		return nil, err
	}
	return c.post("tools/call", map[string]any{"name": name, "arguments": args}, true)
}

// toolText extracts the first content[].text field from a tools/call result
// (the shape MCP uses for js_eval / call_bound_method output).
func toolText(result json.RawMessage) string {
	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(result, &parsed); err != nil {
		return string(result)
	}
	if len(parsed.Content) > 0 {
		return parsed.Content[0].Text
	}
	return string(result)
}

func main() {
	url := flag.String("url", "http://127.0.0.1:9099/mcp", "MCP endpoint")
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mcpdrive [-url ...] <call|js|tools> [args]")
		fmt.Fprintln(os.Stderr, "  call <Method> <jsonArray>   run a bound Go method, e.g.")
		fmt.Fprintln(os.Stderr, `      mcpdrive call main.App.AddWorkPackage '["Spec","2026-01-01","2026-01-20"]'`)
		fmt.Fprintln(os.Stderr, "  js   <code>                 run JS in the window (use return to get a value)")
		fmt.Fprintln(os.Stderr, "  snap <outfile.png> [scale]  rasterize svg.gantt to a PNG file")
		fmt.Fprintln(os.Stderr, "  tools                       list available MCP tools")
		os.Exit(2)
	}

	c := newClient(*url)

	switch args[0] {
	case "tools":
		if err := c.handshake(); err != nil {
			fail(err)
		}
		res, err := c.post("tools/list", map[string]any{}, true)
		if err != nil {
			fail(err)
		}
		fmt.Println(string(res))

	case "call":
		if len(args) < 3 {
			fail(fmt.Errorf("call needs <Method> <jsonArray>"))
		}
		var callArgs []any
		if err := json.Unmarshal([]byte(args[2]), &callArgs); err != nil {
			fail(fmt.Errorf("args must be a JSON array: %w", err))
		}
		res, err := c.callTool("call_bound_method", map[string]any{"name": args[1], "args": callArgs})
		if err != nil {
			fail(err)
		}
		fmt.Println(toolText(res))

	case "js":
		if len(args) < 2 {
			fail(fmt.Errorf("js needs <code>"))
		}
		res, err := c.callTool("js_eval", map[string]any{"js": args[1]})
		if err != nil {
			fail(err)
		}
		fmt.Println(toolText(res))

	case "snap":
		// snap <outfile.png> [scale]: rasterize the on-screen svg.gantt to a PNG
		// file. Used to capture walkthrough screenshots. The SVG is cloned, a
		// white background is inlined, and editor-only chrome (.anchor) is kept
		// so selection borders show in docs; scale defaults to 2.
		if len(args) < 2 {
			fail(fmt.Errorf("snap needs <outfile.png> [scale]"))
		}
		scale := "2"
		if len(args) >= 3 {
			scale = args[2]
		}
		js := snapJS(scale)
		res, err := c.callTool("js_eval", map[string]any{"js": js})
		if err != nil {
			fail(err)
		}
		b64 := strings.TrimSpace(toolText(res))
		if b64 == "" || strings.HasPrefix(b64, "javascript error") || strings.HasPrefix(b64, "timed out") {
			fail(fmt.Errorf("snapshot failed: %s", b64))
		}
		png, derr := base64.StdEncoding.DecodeString(b64)
		if derr != nil {
			fail(fmt.Errorf("decode snapshot base64 (%d bytes): %w", len(b64), derr))
		}
		if werr := os.WriteFile(args[1], png, 0o644); werr != nil {
			fail(werr)
		}
		fmt.Printf("wrote %s (%d bytes)\n", args[1], len(png))

	default:
		fail(fmt.Errorf("unknown command %q", args[0]))
	}
}

// snapJS returns the JS that rasterizes the current on-screen Gantt SVG to a
// base64 PNG (no data-URL prefix), at the given scale.
func snapJS(scale string) string {
	return `
const svg = document.querySelector("svg.gantt");
if (!svg) return "";
const vb = svg.viewBox.baseVal;
const w = vb.width || svg.clientWidth || 1000;
const h = vb.height || svg.clientHeight || 600;
const SVGNS = "http://www.w3.org/2000/svg";
const clone = svg.cloneNode(true);
clone.setAttribute("width", String(w));
clone.setAttribute("height", String(h));
const bg = document.createElementNS(SVGNS, "rect");
bg.setAttribute("x","0"); bg.setAttribute("y","0");
bg.setAttribute("width", String(w)); bg.setAttribute("height", String(h));
bg.setAttribute("fill", "#ffffff");
clone.insertBefore(bg, clone.firstChild);
clone.querySelectorAll(".anchor").forEach((el) => el.remove());
const xml = new XMLSerializer().serializeToString(clone);
const svg64 = "data:image/svg+xml;base64," + btoa(unescape(encodeURIComponent(xml)));
const img = new Image();
await new Promise((res, rej) => { img.onload = () => res(); img.onerror = () => rej(new Error("svg load")); img.src = svg64; });
const scale = ` + scale + `;
const canvas = document.createElement("canvas");
canvas.width = Math.ceil(w*scale); canvas.height = Math.ceil(h*scale);
const ctx = canvas.getContext("2d");
ctx.imageSmoothingEnabled = true; ctx.imageSmoothingQuality = "high";
ctx.fillStyle = "#ffffff"; ctx.fillRect(0,0,canvas.width,canvas.height);
ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
return canvas.toDataURL("image/png").split(",",2)[1] || "";
`
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

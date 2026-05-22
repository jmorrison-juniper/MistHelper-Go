// Package main is the MistHelper-Go entry point.
// It wires the five internal packages (api, output, menu, ssh, web) and drives the
// interactive menu loop or a direct --menu N dispatch, then gracefully shuts down.
package main

import (
	"bufio"    // bufio.NewReader for stdin buffering in the menu dispatcher
	"context"  // context.Background and signal-aware context for lifecycle management
	"flag"     // CLI flag parsing for --menu, --format, --version
	"fmt"      // fmt.Errorf for error wrapping with context strings
	"log/slog" // structured logging (Go 1.21+ standard library)
	"os"       // os.Exit, os.Stdin, os.MkdirAll for process control and I/O
	"os/signal" // signal.NotifyContext for SIGINT/SIGTERM graceful shutdown
	"syscall"   // syscall.SIGTERM for OS-level termination signal
	"time"      // time.Second for shutdown timeout constants

	"github.com/joho/godotenv"                                      // .env file loader — optional, container injects vars directly
	"github.com/jmorrison-juniper/misthelper-go/internal/api"       // Mist API client and configuration
	"github.com/jmorrison-juniper/misthelper-go/internal/menu"      // TUI menu registry and dispatcher
	"github.com/jmorrison-juniper/misthelper-go/internal/output"    // CSV/SQLite output writer
	mssh "github.com/jmorrison-juniper/misthelper-go/internal/ssh"  // SSH server (aliased to avoid name clash)
	"github.com/jmorrison-juniper/misthelper-go/internal/web"       // HTTP status/health server
)

// version is the application version string in CHANGELOG YY.MM.DD.HH.MM (UTC) format.
const version = "26.05.22.00.00"

// appPackages holds all initialised package instances shared across the application lifecycle.
// Passed by value to helper functions so there is no global state.
type appPackages struct {
	cfg        api.Config       // runtime configuration loaded from env vars and CLI flags
	client     *api.Client      // Mist API client wrapping the mistapi-go SDK
	writer     output.Writer    // CSV or SQLite output backend for all menu operations
	registry   *menu.Registry   // registered menu entries; shared by menu loop and SSH sessions
	dispatcher *menu.Dispatcher // interactive menu controller wired to stdin
	sshServer  *mssh.Server     // SSH server on cfg.SSHPort (default 2200)
	webServer  *web.Server      // HTTP status and health server on cfg.WebPort (default 8055)
}

// stubOps defines all 89 menu operations as stubs pending Python→Go port.
// Replace each stub Handler with the real implementation when ported from MistHelper.py.
var stubOps = []struct {
	n    int
	cat  string
	name string
}{
	// Core organisation and site operations (1-4)
	{1, "Core", "List Organisation Sites"},
	{2, "Core", "List Organisation Devices"},
	{3, "Core", "List Organisation Inventory"},
	{4, "Core", "List Organisation Statistics"},
	// WebSocket real-time device commands (5-8)
	{5, "WebSocket", "Get Wireless Device Stats (Live)"},
	{6, "WebSocket", "Get Switch Stats (Live)"},
	{7, "WebSocket", "Get Gateway Stats (Live)"},
	{8, "WebSocket", "Get AP Neighbours (Live)"},
	// Packet captures (9-10)
	{9, "Captures", "Site Packet Capture"},
	{10, "Captures", "Org Packet Capture"},
	// Device inventory and events (11-20)
	{11, "Devices", "List Site Devices"},
	{12, "Devices", "List Site Device Events"},
	{13, "Devices", "List Site Device Stats"},
	{14, "Devices", "Get Org Device Events"},
	{15, "Devices", "List Org AP Stats"},
	{16, "Devices", "List Org Switch Stats"},
	{17, "Devices", "List Org Gateway Stats"},
	{18, "Devices", "Get Org Device Config History"},
	{19, "Devices", "List Org Gateway Config History"},
	{20, "Devices", "List Org Devices with Config"},
	// Sites and location (21-25)
	{21, "Sites", "List Sites with Location Data"},
	{22, "Sites", "List Sites with Gateways"},
	{23, "Sites", "List Sites Summary"},
	{24, "Sites", "Get Site Info"},
	{25, "Sites", "List Site Switch VC Stats"},
	// Inventory and licences (26-30)
	{26, "Inventory", "List Org Inventory"},
	{27, "Inventory", "Get Org Licences"},
	{28, "Inventory", "Get Org Licence Summary"},
	{29, "Inventory", "List Org Licence Subscriptions"},
	{30, "Inventory", "Get Org Security Policies"},
	// Templates and configuration (31-40)
	{31, "Config", "List Org AP Templates"},
	{32, "Config", "List Org RF Templates"},
	{33, "Config", "List Org Gateway Templates"},
	{34, "Config", "List Org Network Templates"},
	{35, "Config", "List Org WLAN Templates"},
	{36, "Config", "List Org Service Policies"},
	{37, "Config", "List Org Networks"},
	{38, "Config", "List Org VPNs"},
	{39, "Config", "List Org EVPN Topologies"},
	{40, "Config", "List Org Site Groups"},
	// Audit and history (41-50)
	{41, "Audit", "Search Org Audit Logs"},
	{42, "Audit", "Search Device Config History"},
	{43, "Audit", "List Org Webhooks"},
	{44, "Audit", "List Site Webhooks"},
	{45, "Audit", "List Org Alarms"},
	{46, "Audit", "List Site Alarms"},
	{47, "Audit", "List Org Device Uptime"},
	{48, "Audit", "List Org Port Stats"},
	{49, "Audit", "List Org BGP Stats"},
	{50, "Audit", "List Org OSPF Stats"},
	// Maps, SLE and advanced analytics (51-62)
	{51, "Analytics", "List Site Maps"},
	{52, "Analytics", "Get Site SLE Summary"},
	{53, "Analytics", "Get Site SLE Trend"},
	{54, "Analytics", "Get Org SLE Summary"},
	{55, "Analytics", "List Site SLE Classifiers"},
	{56, "Analytics", "Get Site WiFi SLE"},
	{57, "Analytics", "Get Site WAN SLE"},
	{58, "Analytics", "Get Site Wired SLE"},
	{59, "Analytics", "Get Org RRM Info"},
	{60, "Analytics", "List Rogue Devices"},
	{61, "Analytics", "List Site Rogue Devices"},
	{62, "Analytics", "List Org Insights"},
	// WIP / experimental features (63-65)
	{63, "WIP", "WIP Feature 63"},
	{64, "WIP", "WIP Feature 64"},
	{65, "WIP", "WIP Feature 65"},
	// Client and user data (66-72)
	{66, "Clients", "Search Wireless Clients"},
	{67, "Clients", "Search Wired Clients"},
	{68, "Clients", "Search WAN Clients"},
	{69, "Clients", "List Guest Authorisations"},
	{70, "Clients", "Search NAC Clients"},
	{71, "Clients", "List Client Events"},
	{72, "Clients", "List Client Stats"},
	// WLAN configuration (73-78)
	{73, "WLAN", "List Org WLANs"},
	{74, "WLAN", "List Site WLANs"},
	{75, "WLAN", "List Org PSKs"},
	{76, "WLAN", "List Site PSKs"},
	{77, "WLAN", "List Org NAC Rules"},
	{78, "WLAN", "List Org NAC Tags"},
	// RF and radio management (79-81)
	{79, "RF", "Get Site RF Info"},
	{80, "RF", "List Site AP Channels"},
	{81, "RF", "List Org RF Templates Detail"},
	// Admin and API tokens (82-86)
	{82, "Admin", "List Org API Tokens"},
	{83, "Admin", "List Org Admins"},
	{84, "Admin", "List Org Site Groups Detail"},
	{85, "Admin", "Get Org Info"},
	{86, "Admin", "List Org MX Edges"},
	// Additional WebSocket commands (87-89)
	{87, "WebSocket", "Get WAN Device Stats (Live)"},
	{88, "WebSocket", "Get Switch Port Stats (Live)"},
	{89, "WebSocket", "Get SSR Device Stats (Live)"},
}

// main is the application entry point.
// Parses flags, wires all packages, starts background servers, runs the menu, then shuts down.
func main() {
	menuFlag := flag.Int("menu", -1, "Run a menu option directly (-1=interactive, 0=quit, N=operation N)") // --menu flag for direct dispatch or clean-quit
	formatFlag := flag.String("format", "", "Output format: csv or sqlite (overrides OUTPUT_FORMAT)")      // --format overrides the OUTPUT_FORMAT env var
	showVersion := flag.Bool("version", false, "Print version and exit")                                   // --version prints the build version string
	flag.Parse()                                                                                           // parse before any side effects that read flags

	if *showVersion {                                            // handle --version before loading credentials — no .env needed
		slog.Info("MistHelper-Go", "version", version) // emit structured version log; slog writes to stderr by default
		os.Exit(0)                                     // clean exit: container health checks can call --version safely
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM) // root context; cancelled on Ctrl+C or SIGTERM
	defer stop()                                                                           // release signal resources and cancel ctx when main returns

	pkgs, err := initPackages(*formatFlag) // load config, wire all five packages; may write data/ssh_host_rsa_key
	if err != nil {
		slog.Error("initialisation failed", "error", err) // log before os.Exit so container logs capture the cause
		os.Exit(1)                                        // non-zero exit signals container orchestration to restart
	}
	defer func() { // flush and close the output backend on any exit path
		if err := pkgs.writer.Close(); err != nil { // check the close error to catch write-flush failures
			slog.Error("failed to close output writer", "error", err) // log so operators know data may be incomplete
		}
	}()

	registerStubs(pkgs.registry)  // populate registry with placeholder handlers for all 89 operations
	startServers(ctx, pkgs)       // start SSH (port 2200) and web (port 8055) in background goroutines

	if err := runOrDispatch(ctx, pkgs.dispatcher, *menuFlag); err != nil { // run interactive menu or direct dispatch
		slog.Error("menu exited with error", "error", err)               // log abnormal exits for operator visibility
		shutdown(pkgs)                                                    // attempt graceful shutdown before dying
		os.Exit(1)                                                        // non-zero so container orchestration knows something went wrong
	}

	shutdown(pkgs) // graceful shutdown on clean exit: SSH drain (30 s) → web (5 s) → done
}

// initPackages loads the .env file, validates configuration, and constructs all five packages.
// formatFlag is the --format CLI value (empty string means use env var or default "csv").
func initPackages(formatFlag string) (appPackages, error) {
	if err := os.MkdirAll("data", 0750); err != nil {                                     // ensure data/ exists before any file writes (host key, CSV, SQLite)
		return appPackages{}, fmt.Errorf("create data directory: %w", err)              // fail early if the filesystem is read-only
	}
	if err := godotenv.Load(); err != nil {                                               // .env is optional when running in a container with injected env vars
		slog.Debug("no .env file found, relying on environment variables", "error", err) // not fatal; note for debugging misconfigured deployments
	}
	cfg, err := api.LoadConfig(formatFlag) // validate required env vars (MIST_API_TOKEN, MIST_ORG_ID)
	if err != nil {
		return appPackages{}, fmt.Errorf("load config: %w", err) // message includes which env var is missing
	}
	client, err := api.NewClient(cfg) // initialise the mistapi-go SDK client; no network call at this stage
	if err != nil {
		return appPackages{}, fmt.Errorf("create API client: %w", err) // only fails if APIToken is empty post-validation
	}
	writer, err := output.NewWriter(cfg, "data") // create CSV or SQLite writer; files are created on first Write()
	if err != nil {
		return appPackages{}, fmt.Errorf("create output writer: %w", err) // fails if format is invalid or data/ is unwritable
	}
	registry := menu.NewRegistry()                                                // empty registry; stubs are registered in main after initPackages
	dispatcher := menu.NewDispatcher(registry, bufio.NewReader(os.Stdin), writer) // wire stdin into the menu loop
	signer, err := mssh.LoadOrCreateHostKey("data")                               // generate RSA host key on first boot; reload on subsequent starts
	if err != nil {
		return appPackages{}, fmt.Errorf("load SSH host key: %w", err) // fails if data/ is unwritable or key is corrupted
	}
	sshSrv := mssh.NewServer(cfg, signer, registry, writer) // SSH server uses same registry as interactive menu
	webSrv := web.NewServer(cfg)                             // HTTP server configured but not yet listening
	slog.Info("packages initialised", "format", cfg.OutputFormat, "ssh_port", cfg.SSHPort, "web_port", cfg.WebPort)
	return appPackages{cfg: cfg, client: client, writer: writer, registry: registry, dispatcher: dispatcher, sshServer: sshSrv, webServer: webSrv}, nil
}

// registerStubs registers placeholder handlers for all 89 menu operations.
// Each stub logs the invocation and prints a user-facing "not yet implemented" message.
func registerStubs(r *menu.Registry) {
	slog.Info("registering stub menu handlers", "count", len(stubOps)) // log count so we can confirm all 89 are loaded
	for _, op := range stubOps {                                        // range over the package-level stub table
		r.Register(menu.Entry{                                         // register each operation in the shared registry
			Number:   op.n,                           // integer option number the user types at the menu prompt
			Title:    op.name,                        // human-readable name shown in the menu display
			Category: op.cat,                         // category header used to group related operations visually
			Handler:  makeStubHandler(op.n, op.name), // closure captures n and name for the log and print message
		})
	}
	slog.Debug("stub registration complete", "count", len(stubOps)) // confirm after loop completes
}

// makeStubHandler returns a HandlerFunc that prints "not yet implemented" for operation n.
// All stubs return nil so the interactive menu loop continues after showing the message.
func makeStubHandler(n int, name string) menu.HandlerFunc {
	return func(ctx context.Context, reader *bufio.Reader, w output.Writer) error { // closure captures n and name from the outer scope
		slog.Info("stub: operation not yet implemented", "operation", n, "name", name) // log so audit trail shows which stub was invoked
		fmt.Printf("  Operation %d (%s) is not yet implemented.\n  Port from MistHelper.py first.\n\n", n, name) // clear user-facing message directing the porter to the Python reference
		return nil // nil keeps the interactive menu loop running after this stub exits
	}
}

// startServers launches the SSH and web servers in background goroutines.
// Both servers run until their context is cancelled or Shutdown is called.
func startServers(ctx context.Context, pkgs appPackages) {
	slog.Info("starting SSH server", "port", pkgs.cfg.SSHPort) // log before goroutine starts (not from inside the goroutine)
	go func() {
		if err := pkgs.sshServer.ListenAndServe(ctx); err != nil { // blocks until ctx.Done() or a fatal bind error
			slog.Error("SSH server exited unexpectedly", "error", err) // log unexpected exits for operator visibility
		}
	}()
	slog.Info("starting web server", "port", pkgs.cfg.WebPort) // log before goroutine starts
	go func() {
		if err := pkgs.webServer.ListenAndServe(); err != nil { // blocks until Shutdown is called or a bind error occurs
			slog.Error("web server exited unexpectedly", "error", err) // log bind failures (e.g. port already in use)
		}
	}()
	slog.Debug("background servers launched") // confirm both goroutines have been started
}

// runOrDispatch runs the interactive menu (menuNum < 0), exits cleanly (menuNum == 0),
// or dispatches a single operation by number (menuNum > 0) for automation.
func runOrDispatch(ctx context.Context, d *menu.Dispatcher, menuNum int) error {
	if menuNum < 0 { // default: no --menu flag given, enter the interactive loop
		slog.Info("starting interactive menu") // log so the operator knows which mode is active
		return d.Run(ctx)                      // block in interactive loop until EOF, option 0, or context cancel
	}
	if menuNum == 0 { // --menu 0 is the explicit quit / smoke-test invocation
		slog.Info("menu option 0: exiting cleanly") // log so container logs show this was intentional
		return nil                                  // nil causes main to proceed to shutdown and exit 0
	}
	slog.Info("direct menu dispatch", "option", menuNum) // log before dispatch so the invocation is traceable
	return d.Dispatch(ctx, menuNum)                      // run a single operation then return
}

// shutdown drains active SSH sessions (30 s max) then stops the HTTP server (5 s max).
// Called on both clean and error exit paths to ensure ports are always released.
func shutdown(pkgs appPackages) {
	slog.Info("beginning graceful shutdown sequence")                                                    // log entry so operators know shutdown is in progress
	sshCtx, sshCancel := context.WithTimeout(context.Background(), 30*time.Second)                      // 30 s for active SSH sessions to finish their current operation
	defer sshCancel()                                                                                    // release timeout resources even if Shutdown returns early
	if err := pkgs.sshServer.Shutdown(sshCtx); err != nil {                                            // wait for active SSH sessions to complete
		slog.Error("SSH server shutdown error", "error", err)                                          // log but continue to web shutdown
	}
	slog.Debug("SSH server drained")                                                                    // log after SSH is clear
	webCtx, webCancel := context.WithTimeout(context.Background(), 5*time.Second)                       // 5 s for in-flight HTTP requests to complete
	defer webCancel()                                                                                    // release timeout resources
	if err := pkgs.webServer.Shutdown(webCtx); err != nil {                                            // gracefully stop the HTTP server
		slog.Error("web server shutdown error", "error", err)                                          // log but continue — ports are freed at process exit
	}
	slog.Info("graceful shutdown complete")                                                             // log so container orchestration can see a clean teardown
}

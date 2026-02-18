package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"encoding/json"

	interactionstore "github.com/Meesho/BharatMLStack/go-sdk/pkg/interaction-store"
	"github.com/Meesho/BharatMLStack/go-sdk/pkg/onfs"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type CLIConfig struct {
	Host      string
	Port      string
	PlainText bool
	Timeout   int
	BatchSize int
	Mode      string
	InputFile string
	// Interaction Store specific config
	ISHost string
	ISPort string
}

// ServiceType indicates which service to use
type ServiceType int

const (
	ServiceONFS ServiceType = iota
	ServiceInteractionStore
)

type Session struct {
	CallerID       string
	CallerToken    string
	Client         onfs.Client
	ISClient       interactionstore.Client
	IsONFSLoggedIn bool
	IsISLoggedIn   bool
	AppName        string
	ActiveService  ServiceType
}

// EtcdConfig holds etcd connection configuration
type EtcdConfig struct {
	Endpoints []string
	Username  string
	Password  string
	Timeout   time.Duration
}

// FeatureSchemaConfig represents the feature schema structure from etcd
type FeatureSchemaConfig struct {
	Labels      string                 `json:"labels"`
	FeatureMeta map[string]FeatureMeta `json:"feature-meta"`
}

type FeatureMeta struct {
	Sequence             int    `json:"sequence"`
	DefaultValuesInBytes []byte `json:"default-value"`
	StringLength         uint16 `json:"string-length"`
	VectorLength         uint16 `json:"vector-length"`
}

type FeatureGroupConfig struct {
	Id            int                            `json:"id"`
	ActiveVersion string                         `json:"active-version"`
	Features      map[string]FeatureSchemaConfig `json:"features"`
	DataType      string                         `json:"data-type"`
}

type EntityConfig struct {
	Label         string                        `json:"label"`
	Keys          map[string]KeyConfig          `json:"keys"`
	FeatureGroups map[string]FeatureGroupConfig `json:"feature-groups"`
}

type KeyConfig struct {
	Sequence    int    `json:"sequence"`
	EntityLabel string `json:"entity-label"`
	ColumnLabel string `json:"column-label"`
}

type FeatureRegistry struct {
	Entities map[string]EntityConfig `json:"entities"`
}

// EtcdSchemaResolver handles schema resolution from etcd
type EtcdSchemaResolver struct {
	client  *clientv3.Client
	appName string
}

func NewEtcdSchemaResolver(config *EtcdConfig, appName string) (*EtcdSchemaResolver, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   config.Endpoints,
		Username:    config.Username,
		Password:    config.Password,
		DialTimeout: config.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return &EtcdSchemaResolver{
		client:  client,
		appName: appName,
	}, nil
}

func (e *EtcdSchemaResolver) Close() error {
	return e.client.Close()
}

func (e *EtcdSchemaResolver) GetFeatureSchema(entityLabel, fgLabel string) (*FeatureSchemaConfig, error) {
	// Query the specific feature group active version
	activeVersionPath := fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/active-version", e.appName, entityLabel, fgLabel)

	fmt.Printf("🔍 Querying active version at: %s\n", activeVersionPath)

	resp, err := e.client.Get(context.Background(), activeVersionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get active version from etcd: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("feature group %s not found for entity %s", fgLabel, entityLabel)
	}

	activeVersion := string(resp.Kvs[0].Value)
	fmt.Printf("📌 Active version: %s\n", activeVersion)

	// Get feature labels for the active version
	labelsPath := fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/features/%s/labels", e.appName, entityLabel, fgLabel, activeVersion)

	fmt.Printf("🔍 Querying feature labels at: %s\n", labelsPath)

	labelsResp, err := e.client.Get(context.Background(), labelsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature labels from etcd: %w", err)
	}

	if len(labelsResp.Kvs) == 0 {
		return nil, fmt.Errorf("feature labels not found for version %s", activeVersion)
	}

	labels := string(labelsResp.Kvs[0].Value)
	fmt.Printf("📝 Feature labels: %s\n", labels)

	// Get feature metadata for the active version
	metaPath := fmt.Sprintf("/config/%s/entities/%s/feature-groups/%s/features/%s/feature-meta", e.appName, entityLabel, fgLabel, activeVersion)

	fmt.Printf("🔍 Querying feature metadata at: %s\n", metaPath)

	metaResp, err := e.client.Get(context.Background(), metaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature metadata from etcd: %w", err)
	}

	if len(metaResp.Kvs) == 0 {
		return nil, fmt.Errorf("feature metadata not found for version %s", activeVersion)
	}

	var featureMeta map[string]FeatureMeta
	if err := json.Unmarshal(metaResp.Kvs[0].Value, &featureMeta); err != nil {
		return nil, fmt.Errorf("failed to parse feature metadata: %w", err)
	}

	fmt.Printf("✅ Loaded feature metadata with %d features\n", len(featureMeta))

	return &FeatureSchemaConfig{
		Labels:      labels,
		FeatureMeta: featureMeta,
	}, nil
}

func (e *EtcdSchemaResolver) GetKeySchema(entityLabel string) ([]string, error) {
	// Query all keys for the entity using prefix search
	keysPath := fmt.Sprintf("/config/%s/entities/%s/keys/", e.appName, entityLabel)

	fmt.Printf("🔍 Querying entity keys at: %s\n", keysPath)

	resp, err := e.client.Get(context.Background(), keysPath, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get entity keys from etcd: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("no keys found for entity %s", entityLabel)
	}

	// Parse key configurations from individual etcd keys
	keyConfigs := make(map[int]KeyConfig)

	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		value := string(kv.Value)

		// Extract key index and property from path like "/config/onfs/entities/user/keys/0/column-label"
		pathParts := strings.Split(key, "/")
		if len(pathParts) >= 7 {
			keyIndexStr := pathParts[6] // The "0" part
			property := pathParts[7]    // The "column-label", "sequence", or "entity-label" part

			keyIndex, err := strconv.Atoi(keyIndexStr)
			if err != nil {
				fmt.Printf("⚠️  Warning: Invalid key index %s\n", keyIndexStr)
				continue
			}

			// Initialize key config if not exists
			if _, exists := keyConfigs[keyIndex]; !exists {
				keyConfigs[keyIndex] = KeyConfig{}
			}

			keyConfig := keyConfigs[keyIndex]

			switch property {
			case "column-label":
				keyConfig.ColumnLabel = value
			case "sequence":
				if seq, err := strconv.Atoi(value); err == nil {
					keyConfig.Sequence = seq
				}
			case "entity-label":
				keyConfig.EntityLabel = value
			}

			keyConfigs[keyIndex] = keyConfig
			fmt.Printf("🔑 Updated key config %d: %+v\n", keyIndex, keyConfig)
		}
	}

	if len(keyConfigs) == 0 {
		return nil, fmt.Errorf("no valid key configurations found for entity %s", entityLabel)
	}

	// Build ordered key schema based on sequence
	maxSequence := -1
	for _, keyConfig := range keyConfigs {
		if keyConfig.Sequence > maxSequence {
			maxSequence = keyConfig.Sequence
		}
	}

	keySchema := make([]string, maxSequence+1)
	for _, keyConfig := range keyConfigs {
		if keyConfig.Sequence >= 0 && keyConfig.Sequence < len(keySchema) {
			keySchema[keyConfig.Sequence] = keyConfig.ColumnLabel
		}
	}

	// Filter out empty slots
	var filteredKeySchema []string
	for _, col := range keySchema {
		if col != "" {
			filteredKeySchema = append(filteredKeySchema, col)
		}
	}

	fmt.Printf("🎯 Entity %s key schema: %v\n", entityLabel, filteredKeySchema)
	return filteredKeySchema, nil
}

func main() {
	config := parseCLIArgs()

	fmt.Println("🚀 BharatMLStack CLI Tool - SQL-like Interface")
	fmt.Println("Supports: Online Feature Store (ONFS) & Interaction Store (IS)")
	fmt.Println("Type 'help' for available commands or 'exit' to quit")

	session := &Session{
		ActiveService: ServiceONFS,
	}

	if config.Mode == "interactive" {
		runInteractiveMode(config, session)
	} else if config.Mode == "file" && config.InputFile != "" {
		runFileMode(config, session)
	} else {
		runInteractiveMode(config, session)
	}
}

func parseCLIArgs() *CLIConfig {
	config := &CLIConfig{}

	flag.StringVar(&config.Host, "host", "localhost", "ONFS service host")
	flag.StringVar(&config.Port, "port", "8089", "ONFS service port")
	flag.StringVar(&config.ISHost, "is-host", "localhost", "Interaction Store service host")
	flag.StringVar(&config.ISPort, "is-port", "9700", "Interaction Store service port")
	flag.BoolVar(&config.PlainText, "plaintext", true, "Use plaintext connection (no TLS)")
	flag.IntVar(&config.Timeout, "timeout", 30000, "Request timeout in milliseconds")
	flag.IntVar(&config.BatchSize, "batch-size", 50, "Batch size for requests")
	flag.StringVar(&config.Mode, "mode", "interactive", "Mode: interactive or file")
	flag.StringVar(&config.InputFile, "input", "", "Input file with SQL commands")

	flag.Parse()

	return config
}

func runInteractiveMode(config *CLIConfig, session *Session) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		prompt := getPrompt(session)
		fmt.Print(prompt)

		if !scanner.Scan() {
			break
		}

		command := strings.TrimSpace(scanner.Text())
		if command == "" {
			continue
		}

		if command == "exit" || command == "quit" {
			fmt.Println("👋 Goodbye!")
			break
		}

		executeCommand(config, session, command)
	}
}

func getPrompt(session *Session) string {
	var service string
	var loggedIn bool
	var callerID string

	if session.ActiveService == ServiceInteractionStore {
		service = "is"
		loggedIn = session.IsISLoggedIn
		callerID = session.CallerID
	} else {
		service = "onfs"
		loggedIn = session.IsONFSLoggedIn
		callerID = session.CallerID
	}

	if loggedIn {
		return fmt.Sprintf("%s[%s]> ", service, callerID)
	}
	return fmt.Sprintf("%s> ", service)
}

func runFileMode(config *CLIConfig, session *Session) {
	data, err := os.ReadFile(config.InputFile)
	if err != nil {
		fmt.Printf("❌ Error reading file: %v\n", err)
		return
	}

	commands := strings.Split(string(data), "\n")
	for _, command := range commands {
		command = strings.TrimSpace(command)
		if command == "" || strings.HasPrefix(command, "--") {
			continue
		}

		fmt.Printf("onfs> %s\n", command)
		executeCommand(config, session, command)
	}
}

func executeCommand(config *CLIConfig, session *Session, command string) {
	command = strings.TrimSpace(command)
	lowerCmd := strings.ToLower(command)

	switch {
	case command == "help":
		showHelp()
	// Service switching commands
	case lowerCmd == "use onfs":
		session.ActiveService = ServiceONFS
		fmt.Println("🔄 Switched to Online Feature Store (ONFS)")
	case lowerCmd == "use is" || lowerCmd == "use interaction-store":
		session.ActiveService = ServiceInteractionStore
		fmt.Println("🔄 Switched to Interaction Store")
	// ONFS commands
	case strings.HasPrefix(lowerCmd, "login") && session.ActiveService == ServiceONFS:
		handleLogin(config, session, command)
	case strings.HasPrefix(lowerCmd, "insert") && session.ActiveService == ServiceONFS:
		handleInsert(session, command)
	case strings.HasPrefix(lowerCmd, "select_decoded") && session.ActiveService == ServiceONFS:
		handleSelect(session, command, true)
	case strings.HasPrefix(lowerCmd, "select") && session.ActiveService == ServiceONFS:
		handleSelect(session, command, false)
	// Interaction Store commands
	case strings.HasPrefix(lowerCmd, "login") && session.ActiveService == ServiceInteractionStore:
		handleISLogin(config, session, command)
	case strings.HasPrefix(lowerCmd, "persist_click"):
		handleISPersistClick(session, command)
	case strings.HasPrefix(lowerCmd, "persist_order"):
		handleISPersistOrder(session, command)
	case strings.HasPrefix(lowerCmd, "retrieve_clicks"):
		handleISRetrieveClicks(session, command)
	case strings.HasPrefix(lowerCmd, "retrieve_orders"):
		handleISRetrieveOrders(session, command)
	case strings.HasPrefix(lowerCmd, "retrieve_interactions"):
		handleISRetrieveInteractions(session, command)
	// Common commands
	case command == "status":
		showStatus(session)
	case command == "logout":
		handleLogout(session)
	default:
		fmt.Printf("❌ Unknown command: %s\nType 'help' for available commands.\n", command)
	}
}

func showHelp() {
	fmt.Println("\n📖 BharatMLStack CLI Commands:")
	fmt.Println()
	fmt.Println("🔀 Service Selection:")
	fmt.Println("  use onfs                                     - Switch to Online Feature Store")
	fmt.Println("  use is                                       - Switch to Interaction Store")
	fmt.Println()
	fmt.Println("🔐 Session Management:")
	fmt.Println("  login <app_name> <caller_id> <caller_token>  - Login with credentials and app context")
	fmt.Println("  logout                                       - Logout from current session")
	fmt.Println("  status                                       - Show current session status")
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("📦 ONLINE FEATURE STORE (ONFS) Commands:")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("  INSERT Syntax:")
	fmt.Println("    insert into <entity>.<fg> (<features>) values (<values>)")
	fmt.Println("    insert into <entity>.<fg1,fg2> (<features>) values (<values>)")
	fmt.Println()
	fmt.Println("  SELECT Syntax:")
	fmt.Println("    select <features> from <entity>.<fg> where <key_schema>=<key_values>")
	fmt.Println("    select_decoded <features> from <entity>.<fg> where <key_schema>=<key_values>")
	fmt.Println()
	fmt.Println("  Examples:")
	fmt.Println("    use onfs")
	fmt.Println("    login onfs onfs-cli test")
	fmt.Println("    insert into user.profile (age,location) values (25,'NYC')")
	fmt.Println("    select age,location from user.profile where user_id=123")
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("🔄 INTERACTION STORE Commands:")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("  PERSIST Commands:")
	fmt.Println("    persist_click <user_id> <catalog_id> <product_id> <timestamp> [metadata]")
	fmt.Println("    persist_order <user_id> <catalog_id> <product_id> <sub_order_num> <timestamp> [metadata]")
	fmt.Println()
	fmt.Println("  RETRIEVE Commands:")
	fmt.Println("    retrieve_clicks <user_id> <start_ts> <end_ts> <limit>")
	fmt.Println("    retrieve_orders <user_id> <start_ts> <end_ts> <limit>")
	fmt.Println("    retrieve_interactions <user_id> <types> <start_ts> <end_ts> <limit>")
	fmt.Println("      (types: click,order or click or order)")
	fmt.Println()
	fmt.Println("  Examples:")
	fmt.Println("    use is")
	fmt.Println("    login is-app is-caller")
	fmt.Println("    persist_click user123 100 200 1704067200000")
	fmt.Println("    persist_click user123 100 200 1704067200000 '{\"source\":\"homepage\"}'")
	fmt.Println("    persist_order user123 100 200 SUB001 1704067200000")
	fmt.Println("    persist_order user123 100 200 SUB001 1704067200000 '{\"payment\":\"upi\"}'")
	fmt.Println("    retrieve_clicks user123 1704067200000 1704153600000 100")
	fmt.Println("    retrieve_orders user123 1704067200000 1704153600000 100")
	fmt.Println("    retrieve_interactions user123 click,order 1704067200000 1704153600000 100")
	fmt.Println()
	fmt.Println("💡 Data Types:")
	fmt.Println("  Strings: 'value' or \"value\"")
	fmt.Println("  Numbers: 123, 45.67")
	fmt.Println("  Booleans: true, false")
	fmt.Println("  Arrays: [1,2,3] or ['a','b','c']")
	fmt.Println("  Timestamps: Unix milliseconds (e.g., 1704067200000)")
	fmt.Println("  Metadata: JSON string (e.g., '{\"source\":\"homepage\"}')")
	fmt.Println()
	fmt.Println("🚪 Other:")
	fmt.Println("  help                              - Show this help")
	fmt.Println("  exit                              - Exit the CLI")
}

func handleLogin(config *CLIConfig, session *Session, command string) {
	parts := strings.Fields(command)
	if len(parts) != 4 {
		fmt.Println("❌ Usage: login <app_name> <caller_id> <caller_token>")
		return
	}

	appName := parts[1]
	callerID := parts[2]
	callerToken := parts[3]

	fmt.Printf("🔐 Logging in as %s for app %s...\n", callerID, appName)

	// First authenticate with etcd
	etcdConfig := &EtcdConfig{
		Endpoints: []string{"localhost:2379"}, // Default etcd endpoint
		Timeout:   5 * time.Second,
	}

	// Try to get etcd config from environment
	if etcdEndpoints := os.Getenv("ETCD_SERVER"); etcdEndpoints != "" {
		etcdConfig.Endpoints = strings.Split(etcdEndpoints, ",")
	}
	if etcdUsername := os.Getenv("ETCD_USERNAME"); etcdUsername != "" {
		etcdConfig.Username = etcdUsername
	}
	if etcdPassword := os.Getenv("ETCD_PASSWORD"); etcdPassword != "" {
		etcdConfig.Password = etcdPassword
	}

	// Authenticate against etcd
	authenticated, err := authenticateWithEtcd(etcdConfig, appName, callerID, callerToken)
	if err != nil {
		fmt.Printf("❌ Authentication failed: %v\n", err)
		return
	}

	if !authenticated {
		fmt.Printf("❌ Invalid credentials for caller %s in app %s\n", callerID, appName)
		return
	}

	fmt.Printf("✅ etcd authentication successful for %s\n", callerID)

	// Log entities access
	if err := logEtcdEntitiesAccess(etcdConfig, appName, callerID); err != nil {
		fmt.Printf("⚠️  Warning: Could not log entities access: %v\n", err)
	}

	// Create ONFS client
	onfsConfig := &onfs.Config{
		Host:        config.Host,
		Port:        config.Port,
		DeadLine:    config.Timeout,
		PlainText:   config.PlainText,
		BatchSize:   config.BatchSize,
		CallerId:    callerID,
		CallerToken: callerToken,
	}

	timing := func(name string, value time.Duration, tags []string) {
		fmt.Printf("📊 [METRIC] %s: %v\n", name, value)
	}

	count := func(name string, value int64, tags []string) {
		fmt.Printf("📊 [METRIC] %s: %d\n", name, value)
	}

	client := onfs.InitClient(onfs.Version1, onfsConfig, timing, count)

	session.CallerID = callerID
	session.CallerToken = callerToken
	session.Client = client
	session.IsONFSLoggedIn = true
	session.AppName = appName

	fmt.Printf("✅ Successfully logged in as %s for app %s\n", callerID, appName)
}

// authenticateWithEtcd validates caller credentials against etcd
func authenticateWithEtcd(config *EtcdConfig, appName, callerID, callerToken string) (bool, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   config.Endpoints,
		Username:    config.Username,
		Password:    config.Password,
		DialTimeout: config.Timeout,
	})
	if err != nil {
		return false, fmt.Errorf("failed to create etcd client: %w", err)
	}
	defer client.Close()

	// Check authentication path: /config/<APP_NAME>/security/reader/<callerId>
	authPath := fmt.Sprintf("/config/%s/security/reader/%s", appName, callerID)

	fmt.Printf("🔍 Checking etcd authentication at: %s\n", authPath)

	resp, err := client.Get(context.Background(), authPath)
	if err != nil {
		return false, fmt.Errorf("failed to query etcd: %w", err)
	}

	if len(resp.Kvs) == 0 {
		fmt.Printf("❌ No authentication entry found at %s\n", authPath)
		return false, nil
	}

	// Parse the stored token
	var authData struct {
		Token string `json:"token"`
	}

	if err := json.Unmarshal(resp.Kvs[0].Value, &authData); err != nil {
		return false, fmt.Errorf("failed to parse authentication data: %w", err)
	}

	fmt.Printf("🔐 Found stored token for %s\n", callerID)

	// Validate token
	if authData.Token != callerToken {
		fmt.Printf("❌ Token mismatch for caller %s\n", callerID)
		return false, nil
	}

	fmt.Printf("✅ Token validation successful for %s\n", callerID)
	return true, nil
}

// logEtcdEntitiesAccess logs access to entities configuration
func logEtcdEntitiesAccess(config *EtcdConfig, appName, callerID string) error {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   config.Endpoints,
		Username:    config.Username,
		Password:    config.Password,
		DialTimeout: config.Timeout,
	})
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}
	defer client.Close()

	// Query entities path: /config/<APP_NAME>/entities
	entitiesPath := fmt.Sprintf("/config/%s/entities", appName)

	fmt.Printf("📋 Logging etcd entities access at: %s\n", entitiesPath)

	resp, err := client.Get(context.Background(), entitiesPath, clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("failed to query entities: %w", err)
	}

	fmt.Printf("📊 Found %d entity configuration entries\n", len(resp.Kvs))

	// Log entity names for debugging
	entityNames := make([]string, 0)
	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		// Extract entity name from path
		parts := strings.Split(key, "/")
		if len(parts) > 4 {
			entityName := parts[4] // /config/app/entities/[entityName]/...
			if entityName != "" && !contains(entityNames, entityName) {
				entityNames = append(entityNames, entityName)
			}
		}
	}

	if len(entityNames) > 0 {
		fmt.Printf("🎯 Available entities for %s: %v\n", callerID, entityNames)
	} else {
		fmt.Printf("⚠️  No entities found in configuration for app %s\n", appName)
	}

	return nil
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func handleLogout(session *Session) {
	if session.ActiveService == ServiceInteractionStore {
		if !session.IsISLoggedIn {
			fmt.Println("❌ Not logged in to Interaction Store")
			return
		}
		fmt.Printf("👋 Logging out %s from Interaction Store\n", session.CallerID)
		session.IsISLoggedIn = false
		session.ISClient = nil
	} else {
		if !session.IsONFSLoggedIn {
			fmt.Println("❌ Not logged in to ONFS")
			return
		}
		fmt.Printf("👋 Logging out %s from app %s\n", session.CallerID, session.AppName)
		session.IsONFSLoggedIn = false
		session.Client = nil
	}

	// Clear shared fields if both are logged out
	if !session.IsONFSLoggedIn && !session.IsISLoggedIn {
		session.CallerID = ""
		session.CallerToken = ""
		session.AppName = ""
	}
}

func showStatus(session *Session) {
	fmt.Println("\n📊 Session Status:")

	// Show active service
	if session.ActiveService == ServiceInteractionStore {
		fmt.Println("  Active Service: 🔄 Interaction Store")
	} else {
		fmt.Println("  Active Service: 📦 Online Feature Store (ONFS)")
	}

	fmt.Println()

	// ONFS Status
	fmt.Println("  📦 ONFS:")
	if session.IsONFSLoggedIn {
		fmt.Printf("    Status: ✅ Logged in\n")
		fmt.Printf("    App Name: %s\n", session.AppName)
		fmt.Printf("    Caller ID: %s\n", session.CallerID)
	} else {
		fmt.Printf("    Status: ❌ Not logged in\n")
	}

	fmt.Println()

	// Interaction Store Status
	fmt.Println("  🔄 Interaction Store:")
	if session.IsISLoggedIn {
		fmt.Printf("    Status: ✅ Logged in\n")
		fmt.Printf("    Caller ID: %s\n", session.CallerID)
	} else {
		fmt.Printf("    Status: ❌ Not logged in\n")
	}

	fmt.Println()
	fmt.Println("  Use 'use onfs' or 'use is' to switch services")
	fmt.Println("  Use 'login <app_name> <caller_id> <caller_token>' to authenticate")
}

func handleInsert(session *Session, command string) {
	if !session.IsONFSLoggedIn {
		fmt.Println("❌ Please login first using: login <app_name> <caller_id> <caller_token>")
		return
	}

	// Initialize etcd resolver using session context
	etcdConfig := &EtcdConfig{
		Endpoints: []string{"localhost:2379"}, // Default etcd endpoint
		Timeout:   5 * time.Second,
	}

	// Try to get etcd config from environment
	if etcdEndpoints := os.Getenv("ETCD_SERVER"); etcdEndpoints != "" {
		etcdConfig.Endpoints = strings.Split(etcdEndpoints, ",")
	}
	if etcdUsername := os.Getenv("ETCD_USERNAME"); etcdUsername != "" {
		etcdConfig.Username = etcdUsername
	}
	if etcdPassword := os.Getenv("ETCD_PASSWORD"); etcdPassword != "" {
		etcdConfig.Password = etcdPassword
	}

	// Use the session's app name instead of environment variable
	appName := session.AppName
	if appName == "" {
		fmt.Println("❌ No app context found. Please login again with app name.")
		return
	}

	resolver, err := NewEtcdSchemaResolver(etcdConfig, appName)
	if err != nil {
		fmt.Printf("❌ Failed to connect to etcd: %v\n", err)
		fmt.Println("💡 Falling back to type inference...")
		handleInsertWithTypeInference(session, command)
		return
	}
	defer resolver.Close()

	insertData, err := parseInsertWithSchema(command, resolver)
	if err != nil {
		fmt.Printf("❌ Error parsing INSERT with schema: %v\n", err)
		return
	}

	// Convert to ONFS persist request
	persistRequest := &onfs.PersistFeaturesRequest{
		EntityLabel:   insertData.Entity,
		KeysSchema:    insertData.KeysSchema,
		FeatureGroups: insertData.FeatureGroups,
		Data:          insertData.Data,
	}

	fmt.Printf("📝 Persisting features for entity: %s (schema-validated for app: %s)\n", insertData.Entity, appName)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := session.Client.PersistFeatures(ctx, persistRequest)
	if err != nil {
		fmt.Printf("❌ Error persisting features: %v\n", err)
		return
	}

	fmt.Printf("✅ Insert successful: %s\n", response.Message)
}

func parseInsertWithSchema(command string, resolver *EtcdSchemaResolver) (*InsertData, error) {
	// Parse: insert into entity.fg1,fg2 (features) values (values)
	re := regexp.MustCompile(`(?i)insert\s+into\s+([^.\s]+)\.([^.\s\(]+)\s*\(([^)]+)\)\s+values\s*\(([^)]+)\)`)
	matches := re.FindStringSubmatch(command)

	if len(matches) != 5 {
		return nil, fmt.Errorf("invalid INSERT syntax. Use: insert into entity.fg (features) values (values)")
	}

	entity := matches[1]
	fgString := matches[2]
	featuresString := matches[3]
	valuesString := matches[4]

	// Parse feature groups (support comma-separated: fg1,fg2)
	fgNames := strings.Split(fgString, ",")
	for i := range fgNames {
		fgNames[i] = strings.TrimSpace(fgNames[i])
	}

	// Parse features
	features := strings.Split(featuresString, ",")
	for i := range features {
		features[i] = strings.TrimSpace(features[i])
	}

	// Parse values
	values, err := parseValues(valuesString)
	if err != nil {
		return nil, fmt.Errorf("error parsing values: %v", err)
	}

	// Get key schema from etcd
	keySchema, err := resolver.GetKeySchema(entity)
	if err != nil {
		return nil, fmt.Errorf("error getting key schema: %v", err)
	}

	// Create feature groups with schema validation
	var featureGroups []onfs.FeatureGroupSchema
	featuresPerGroup := len(features) / len(fgNames)

	for i, fgName := range fgNames {
		start := i * featuresPerGroup
		end := start + featuresPerGroup
		if i == len(fgNames)-1 {
			end = len(features) // Last group gets remaining features
		}

		groupFeatures := features[start:end]

		// Get schema for this feature group
		schema, err := resolver.GetFeatureSchema(entity, fgName)
		if err != nil {
			return nil, fmt.Errorf("error getting schema for feature group %s: %v", fgName, err)
		}

		// Validate features exist in schema
		schemaFeatures := strings.Split(schema.Labels, ",")
		schemaFeatureSet := make(map[string]bool)
		for _, f := range schemaFeatures {
			schemaFeatureSet[strings.TrimSpace(f)] = true
		}

		for _, feature := range groupFeatures {
			if !schemaFeatureSet[feature] {
				return nil, fmt.Errorf("feature %s not found in schema for feature group %s", feature, fgName)
			}
		}

		featureGroups = append(featureGroups, onfs.FeatureGroupSchema{
			Label:         fgName,
			FeatureLabels: groupFeatures,
		})
	}

	// Assume first value is key, rest are features
	if len(values) < len(keySchema)+len(features) {
		return nil, fmt.Errorf("insufficient values provided. Expected %d key values + %d feature values", len(keySchema), len(features))
	}

	keyValues := values[:len(keySchema)]
	featureValues := values[len(keySchema):]

	// Create ONFS values structure using schema-based type inference
	onfsValues := onfs.Values{}

	// Process each feature group to determine correct types
	featureIndex := 0
	for i, fgName := range fgNames {
		schema, _ := resolver.GetFeatureSchema(entity, fgName)

		start := i * featuresPerGroup
		end := start + featuresPerGroup
		if i == len(fgNames)-1 {
			end = len(features)
		}

		groupFeatures := features[start:end]

		for _, featureName := range groupFeatures {
			if featureIndex >= len(featureValues) {
				break
			}

			val := featureValues[featureIndex]
			featureIndex++

			// Get feature metadata from schema
			if meta, exists := schema.FeatureMeta[featureName]; exists {
				// Use schema metadata to determine correct type
				if meta.StringLength > 0 {
					// String type
					onfsValues.StringValues = append(onfsValues.StringValues, val)
				} else if meta.VectorLength > 0 {
					// Vector type - parse as array
					if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
						vectorStr := strings.Trim(val, "[]")
						vectorParts := strings.Split(vectorStr, ",")
						var vectorValues onfs.Values
						for _, part := range vectorParts {
							part = strings.TrimSpace(part)
							if f, err := strconv.ParseFloat(part, 32); err == nil {
								vectorValues.Fp32Values = append(vectorValues.Fp32Values, f)
							}
						}
						onfsValues.Vector = append(onfsValues.Vector, onfs.Vector{Values: vectorValues})
					}
				} else {
					// Numeric or boolean type - infer from value
					if val == "true" || val == "false" {
						onfsValues.BoolValues = append(onfsValues.BoolValues, val == "true")
					} else if isNumeric(val) {
						if strings.Contains(val, ".") {
							if f, err := strconv.ParseFloat(val, 64); err == nil {
								onfsValues.Fp64Values = append(onfsValues.Fp64Values, f)
							}
						} else {
							if i, err := strconv.ParseInt(val, 10, 32); err == nil {
								onfsValues.Int32Values = append(onfsValues.Int32Values, int32(i))
							}
						}
					} else {
						// Default to string
						onfsValues.StringValues = append(onfsValues.StringValues, val)
					}
				}
			} else {
				// Fallback to type inference if no metadata
				if val == "true" || val == "false" {
					onfsValues.BoolValues = append(onfsValues.BoolValues, val == "true")
				} else if isNumeric(val) {
					if strings.Contains(val, ".") {
						if f, err := strconv.ParseFloat(val, 64); err == nil {
							onfsValues.Fp64Values = append(onfsValues.Fp64Values, f)
						}
					} else {
						if i, err := strconv.ParseInt(val, 10, 32); err == nil {
							onfsValues.Int32Values = append(onfsValues.Int32Values, int32(i))
						}
					}
				} else {
					onfsValues.StringValues = append(onfsValues.StringValues, val)
				}
			}
		}
	}

	data := []onfs.Data{{
		KeyValues:     keyValues,
		FeatureValues: []onfs.FeatureValues{{Values: onfsValues}},
	}}

	return &InsertData{
		Entity:        entity,
		FeatureGroups: featureGroups,
		KeysSchema:    keySchema,
		Data:          data,
	}, nil
}

// Fallback function for when etcd is not available
func handleInsertWithTypeInference(session *Session, command string) {
	insertData, err := parseInsert(command)
	if err != nil {
		fmt.Printf("❌ Error parsing INSERT: %v\n", err)
		return
	}

	// Convert to ONFS persist request
	persistRequest := &onfs.PersistFeaturesRequest{
		EntityLabel:   insertData.Entity,
		KeysSchema:    insertData.KeysSchema,
		FeatureGroups: insertData.FeatureGroups,
		Data:          insertData.Data,
	}

	fmt.Printf("📝 Persisting features for entity: %s (type-inferred)\n", insertData.Entity)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := session.Client.PersistFeatures(ctx, persistRequest)
	if err != nil {
		fmt.Printf("❌ Error persisting features: %v\n", err)
		return
	}

	fmt.Printf("✅ Insert successful: %s\n", response.Message)
}

func handleSelect(session *Session, command string, decoded bool) {
	if !session.IsONFSLoggedIn {
		fmt.Println("❌ Please login first using: login <app_name> <caller_id> <caller_token>")
		return
	}

	selectData, err := parseSelect(command)
	if err != nil {
		fmt.Printf("❌ Error parsing SELECT: %v\n", err)
		return
	}

	// Convert to ONFS query
	query := &onfs.Query{
		EntityLabel:   selectData.Entity,
		FeatureGroups: selectData.FeatureGroups,
		KeysSchema:    selectData.KeysSchema,
		Keys:          selectData.Keys,
	}

	fmt.Printf("🔍 Querying features for entity: %s\n", selectData.Entity)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if decoded {
		result, err := session.Client.RetrieveDecodedFeatures(ctx, query)
		if err != nil {
			fmt.Printf("❌ Error retrieving decoded features: %v\n", err)
			return
		}
		printDecodedResult(result)
	} else {
		result, err := session.Client.RetrieveFeatures(ctx, query)
		if err != nil {
			fmt.Printf("❌ Error retrieving features: %v\n", err)
			return
		}
		printResult(result)
	}
}

type InsertData struct {
	Entity        string
	FeatureGroups []onfs.FeatureGroupSchema
	KeysSchema    []string
	Data          []onfs.Data
}

type SelectData struct {
	Entity        string
	FeatureGroups []onfs.FeatureGroup
	KeysSchema    []string
	Keys          []onfs.Keys
}

func parseInsert(command string) (*InsertData, error) {
	// Parse: insert into entity.fg1,fg2 (features) values (values)
	re := regexp.MustCompile(`(?i)insert\s+into\s+([^.\s]+)\.([^.\s\(]+)\s*\(([^)]+)\)\s+values\s*\(([^)]+)\)`)
	matches := re.FindStringSubmatch(command)

	if len(matches) != 5 {
		return nil, fmt.Errorf("invalid INSERT syntax. Use: insert into entity.fg (features) values (values)")
	}

	entity := matches[1]
	fgString := matches[2]
	featuresString := matches[3]
	valuesString := matches[4]

	// Parse feature groups (support comma-separated: fg1,fg2)
	fgNames := strings.Split(fgString, ",")
	for i := range fgNames {
		fgNames[i] = strings.TrimSpace(fgNames[i])
	}

	// Parse features
	features := strings.Split(featuresString, ",")
	for i := range features {
		features[i] = strings.TrimSpace(features[i])
	}

	// Parse values
	values, err := parseValues(valuesString)
	if err != nil {
		return nil, fmt.Errorf("error parsing values: %v", err)
	}

	// Create feature groups
	var featureGroups []onfs.FeatureGroupSchema
	featuresPerGroup := len(features) / len(fgNames)

	for i, fgName := range fgNames {
		start := i * featuresPerGroup
		end := start + featuresPerGroup
		if i == len(fgNames)-1 {
			end = len(features) // Last group gets remaining features
		}

		featureGroups = append(featureGroups, onfs.FeatureGroupSchema{
			Label:         fgName,
			FeatureLabels: features[start:end],
		})
	}

	// Assume first value is key, rest are features
	keyValues := []string{values[0]}
	featureValues := values[1:]

	// Create ONFS values structure
	onfsValues := onfs.Values{}
	for _, val := range featureValues {
		if isNumeric(val) {
			if strings.Contains(val, ".") {
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					onfsValues.Fp64Values = append(onfsValues.Fp64Values, f)
				}
			} else {
				if i, err := strconv.ParseInt(val, 10, 32); err == nil {
					onfsValues.Int32Values = append(onfsValues.Int32Values, int32(i))
				}
			}
		} else if val == "true" || val == "false" {
			onfsValues.BoolValues = append(onfsValues.BoolValues, val == "true")
		} else {
			onfsValues.StringValues = append(onfsValues.StringValues, val)
		}
	}

	data := []onfs.Data{{
		KeyValues:     keyValues,
		FeatureValues: []onfs.FeatureValues{{Values: onfsValues}},
	}}

	return &InsertData{
		Entity:        entity,
		FeatureGroups: featureGroups,
		KeysSchema:    []string{entity + "_id"}, // Default key schema
		Data:          data,
	}, nil
}

func parseSelect(command string) (*SelectData, error) {
	// Parse: select features from entity.fg where key_schema=key_value
	re := regexp.MustCompile(`(?i)^select(?:_decoded)?\s+(.+?)\s+from\s+([^.]+)\.([^\s]+)\s+where\s+([^\s=]+)\s*=\s*(.+)$`)

	matches := re.FindStringSubmatch(command)

	if len(matches) != 6 {
		return nil, fmt.Errorf("invalid SELECT syntax. Use: select features from entity.fg where key_schema=key_value")
	}

	featuresString := strings.TrimSpace(matches[1])
	entity := matches[2]
	fgString := matches[3]
	keySchema := strings.TrimSpace(matches[4])
	keyValue := strings.TrimSpace(matches[5])

	// Remove quotes from key value
	keyValue = strings.Trim(keyValue, "'\"")

	// Parse features
	var features []string
	if featuresString == "*" {
		features = []string{"*"} // Will be handled by server
	} else {
		features = strings.Split(featuresString, ",")
		for i := range features {
			features[i] = strings.TrimSpace(features[i])
		}
	}

	// Parse feature groups
	fgNames := strings.Split(fgString, ",")
	var featureGroups []onfs.FeatureGroup

	for _, fgName := range fgNames {
		featureGroups = append(featureGroups, onfs.FeatureGroup{
			Label:         strings.TrimSpace(fgName),
			FeatureLabels: features,
		})
	}

	return &SelectData{
		Entity:        entity,
		FeatureGroups: featureGroups,
		KeysSchema:    []string{keySchema},
		Keys: []onfs.Keys{{
			Cols: []string{keyValue},
		}},
	}, nil
}

func parseValues(valuesString string) ([]string, error) {
	var values []string
	var current strings.Builder
	var inQuotes bool
	var quoteChar rune

	for _, char := range valuesString {
		switch {
		case char == '\'' || char == '"':
			if !inQuotes {
				inQuotes = true
				quoteChar = char
			} else if char == quoteChar {
				inQuotes = false
			} else {
				current.WriteRune(char)
			}
		case char == ',' && !inQuotes:
			values = append(values, strings.TrimSpace(current.String()))
			current.Reset()
		default:
			current.WriteRune(char)
		}
	}

	// Add the last value
	if current.Len() > 0 {
		values = append(values, strings.TrimSpace(current.String()))
	}

	return values, nil
}

func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func printResult(result *onfs.Result) {
	fmt.Println("\n📊 Query Result:")
	fmt.Printf("Entity: %s\n", result.EntityLabel)
	fmt.Printf("Keys Schema: %v\n", result.KeysSchema)

	if len(result.FeatureSchemas) > 0 {
		fmt.Println("\nFeature Schemas:")
		for _, schema := range result.FeatureSchemas {
			fmt.Printf("  📁 %s: ", schema.FeatureGroupLabel)
			for i, feature := range schema.Features {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("%s", feature.Label)
			}
			fmt.Println()
		}
	}

	fmt.Printf("\n📝 Rows (%d total):\n", len(result.Rows))
	for i, row := range result.Rows {
		fmt.Printf("  Row %d: Keys=%v, Columns=%d bytes\n", i+1, row.Keys, len(row.Columns))
	}
}

func printDecodedResult(result *onfs.DecodedResult) {
	fmt.Println("\n📊 Decoded Query Result:")
	fmt.Printf("Keys Schema: %v\n", result.KeysSchema)

	if len(result.FeatureSchemas) > 0 {
		fmt.Println("\nFeature Schemas:")
		for _, schema := range result.FeatureSchemas {
			fmt.Printf("  📁 %s: ", schema.FeatureGroupLabel)
			for i, feature := range schema.Features {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("%s", feature.Label)
			}
			fmt.Println()
		}
	}

	fmt.Printf("\n📝 Decoded Rows (%d total):\n", len(result.Rows))
	for i, row := range result.Rows {
		fmt.Printf("  Row %d:\n", i+1)
		fmt.Printf("    Keys: %v\n", row.Keys)
		fmt.Printf("    Values: %v\n", row.Columns)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// INTERACTION STORE HANDLERS
// ═══════════════════════════════════════════════════════════════════════════

// handleISLogin handles login for Interaction Store
func handleISLogin(config *CLIConfig, session *Session, command string) {
	parts := strings.Fields(command)
	if len(parts) != 3 {
		fmt.Println("❌ Usage: login <app_name> <caller_id>")
		return
	}

	appName := parts[1]
	callerID := parts[2]

	fmt.Printf("🔐 Logging in to Interaction Store as %s...\n", callerID)

	// Create Interaction Store client
	isConfig := &interactionstore.Config{
		Host:      config.ISHost,
		Port:      config.ISPort,
		DeadLine:  config.Timeout,
		PlainText: config.PlainText,
		CallerId:  callerID,
	}

	timing := func(name string, value time.Duration, tags []string) {
		fmt.Printf("📊 [METRIC] %s: %v\n", name, value)
	}

	count := func(name string, value int64, tags []string) {
		fmt.Printf("📊 [METRIC] %s: %d\n", name, value)
	}

	// Reset registry for re-initialization
	interactionstore.ResetRegistry()

	client := interactionstore.InitClient(interactionstore.Version1, isConfig, timing, count)

	session.CallerID = callerID
	session.ISClient = client
	session.IsISLoggedIn = true
	session.AppName = appName

	fmt.Printf("✅ Successfully logged in to Interaction Store as %s\n", callerID)
}

// handleISPersistClick handles persist_click command
func handleISPersistClick(session *Session, command string) {
	if !session.IsISLoggedIn {
		fmt.Println("❌ Please login to Interaction Store first using: use is && login <app_name> <caller_id>")
		return
	}

	// Parse: persist_click <user_id> <catalog_id> <product_id> <timestamp> [metadata]
	parts := strings.Fields(command)
	if len(parts) < 5 {
		fmt.Println("❌ Usage: persist_click <user_id> <catalog_id> <product_id> <timestamp> [metadata]")
		return
	}

	userID := parts[1]
	catalogID, err := strconv.ParseInt(parts[2], 10, 32)
	if err != nil {
		fmt.Printf("❌ Invalid catalog_id: %v\n", err)
		return
	}
	productID, err := strconv.ParseInt(parts[3], 10, 32)
	if err != nil {
		fmt.Printf("❌ Invalid product_id: %v\n", err)
		return
	}
	timestamp, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		fmt.Printf("❌ Invalid timestamp: %v\n", err)
		return
	}

	metadata := ""
	if len(parts) > 5 {
		metadata = strings.Join(parts[5:], " ")
		metadata = strings.Trim(metadata, "'\"")
	}

	request := &interactionstore.PersistClickDataRequest{
		UserId: userID,
		Data: []interactionstore.ClickData{
			{
				CatalogId: int32(catalogID),
				ProductId: int32(productID),
				Timestamp: timestamp,
				Metadata:  metadata,
			},
		},
	}

	fmt.Printf("📝 Persisting click data for user: %s\n", userID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := session.ISClient.PersistClickData(ctx, request)
	if err != nil {
		fmt.Printf("❌ Error persisting click data: %v\n", err)
		return
	}

	fmt.Printf("✅ Click data persisted: %s\n", response.Message)
}

// handleISPersistOrder handles persist_order command
func handleISPersistOrder(session *Session, command string) {
	if !session.IsISLoggedIn {
		fmt.Println("❌ Please login to Interaction Store first using: use is && login <app_name> <caller_id>")
		return
	}

	// Parse: persist_order <user_id> <catalog_id> <product_id> <sub_order_num> <timestamp> [metadata]
	parts := strings.Fields(command)
	if len(parts) < 6 {
		fmt.Println("❌ Usage: persist_order <user_id> <catalog_id> <product_id> <sub_order_num> <timestamp> [metadata]")
		return
	}

	userID := parts[1]
	catalogID, err := strconv.ParseInt(parts[2], 10, 32)
	if err != nil {
		fmt.Printf("❌ Invalid catalog_id: %v\n", err)
		return
	}
	productID, err := strconv.ParseInt(parts[3], 10, 32)
	if err != nil {
		fmt.Printf("❌ Invalid product_id: %v\n", err)
		return
	}
	subOrderNum := parts[4]
	timestamp, err := strconv.ParseInt(parts[5], 10, 64)
	if err != nil {
		fmt.Printf("❌ Invalid timestamp: %v\n", err)
		return
	}

	metadata := ""
	if len(parts) > 6 {
		metadata = strings.Join(parts[6:], " ")
		metadata = strings.Trim(metadata, "'\"")
	}

	request := &interactionstore.PersistOrderDataRequest{
		UserId: userID,
		Data: []interactionstore.OrderData{
			{
				CatalogId:   int32(catalogID),
				ProductId:   int32(productID),
				SubOrderNum: subOrderNum,
				Timestamp:   timestamp,
				Metadata:    metadata,
			},
		},
	}

	fmt.Printf("📝 Persisting order data for user: %s\n", userID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := session.ISClient.PersistOrderData(ctx, request)
	if err != nil {
		fmt.Printf("❌ Error persisting order data: %v\n", err)
		return
	}

	fmt.Printf("✅ Order data persisted: %s\n", response.Message)
}

// handleISRetrieveClicks handles retrieve_clicks command
func handleISRetrieveClicks(session *Session, command string) {
	if !session.IsISLoggedIn {
		fmt.Println("❌ Please login to Interaction Store first using: use is && login <app_name> <caller_id>")
		return
	}

	// Parse: retrieve_clicks <user_id> <start_ts> <end_ts> <limit>
	parts := strings.Fields(command)
	if len(parts) != 5 {
		fmt.Println("❌ Usage: retrieve_clicks <user_id> <start_ts> <end_ts> <limit>")
		return
	}

	userID := parts[1]
	startTS, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		fmt.Printf("❌ Invalid start_timestamp: %v\n", err)
		return
	}
	endTS, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		fmt.Printf("❌ Invalid end_timestamp: %v\n", err)
		return
	}
	limit, err := strconv.ParseInt(parts[4], 10, 32)
	if err != nil {
		fmt.Printf("❌ Invalid limit: %v\n", err)
		return
	}

	request := &interactionstore.RetrieveDataRequest{
		UserId:         userID,
		StartTimestamp: startTS,
		EndTimestamp:   endTS,
		Limit:          int32(limit),
	}

	fmt.Printf("🔍 Retrieving click interactions for user: %s\n", userID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := session.ISClient.RetrieveClickInteractions(ctx, request)
	if err != nil {
		fmt.Printf("❌ Error retrieving click data: %v\n", err)
		return
	}

	printClickResults(response)
}

// handleISRetrieveOrders handles retrieve_orders command
func handleISRetrieveOrders(session *Session, command string) {
	if !session.IsISLoggedIn {
		fmt.Println("❌ Please login to Interaction Store first using: use is && login <app_name> <caller_id>")
		return
	}

	// Parse: retrieve_orders <user_id> <start_ts> <end_ts> <limit>
	parts := strings.Fields(command)
	if len(parts) != 5 {
		fmt.Println("❌ Usage: retrieve_orders <user_id> <start_ts> <end_ts> <limit>")
		return
	}

	userID := parts[1]
	startTS, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		fmt.Printf("❌ Invalid start_timestamp: %v\n", err)
		return
	}
	endTS, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		fmt.Printf("❌ Invalid end_timestamp: %v\n", err)
		return
	}
	limit, err := strconv.ParseInt(parts[4], 10, 32)
	if err != nil {
		fmt.Printf("❌ Invalid limit: %v\n", err)
		return
	}

	request := &interactionstore.RetrieveDataRequest{
		UserId:         userID,
		StartTimestamp: startTS,
		EndTimestamp:   endTS,
		Limit:          int32(limit),
	}

	fmt.Printf("🔍 Retrieving order interactions for user: %s\n", userID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := session.ISClient.RetrieveOrderInteractions(ctx, request)
	if err != nil {
		fmt.Printf("❌ Error retrieving order data: %v\n", err)
		return
	}

	printOrderResults(response)
}

// handleISRetrieveInteractions handles retrieve_interactions command
func handleISRetrieveInteractions(session *Session, command string) {
	if !session.IsISLoggedIn {
		fmt.Println("❌ Please login to Interaction Store first using: use is && login <app_name> <caller_id>")
		return
	}

	// Parse: retrieve_interactions <user_id> <types> <start_ts> <end_ts> <limit>
	parts := strings.Fields(command)
	if len(parts) != 6 {
		fmt.Println("❌ Usage: retrieve_interactions <user_id> <types> <start_ts> <end_ts> <limit>")
		fmt.Println("   types: click,order or click or order")
		return
	}

	userID := parts[1]
	typesStr := parts[2]
	startTS, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		fmt.Printf("❌ Invalid start_timestamp: %v\n", err)
		return
	}
	endTS, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		fmt.Printf("❌ Invalid end_timestamp: %v\n", err)
		return
	}
	limit, err := strconv.ParseInt(parts[5], 10, 32)
	if err != nil {
		fmt.Printf("❌ Invalid limit: %v\n", err)
		return
	}

	// Parse interaction types
	var interactionTypes []interactionstore.InteractionType
	typesParts := strings.Split(typesStr, ",")
	for _, t := range typesParts {
		t = strings.TrimSpace(strings.ToLower(t))
		switch t {
		case "click":
			interactionTypes = append(interactionTypes, interactionstore.InteractionTypeClick)
		case "order":
			interactionTypes = append(interactionTypes, interactionstore.InteractionTypeOrder)
		default:
			fmt.Printf("❌ Unknown interaction type: %s (use: click, order)\n", t)
			return
		}
	}

	request := &interactionstore.RetrieveInteractionsRequest{
		UserId:           userID,
		InteractionTypes: interactionTypes,
		StartTimestamp:   startTS,
		EndTimestamp:     endTS,
		Limit:            int32(limit),
	}

	fmt.Printf("🔍 Retrieving interactions for user: %s (types: %s)\n", userID, typesStr)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := session.ISClient.RetrieveInteractions(ctx, request)
	if err != nil {
		fmt.Printf("❌ Error retrieving interactions: %v\n", err)
		return
	}

	printInteractionsResults(response)
}

// printClickResults prints click interaction results
func printClickResults(response *interactionstore.RetrieveClickDataResponse) {
	fmt.Println("\n📊 Click Interactions Result:")
	fmt.Printf("Total Events: %d\n", len(response.Data))

	if len(response.Data) == 0 {
		fmt.Println("  No click events found.")
		return
	}

	fmt.Println("\n📝 Events:")
	for i, event := range response.Data {
		fmt.Printf("  %d. CatalogID=%d, ProductID=%d, Timestamp=%d",
			i+1, event.CatalogId, event.ProductId, event.Timestamp)
		if event.Metadata != "" {
			fmt.Printf(", Metadata=%s", event.Metadata)
		}
		fmt.Println()
	}
}

// printOrderResults prints order interaction results
func printOrderResults(response *interactionstore.RetrieveOrderDataResponse) {
	fmt.Println("\n📊 Order Interactions Result:")
	fmt.Printf("Total Events: %d\n", len(response.Data))

	if len(response.Data) == 0 {
		fmt.Println("  No order events found.")
		return
	}

	fmt.Println("\n📝 Events:")
	for i, event := range response.Data {
		fmt.Printf("  %d. CatalogID=%d, ProductID=%d, SubOrderNum=%s, Timestamp=%d",
			i+1, event.CatalogId, event.ProductId, event.SubOrderNum, event.Timestamp)
		if event.Metadata != "" {
			fmt.Printf(", Metadata=%s", event.Metadata)
		}
		fmt.Println()
	}
}

// printInteractionsResults prints mixed interaction results
func printInteractionsResults(response *interactionstore.RetrieveInteractionsResponse) {
	fmt.Println("\n📊 Interactions Result:")

	if len(response.Data) == 0 {
		fmt.Println("  No interactions found.")
		return
	}

	for interactionType, data := range response.Data {
		fmt.Printf("\n📁 %s:\n", interactionType)

		if len(data.ClickEvents) > 0 {
			fmt.Printf("  Click Events (%d):\n", len(data.ClickEvents))
			for i, event := range data.ClickEvents {
				fmt.Printf("    %d. CatalogID=%d, ProductID=%d, Timestamp=%d",
					i+1, event.CatalogId, event.ProductId, event.Timestamp)
				if event.Metadata != "" {
					fmt.Printf(", Metadata=%s", event.Metadata)
				}
				fmt.Println()
			}
		}

		if len(data.OrderEvents) > 0 {
			fmt.Printf("  Order Events (%d):\n", len(data.OrderEvents))
			for i, event := range data.OrderEvents {
				fmt.Printf("    %d. CatalogID=%d, ProductID=%d, SubOrderNum=%s, Timestamp=%d",
					i+1, event.CatalogId, event.ProductId, event.SubOrderNum, event.Timestamp)
				if event.Metadata != "" {
					fmt.Printf(", Metadata=%s", event.Metadata)
				}
				fmt.Println()
			}
		}
	}
}

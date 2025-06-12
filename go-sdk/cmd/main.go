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
}

type Session struct {
	CallerID    string
	CallerToken string
	Client      onfs.Client
	IsLoggedIn  bool
	AppName     string
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

	fmt.Println("🚀 ONFS CLI Tool - SQL-like Interface")
	fmt.Println("Type 'help' for available commands or 'exit' to quit")

	session := &Session{}

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
		if session.IsLoggedIn {
			fmt.Printf("onfs[%s]> ", session.CallerID)
		} else {
			fmt.Print("onfs> ")
		}

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

	switch {
	case command == "help":
		showHelp()
	case strings.HasPrefix(strings.ToLower(command), "login"):
		handleLogin(config, session, command)
	case strings.HasPrefix(strings.ToLower(command), "insert"):
		handleInsert(session, command)
	case strings.HasPrefix(strings.ToLower(command), "select_decoded"):
		handleSelect(session, command, true)
	case strings.HasPrefix(strings.ToLower(command), "select"):
		handleSelect(session, command, false)
	case command == "status":
		showStatus(session)
	case command == "logout":
		handleLogout(session)
	default:
		fmt.Printf("❌ Unknown command: %s\nType 'help' for available commands.\n", command)
	}
}

func showHelp() {
	fmt.Println("\n📖 ONFS CLI Commands:")
	fmt.Println()
	fmt.Println("🔐 Session Management:")
	fmt.Println("  login <app_name> <caller_id> <caller_token>  - Login with credentials and app context")
	fmt.Println("  logout                                       - Logout from current session")
	fmt.Println("  status                                       - Show current session status")
	fmt.Println()
	fmt.Println("�� Data Operations:")
	fmt.Println("  INSERT Syntax:")
	fmt.Println("    insert into <entity>.<fg> (<features>) values (<values>)")
	fmt.Println("    insert into <entity>.<fg1,fg2> (<features>) values (<values>)")
	fmt.Println()
	fmt.Println("  SELECT Syntax:")
	fmt.Println("    select <features> from <entity>.<fg> where <key_schema>=<key_values>")
	fmt.Println("    select_decoded <features> from <entity>.<fg> where <key_schema>=<key_values>")
	fmt.Println()
	fmt.Println("📚 Examples:")
	fmt.Println("  login onfs onfs-cli test")
	fmt.Println("  insert into user.profile (age,location) values (25,'NYC')")
	fmt.Println("  insert into user.profile,preferences (age,location,theme) values (25,'NYC','dark')")
	fmt.Println("  select age,location from user.profile where user_id=123")
	fmt.Println("  select_decoded age,location from user.profile where user_id=123")
	fmt.Println()
	fmt.Println("💡 Data Types:")
	fmt.Println("  Strings: 'value' or \"value\"")
	fmt.Println("  Numbers: 123, 45.67")
	fmt.Println("  Booleans: true, false")
	fmt.Println("  Arrays: [1,2,3] or ['a','b','c']")
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
	session.IsLoggedIn = true
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
	if !session.IsLoggedIn {
		fmt.Println("❌ Not logged in")
		return
	}

	fmt.Printf("👋 Logging out %s from app %s\n", session.CallerID, session.AppName)
	session.IsLoggedIn = false
	session.CallerID = ""
	session.CallerToken = ""
	session.AppName = ""
	session.Client = nil
}

func showStatus(session *Session) {
	fmt.Println("\n📊 Session Status:")
	if session.IsLoggedIn {
		fmt.Printf("  Status: ✅ Logged in\n")
		fmt.Printf("  App Name: %s\n", session.AppName)
		fmt.Printf("  Caller ID: %s\n", session.CallerID)
		fmt.Printf("  Token: %s\n", session.CallerToken)
	} else {
		fmt.Printf("  Status: ❌ Not logged in\n")
		fmt.Println("  Use 'login <app_name> <caller_id> <caller_token>' to authenticate")
	}
}

func handleInsert(session *Session, command string) {
	if !session.IsLoggedIn {
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
	if !session.IsLoggedIn {
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

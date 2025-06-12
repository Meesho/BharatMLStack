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

	"github.com/Meesho/BharatMLStack/go-sdk/pkg/onfs"
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
	fmt.Println("  login <caller_id> <caller_token>  - Login with credentials")
	fmt.Println("  logout                            - Logout from current session")
	fmt.Println("  status                            - Show current session status")
	fmt.Println()
	fmt.Println("📝 Data Operations:")
	fmt.Println("  INSERT Syntax:")
	fmt.Println("    insert into <entity>.<fg> (<features>) values (<values>)")
	fmt.Println("    insert into <entity>.<fg1,fg2> (<features>) values (<values>)")
	fmt.Println()
	fmt.Println("  SELECT Syntax:")
	fmt.Println("    select <features> from <entity>.<fg> where <key_schema>=<key_values>")
	fmt.Println("    select_decoded <features> from <entity>.<fg> where <key_schema>=<key_values>")
	fmt.Println()
	fmt.Println("📚 Examples:")
	fmt.Println("  login onfs-cli test")
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
	if len(parts) != 3 {
		fmt.Println("❌ Usage: login <caller_id> <caller_token>")
		return
	}

	callerID := parts[1]
	callerToken := parts[2]

	fmt.Printf("🔐 Logging in as %s...\n", callerID)

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

	fmt.Printf("✅ Successfully logged in as %s\n", callerID)
}

func handleLogout(session *Session) {
	if !session.IsLoggedIn {
		fmt.Println("❌ Not logged in")
		return
	}

	fmt.Printf("👋 Logging out %s\n", session.CallerID)
	session.IsLoggedIn = false
	session.CallerID = ""
	session.CallerToken = ""
	session.Client = nil
}

func showStatus(session *Session) {
	fmt.Println("\n📊 Session Status:")
	if session.IsLoggedIn {
		fmt.Printf("  Status: ✅ Logged in\n")
		fmt.Printf("  Caller ID: %s\n", session.CallerID)
		fmt.Printf("  Token: %s\n", session.CallerToken)
	} else {
		fmt.Printf("  Status: ❌ Not logged in\n")
		fmt.Println("  Use 'login <caller_id> <caller_token>' to authenticate")
	}
}

func handleInsert(session *Session, command string) {
	if !session.IsLoggedIn {
		fmt.Println("❌ Please login first using: login <caller_id> <caller_token>")
		return
	}

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

	fmt.Printf("📝 Persisting features for entity: %s\n", insertData.Entity)

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
		fmt.Println("❌ Please login first using: login <caller_id> <caller_token>")
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

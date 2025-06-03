package main

import (
	"context"
	"fmt"
	"log"

	gosdk "github.com/Meesho/BharatMLStack/online-feature-store/sdks/go/pkg"
)

func main() {
	// Create client
	client := getClient()

	// Run examples
	retrieveFeaturesExample(client)
	retrieveDecodedFeaturesExample(client)
	persistFeaturesExample(client)
}

// getClient creates a new online-feature-store client with the given configuration
func getClient() *gosdk.ClientV1 {
	config := &gosdk.Config{
		Host:        "online-feature-store-api.stg.meesho.int", // change to your online-feature-store host
		Port:        "80",                                      // change to your online-feature-store port
		DeadLine:    500000,                                    // change to your online-feature-store deadline
		PlainText:   true,                                      // change to your online-feature-store plain text
		CallerId:    "test-caller",                             // change to your online-feature-store caller id
		CallerToken: "test",                                    // change to your online-feature-store caller token
		BatchSize:   50,                                        // change to your online-feature-store batch size
	}
	client := gosdk.NewClientV1(config)
	return client
}

// createSampleQuery creates a sample query for feature retrieval
func createSampleQuery() *gosdk.Query {
	return &gosdk.Query{
		EntityLabel: "user_sscat",
		Keys: []gosdk.Keys{
			{Cols: []string{"100073440", "5999"}},
			{Cols: []string{"100000765", "4400"}},
			{Cols: []string{"100000779", "1013"}},
			{Cols: []string{"100001215", "4101"}},
			{Cols: []string{"100004719", "1207"}},
			{Cols: []string{"100009488", "1094"}},
			{Cols: []string{"10001089", "3172"}},
			{Cols: []string{"100013069", "5582"}},
			{Cols: []string{"100016752", "2374"}},
		},
		KeysSchema: []string{"user_id", "sscat_id"},
		FeatureGroups: []gosdk.FeatureGroup{
			{
				Label: "derived_string",
				FeatureLabels: []string{
					"attribute_value_1_click_28days",
					"attribute_value_1_orders_56days",
					"attribute_value_2_click_28days",
					"attribute_value_2_orders_56days",
					"attribute_value_3_click_28days",
					"attribute_value_3_orders_56days",
					"attribute_value_1_orders_28days",
					"attribute_value_2_orders_28days",
					"attribute_value_3_orders_28days",
				},
			},
			{
				Label: "derived_int64",
				FeatureLabels: []string{
					"user_sscat__days_since_last_order",
					"user_sscat__platform_clicks_14_days",
					"user_sscat__platform_orders_56_days",
					"user_sscat__platform_orders_28_days",
					"clicks_7day",
					"orders_7day",
					"views_7day",
				},
			},
		},
	}
}

// createSamplePersistRequest creates a sample request for feature persistence
func createSamplePersistRequest() *gosdk.PersistFeaturesRequest {
	return &gosdk.PersistFeaturesRequest{
		EntityLabel: "user_sscat",
		KeysSchema:  []string{"user_id", "sscat_id"},
		FeatureGroups: []gosdk.FeatureGroupSchema{
			{
				Label: "derived_int64",
				FeatureLabels: []string{
					"user_sscat__platform_clicks_14_days",
					"clicks_7day",
					"user_sscat__platform_orders_28_days",
				},
			},
			{
				Label: "derived_string",
				FeatureLabels: []string{
					"attribute_value_1_click_28days",
					"attribute_value_1_orders_56days",
					"attribute_value_2_click_28days",
					"attribute_value_2_orders_56days",
					"attribute_value_3_click_28days",
					"attribute_value_3_orders_56days",
					"attribute_value_1_orders_28days",
					"attribute_value_2_orders_28days",
				},
			},
		},
		Data: []gosdk.Data{
			{
				KeyValues: []string{"-2", "-3"},
				FeatureValues: []gosdk.FeatureValues{
					{
						Values: gosdk.Values{
							Int64Values: []int64{25, 12, 5},
						},
					},
					{
						Values: gosdk.Values{
							StringValues: []string{
								"high", "medium", "low", "high",
								"medium", "low", "high", "medium",
							},
							Vector: []gosdk.Vector{
								{
									Values: gosdk.Values{
										Int32Values: []int32{0, 1, 2, 3, 4, 5},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// retrieveFeaturesExample demonstrates feature retrieval
func retrieveFeaturesExample(client *gosdk.ClientV1) {
	request := createSampleQuery()

	// Get features
	response, err := client.RetrieveFeatures(context.Background(), request)
	if err != nil {
		log.Fatalf("Failed to get features: %v", err)
	}

	// Print results
	fmt.Println("\n=== Feature Retrieval Results ===")
	fmt.Printf("Keys Schema: %v\n", response.KeysSchema)
	for _, row := range response.Rows {
		fmt.Printf("\nKeys: %v\n", row.Keys)
		fmt.Printf("Columns: %v\n", row.Columns)
	}
}

// retrieveDecodedFeaturesExample demonstrates decoded feature retrieval
func retrieveDecodedFeaturesExample(client *gosdk.ClientV1) {
	request := createSampleQuery()

	// Get decoded features
	response, err := client.RetrieveDecodedFeatures(context.Background(), request)
	if err != nil {
		log.Fatalf("Failed to get decoded features: %v", err)
	}

	// Print results
	fmt.Println("\n=== Decoded Feature Retrieval Results ===")
	fmt.Printf("Keys Schema: %v\n", response.KeysSchema)
	for _, row := range response.Rows {
		fmt.Printf("\nKeys: %v\n", row.Keys)
		fmt.Printf("Decoded Columns: %v\n", row.Columns)
	}
}

// persistFeaturesExample demonstrates feature persistence
func persistFeaturesExample(client *gosdk.ClientV1) {
	request := createSamplePersistRequest()

	// Call PersistFeatures
	response, err := client.PersistFeatures(context.Background(), request)
	if err != nil {
		log.Fatalf("Failed to persist features: %v", err)
	}

	// Print response
	fmt.Println("\n=== Feature Persistence Results ===")
	fmt.Printf("Response Message: %s\n", response.Message)
}

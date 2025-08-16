package main

import (
	"context"
	"fmt"
	"log"
	"time"

	retrieve "github.com/Meesho/BharatMLStack/go-sdk/pkg/proto/onfs/retrieve"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	fmt.Println("Attempting to connect to the feature store...")

	conn, err := grpc.NewClient(
		"online-feature-store-api-mp.prd.meesho.int:80",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := retrieve.NewFeatureServiceClient(conn)

	fmt.Println("Connection successful. Retrieving features...")

	md := metadata.New(map[string]string{
		"online-feature-store-auth-token": "test",
		"online-feature-store-caller-id":  "model-proxy-service-experiment",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := client.RetrieveFeatures(ctx, &retrieve.Query{
		EntityLabel: "catalog",
		FeatureGroups: []*retrieve.FeatureGroup{
			{
				Label:         "derived_2_fp32",
				FeatureLabels: []string{"sbid_value"},
			},
			{
				Label: "derived_fp16",
				FeatureLabels: []string{
					"search__organic_clicks_by_views_3_days_percentile",
					"search__organic_clicks_by_views_5_days_percentile",
				},
			},
		},
		KeysSchema: []string{"catalog_id"},
		Keys: []*retrieve.Keys{
			{Cols: []string{"176"}},
			{Cols: []string{"179"}},
		},
	})
	if err != nil {
		log.Fatalf("could not retrieve features: %v", err)
	}

	fmt.Printf("Successfully retrieved features: %v\n", resp)
}
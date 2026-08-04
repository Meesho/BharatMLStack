// onyxdb-e2e-test — end-to-end test for the onyxdb Go SDK against a real stack.
//
// Uses the SDK's etcd-backed client (sdk.NewClient) which automatically
// discovers topology, manages shard routing, and pools connections.
//
// For the VM data plane, tunnel etcd and read-server ports via IAP:
//
//	gcloud compute start-iap-tunnel vm-datascience-mlp-rocksdb-poc-prd-ase1 9091 \
//	  --local-host-port=localhost:9091 \
//	  --project meesho-datascience-prd-0622 --zone asia-southeast1-a &
//	gcloud compute start-iap-tunnel vm-datascience-mlp-rocksdb-poc-prd-ase1 2379 \
//	  --local-host-port=localhost:2379 \
//	  --project meesho-datascience-prd-0622 --zone asia-southeast1-a &
//
// Then:
//
//	cd quick-start/onyxdb/ws && GOWORK=off go run .
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
	sdk "github.com/Meesho/BharatMLStack/onyxdb/onyxdb-go-sdk"
)

// ── Test harness ────────────────────────────────────────────────────────────

type testResult struct {
	name   string
	passed bool
	detail string
}

var allResults []testResult

func check(name string, passed bool, detail string) {
	tag := "PASS"
	if !passed {
		tag = "FAIL"
	}
	fmt.Printf("  [%s] %s", tag, name)
	if detail != "" {
		fmt.Printf(" — %s", detail)
	}
	fmt.Println()
	allResults = append(allResults, testResult{name, passed, detail})
}

func main() {
	etcdAddr := flag.String("etcd", "localhost:2379", "etcd endpoint")
	tenant := flag.String("tenant", "ds", "tenant")
	store := flag.String("store", "ds_catalog_user_geohash_1_3", "store")
	timeoutMs := flag.Int("timeout", 5000, "per-request timeout in ms (generous for IAP tunnels)")
	flag.Parse()

	fmt.Printf("onyxdb Go SDK e2e test — etcd-backed client\n")
	fmt.Printf("  tenant/store: %s/%s\n", *tenant, *store)
	fmt.Printf("  etcd:         %s\n\n", *etcdAddr)

	// Known-present keys in the dataset.
	// Known-present keys in the dataset.

	realKeys := []string{
		"catalog__user_geohash_1_3:derived_fp32:482333412|1984",
		"catalog__user_geohash_1_3:derived_fp32:222213032|2050",
		"catalog__user_geohash_1_3:derived_fp32:484643271|2050",
		"catalog__user_geohash_1_3:derived_fp32:191097285|1",
		"catalog__user_geohash_1_3:derived_fp32:128191251|2053",
		// "catalog__user_geohash_1_3:derived_fp32:10000010|1692",
		// "catalog__user_geohash_1_3:derived_fp32:100000267|4632",
		// "catalog__user_geohash_1_3:derived_fp32:10000035|547",
		// "catalog__user_geohash_1_3:derived_fp32:100000451|509",
		// "catalog__user_geohash_1_3:derived_fp32:100000464|1694",
		// "catalog__user_geohash_1_3:derived_fp32:100000497|409",
		// "catalog__user_geohash_1_3:derived_fp32:100000667|3495",
		// "catalog__user_geohash_1_3:derived_fp32:100000802|1661",
		// "catalog__user_geohash_1_3:derived_fp32:100000802|4622",
		// "catalog__user_geohash_1_3:derived_fp32:100001043|3492",
		// "catalog__user_geohash_1_3:derived_fp32:100001113|4211",
		// "catalog__user_geohash_1_3:derived_fp32:100001360|3492",
		// "catalog__user_geohash_1_3:derived_fp32:100001519|2425",
		// "catalog__user_geohash_1_3:derived_fp32:100001519|4245",
		// "catalog__user_geohash_1_3:derived_fp32:100001619|537",
		// "catalog__user_geohash_1_3:derived_fp32:100001713|531",
		// "catalog__user_geohash_1_3:derived_fp32:100001744|515",
		// "catalog__user_geohash_1_3:derived_fp32:100001844|1671",
		// "catalog__user_geohash_1_3:derived_fp32:100001882|542",
		// "catalog__user_geohash_1_3:derived_fp32:10000196|1631",
		// "catalog__user_geohash_1_3:derived_fp32:100002000|519",
		// "catalog__user_geohash_1_3:derived_fp32:100002016|4253",
		// "catalog__user_geohash_1_3:derived_fp32:100002267|4604",
		// "catalog__user_geohash_1_3:derived_fp32:100002285|1290",
		// "catalog__user_geohash_1_3:derived_fp32:10000266|4212",
		// "catalog__user_geohash_1_3:derived_fp32:100002736|2340",
		// "catalog__user_geohash_1_3:derived_fp32:100002750|1304",
		// "catalog__user_geohash_1_3:derived_fp32:100002842|2412",
		// "catalog__user_geohash_1_3:derived_fp32:100002842|514",
		// "catalog__user_geohash_1_3:derived_fp32:100002842|577",
		// "catalog__user_geohash_1_3:derived_fp32:100002843|1675",
		// "catalog__user_geohash_1_3:derived_fp32:100002846|549",
		// "catalog__user_geohash_1_3:derived_fp32:100002944|1373",
		// "catalog__user_geohash_1_3:derived_fp32:100002944|1660",
		// "catalog__user_geohash_1_3:derived_fp32:100003181|1302",
		// "catalog__user_geohash_1_3:derived_fp32:100003268|522",
		// "catalog__user_geohash_1_3:derived_fp32:10000326|3493",
		// "catalog__user_geohash_1_3:derived_fp32:10000326|924",
		// "catalog__user_geohash_1_3:derived_fp32:100003391|1306",
		// "catalog__user_geohash_1_3:derived_fp32:100003476|1695",
		// "catalog__user_geohash_1_3:derived_fp32:10000352|507",
		// "catalog__user_geohash_1_3:derived_fp32:100003597|1307",
		// "catalog__user_geohash_1_3:derived_fp32:100003783|514",
		// "catalog__user_geohash_1_3:derived_fp32:100003786|5375",
		// "catalog__user_geohash_1_3:derived_fp32:100003872|1285",
	}
	missKey := "catalog__user_geohash_1_3:derived_fp32:999999999|0"

	// ── Suite A: Etcd topology verification ─────────────────────────────────

	fmt.Println("=== Suite A: Etcd topology ===")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	etcd, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{*etcdAddr},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: cannot connect to etcd: %v\n", err)
		os.Exit(1)
	}
	defer etcd.Close()

	fmt.Println("\n--- A1: Read activeVersion ---")
	{
		resp, err := etcd.Get(ctx, model.ActiveVersionPath(*tenant, *store))
		if err != nil || len(resp.Kvs) == 0 {
			check("etcd/activeVersion", false, fmt.Sprintf("err=%v kvs=%d", err, len(resp.Kvs)))
		} else {
			activeVer := string(resp.Kvs[0].Value)
			check("etcd/activeVersion", activeVer != "", fmt.Sprintf("version=%q", activeVer))

			resp2, err := etcd.Get(ctx, model.VersionPrefix(*tenant, *store, activeVer))
			if err != nil || len(resp2.Kvs) == 0 {
				check("etcd/versionMeta", false, fmt.Sprintf("err=%v", err))
			} else {
				var m model.VersionMeta
				json.Unmarshal(resp2.Kvs[0].Value, &m)
				check("etcd/versionMeta-shardCount", m.ShardCount > 0,
					fmt.Sprintf("shardCount=%d", m.ShardCount))
				check("etcd/versionMeta-status", m.Status == model.StatusActive || m.Status == model.StatusReady,
					fmt.Sprintf("got=%s", m.Status))
			}
		}
	}

	fmt.Println("\n--- A2: Pod registrations ---")
	{
		resp, err := etcd.Get(ctx, model.PodWatchPrefix(*tenant, *store), clientv3.WithPrefix())
		if err != nil {
			check("etcd/pod-registrations", false, err.Error())
		} else {
			check("etcd/pod-registrations", len(resp.Kvs) > 0,
				fmt.Sprintf("pods=%d", len(resp.Kvs)))
			for _, kv := range resp.Kvs {
				var pd model.PodData
				json.Unmarshal(kv.Value, &pd)
				fmt.Printf("    %s → serving=%s warm=%v\n", string(kv.Key), pd.ServingVersion, pd.WarmVersions)
			}
		}
	}

	// ── Create SDK client (topology auto-discovered from etcd) ──────────────

	fmt.Println("\n=== Creating SDK client (etcd-backed, auto topology) ===")

	client, err := sdk.NewClient(sdk.Config{
		EtcdEndpoints: []string{*etcdAddr},
		Tenant:        *tenant,
		Store:         *store,
		TimeoutMs:     *timeoutMs,
		ConnsPerPod:   4,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: sdk.NewClient: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Give the topology watcher a moment to initialise from etcd.
	time.Sleep(2 * time.Second)
	fmt.Println("  SDK client ready")

	// ── Suite B: StringGet over TCP via the SDK ─────────────────────────────

	fmt.Println("\n=== Suite B: StringGet via SDK client ===")

	fmt.Println("\n--- B1: StringGet (hits) ---")
	for _, keyStr := range realKeys {
		val, err := client.StringGet(context.Background(), []byte(keyStr))
		if err != nil {
			check(fmt.Sprintf("StringGet(%s)", keyStr), false, err.Error())
		} else {
			check(fmt.Sprintf("StringGet(%s)", keyStr),
				len(val) > 0, fmt.Sprintf("value=%d bytes", len(val)))
		}
	}

	fmt.Println("\n--- B2: StringGet (miss) ---")
	{
		_, err := client.StringGet(context.Background(), []byte(missKey))
		check("StringGet(miss)",
			errors.Is(err, sdk.ErrKeyNotFound), fmt.Sprintf("err=%v", err))
	}

	// ── Suite C: StringBatchGet ─────────────────────────────────────────────

	fmt.Println("\n=== Suite C: StringBatchGet ===")
	fmt.Println("\n--- C1: StringBatchGet (all keys + miss) ---")
	{
		var batchKeys [][]byte
		for _, keyStr := range realKeys {
			batchKeys = append(batchKeys, []byte(keyStr))
		}
		batchKeys = append(batchKeys, []byte(missKey))

		results, err := client.StringBatchGet(context.Background(), batchKeys)
		if err != nil {
			check("StringBatchGet", false, fmt.Sprintf("err=%v", err))
		} else {
			ok := len(results) == len(batchKeys)
			hits := 0
			for i, r := range results {
				isLast := i == len(results)-1
				if isLast {
					// miss key — expect nil value
					ok = ok && r.Value == nil
				} else {
					// real keys — expect non-nil value
					if len(r.Value) > 0 {
						hits++
					} else {
						ok = false
					}
				}
			}
			check(fmt.Sprintf("StringBatchGet(%d keys)", len(batchKeys)),
				ok, fmt.Sprintf("hits=%d/%d", hits, len(results)))
		}
	}
	}

	// ── Suite D: BuildStringKey helper ──────────────────────────────────────

	fmt.Println("\n=== Suite D: BuildStringKey ===")
	{
		k := sdk.BuildStringKey("catalog__user_geohash_1_3", 105959719, 4236)
		want := "catalog__user_geohash_1_3:105959719|4236"
		check("BuildStringKey(2 pks)", string(k) == want, fmt.Sprintf("got=%q", k))

		k2 := sdk.BuildStringKey("entity", 1, 2, 3)
		check("BuildStringKey(3 pks)", string(k2) == "entity:1|2|3", fmt.Sprintf("got=%q", k2))
	}

	// ── Summary ─────────────────────────────────────────────────────────────

	fmt.Println("\n============================================================")
	passed, failed := 0, 0
	for _, r := range allResults {
		if r.passed {
			passed++
		} else {
			failed++
		}
	}
	fmt.Printf("  %d passed, %d failed, %d total\n", passed, failed, passed+failed)
	if failed > 0 {
		fmt.Println("  SOME TESTS FAILED")
		os.Exit(1)
	}
	fmt.Println("  ALL TESTS PASSED")
	fmt.Println("============================================================")
}

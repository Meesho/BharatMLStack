CREATE KEYSPACE IF NOT EXISTS interaction_store_test
WITH replication = {
    'class': 'NetworkTopologyStrategy',
    'replication_factor': 3
};

USE interaction_store_test;

CREATE TABLE click_interactions_bucket1 (
    user_id text PRIMARY KEY,
    week_0 blob,  
    week_1 blob, 
    week_2 blob,
    week_3 blob,
    week_4 blob,
    week_5 blob,
    week_6 blob,
    week_7 blob
) WITH compaction = {'class': 'SizeTieredCompactionStrategy'};

CREATE TABLE click_interactions_bucket2 (
    user_id text PRIMARY KEY,
    week_8 blob,  
    week_9 blob, 
    week_10 blob,
    week_11 blob,
    week_12 blob,
    week_13 blob,
    week_14 blob,
    week_15 blob
) WITH compaction = {'class': 'SizeTieredCompactionStrategy'};

CREATE TABLE click_interactions_bucket3 (
    user_id text PRIMARY KEY,
    week_16 blob,  
    week_17 blob, 
    week_18 blob,
    week_19 blob,
    week_20 blob,
    week_21 blob,
    week_22 blob,
    week_23 blob
) WITH compaction = {'class': 'SizeTieredCompactionStrategy'};

CREATE TABLE click_interactions_metadata (
    user_id text PRIMARY KEY,
    week_0 int,
    week_1 int,
    week_2 int,
    week_3 int,
    week_4 int,
    week_5 int,
    week_6 int,
    week_7 int,
    week_8 int,
    week_9 int,
    week_10 int,
    week_11 int,
    week_12 int,
    week_13 int,
    week_14 int,
    week_15 int,
    week_16 int,
    week_17 int,
    week_18 int,
    week_19 int,
    week_20 int,
    week_21 int,
    week_22 int,
    week_23 int
) WITH compaction = {'class': 'SizeTieredCompactionStrategy'};

CREATE TABLE order_interactions_bucket1 (
    user_id text PRIMARY KEY,
    week_0 blob,  
    week_1 blob, 
    week_2 blob,
    week_3 blob,
    week_4 blob,
    week_5 blob,
    week_6 blob,
    week_7 blob
) WITH compaction = {'class': 'SizeTieredCompactionStrategy'};

CREATE TABLE order_interactions_bucket2 (
    user_id text PRIMARY KEY,
    week_8 blob,  
    week_9 blob, 
    week_10 blob,
    week_11 blob,
    week_12 blob,
    week_13 blob,
    week_14 blob,
    week_15 blob
) WITH compaction = {'class': 'SizeTieredCompactionStrategy'};

CREATE TABLE order_interactions_bucket3 (
    user_id text PRIMARY KEY,
    week_16 blob,  
    week_17 blob, 
    week_18 blob,
    week_19 blob,
    week_20 blob,
    week_21 blob,
    week_22 blob,
    week_23 blob
) WITH compaction = {'class': 'SizeTieredCompactionStrategy'};

CREATE TABLE order_interactions_metadata (
    user_id text PRIMARY KEY,
    week_0 int,
    week_1 int,
    week_2 int,
    week_3 int,
    week_4 int,
    week_5 int,
    week_6 int,
    week_7 int,
    week_8 int,
    week_9 int,
    week_10 int,
    week_11 int,
    week_12 int,
    week_13 int,
    week_14 int,
    week_15 int,
    week_16 int,
    week_17 int,
    week_18 int,
    week_19 int,
    week_20 int,
    week_21 int,
    week_22 int,
    week_23 int
) WITH compaction = {'class': 'SizeTieredCompactionStrategy'};

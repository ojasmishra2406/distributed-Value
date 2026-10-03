package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	pb "github.com/ojasmishra2406/distributed-Value/api/proto"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var target string

var rootCmd = &cobra.Command{
	Use:   "kv-cli",
	Short: "CLI for Antigravity Distributed KV Store",
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&target, "target", "t", "localhost:8080", "Gateway address")
	rootCmd.AddCommand(putCmd, getCmd, searchCmd, statusCmd)
}

func getClient() (pb.KVServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return pb.NewKVServiceClient(conn), conn, nil
}

var putCmd = &cobra.Command{
	Use:   "put <key> <val>",
	Short: "Put a key-value pair",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, val := args[0], args[1]
		vecStr, _ := cmd.Flags().GetString("vector")
		
		var vector []float32
		if vecStr != "" {
			parts := strings.Split(vecStr, ",")
			for _, p := range parts {
				f, err := strconv.ParseFloat(p, 32)
				if err != nil {
					return err
				}
				vector = append(vector, float32(f))
			}
		}

		client, conn, err := getClient()
		if err != nil {
			return err
		}
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_, err = client.Put(ctx, &pb.PutRequest{
			Key:       []byte(key),
			Value:     []byte(val),
			Timestamp: time.Now().UnixNano(),
			Vector:    vector,
		})
		if err != nil {
			return err
		}
		fmt.Println("OK")
		return nil
	},
}

var getCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a value by key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, conn, err := getClient()
		if err != nil {
			return err
		}
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		res, err := client.Get(ctx, &pb.GetRequest{Key: []byte(args[0])})
		if err != nil {
			return err
		}
		if !res.Found || res.Tombstone {
			fmt.Println("Not found")
			return nil
		}
		fmt.Printf("%s\n", res.Value)
		return nil
	},
}

var searchCmd = &cobra.Command{
	Use:   "search <vector_csv>",
	Short: "Vector search (KNN)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		parts := strings.Split(args[0], ",")
		var vector []float32
		for _, p := range parts {
			f, _ := strconv.ParseFloat(p, 32)
			vector = append(vector, float32(f))
		}

		client, conn, err := getClient()
		if err != nil {
			return err
		}
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		res, err := client.Search(ctx, &pb.SearchRequest{Vector: vector, TopK: 5})
		if err != nil {
			return err
		}
		for i, r := range res.Records {
			fmt.Printf("%d: %s -> %s\n", i+1, r.Key, r.Value)
		}
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check gateway health",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, conn, err := getClient()
		if err != nil {
			return err
		}
		defer conn.Close()
		
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		
		res, err := client.Heartbeat(ctx, &pb.HeartbeatRequest{NodeId: "cli"})
		if err != nil {
			return err
		}
		fmt.Printf("Gateway Alive: %v\n", res.Alive)
		return nil
	},
}

func init() {
	putCmd.Flags().String("vector", "", "Comma-separated floats for FlatVectorIndex vector")
}

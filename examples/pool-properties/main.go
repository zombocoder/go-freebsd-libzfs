//go:build freebsd

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/zombocoder/go-freebsd-libzfs/zpool"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <pool-name>\n", os.Args[0])
		fmt.Println("Shows detailed properties for a ZFS pool")
		os.Exit(1)
	}

	poolName := os.Args[1]
	ctx := context.Background()

	client, err := zpool.New(ctx)
	if err != nil {
		log.Fatalf("Failed to create pool client: %v", err)
	}
	defer client.Close()

	// Check if pool exists
	pool, err := client.Get(ctx, poolName)
	if err != nil {
		log.Fatalf("Failed to get pool %s: %v", poolName, err)
	}

	fmt.Printf("Pool: %s\n", pool.Name)
	fmt.Printf("GUID: %d\n", pool.GUID)
	fmt.Printf("Health: %s\n", pool.Health)
	fmt.Printf("State: %s\n", pool.State)
	fmt.Println()

	// Get all available properties
	props, err := client.GetProperties(ctx, poolName)
	if err != nil {
		log.Fatalf("Failed to get properties: %v", err)
	}

	fmt.Printf("Properties for pool '%s':\n", poolName)
	fmt.Println("=" + fmt.Sprintf("%*s", len(poolName)+20, ""))

	for name, prop := range props {
		fmt.Printf("%-20s: %v (source: %s)\n", name, prop.Value, prop.Source)
	}

	// Test specific property getters
	fmt.Println("\nSpecific property tests:")
	
	if size, err := client.GetStringProperty(ctx, poolName, "size"); err == nil {
		fmt.Printf("Size (string): %s\n", size)
	}
	
	if free, err := client.GetStringProperty(ctx, poolName, "free"); err == nil {
		fmt.Printf("Free (string): %s\n", free)
	}
	
	if allocated, err := client.GetStringProperty(ctx, poolName, "allocated"); err == nil {
		fmt.Printf("Allocated (string): %s\n", allocated)
	}
}
//go:build freebsd

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/zombocoder/go-freebsd-libzfs/zfs"
)

func main() {
	ctx := context.Background()

	client, err := zfs.New(ctx)
	if err != nil {
		log.Fatalf("Failed to create dataset client: %v", err)
	}
	defer client.Close()

	// Get parent dataset filter from command line if provided
	var parentFilter string
	if len(os.Args) > 1 {
		parentFilter = os.Args[1]
		fmt.Printf("Listing snapshots for parent dataset: %s\n", parentFilter)
	} else {
		fmt.Println("Listing all snapshots")
	}

	snapshots, err := client.ListSnapshots(ctx, parentFilter)
	if err != nil {
		log.Fatalf("Failed to list snapshots: %v", err)
	}

	if len(snapshots) == 0 {
		fmt.Println("No snapshots found")
		return
	}

	fmt.Printf("\nFound %d snapshots:\n", len(snapshots))
	fmt.Println(strings.Repeat("=", 80))

	for _, snapshot := range snapshots {
		// Extract snapshot name from full path (everything after @)
		parts := strings.Split(snapshot.Name, "@")
		snapName := "unknown"
		if len(parts) == 2 {
			snapName = parts[1]
		}

		fmt.Printf("%-50s | Parent: %s\n", snapshot.Name, snapshot.Parent)
		fmt.Printf("  Snapshot Name: %-30s | Type: %s | GUID: %d\n", 
			snapName, snapshot.Type, snapshot.GUID)

		// Try to get some properties for the snapshot
		if snapshot.Name != "" {
			props, err := client.GetProperties(ctx, snapshot.Name, "used", "referenced")
			if err == nil {
				for name, prop := range props {
					fmt.Printf("  %s: %v ", name, prop.Value)
				}
				fmt.Println()
			}
		}
		fmt.Println()
	}

	// Group snapshots by parent
	parentMap := make(map[string][]zfs.Snapshot)
	for _, snapshot := range snapshots {
		parentMap[snapshot.Parent] = append(parentMap[snapshot.Parent], snapshot)
	}

	fmt.Printf("\nSnapshot summary by parent dataset:\n")
	fmt.Println(strings.Repeat("=", 50))
	for parent, snaps := range parentMap {
		fmt.Printf("%-30s: %d snapshots\n", parent, len(snaps))
	}
}
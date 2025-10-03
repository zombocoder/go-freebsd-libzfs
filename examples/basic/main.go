//go:build freebsd

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/zombocoder/go-freebsd-libzfs/version"
	"github.com/zombocoder/go-freebsd-libzfs/zpool"
	"github.com/zombocoder/go-freebsd-libzfs/zfs"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("=== FreeBSD Go ZFS Library Test ===")

	// Test version detection
	fmt.Println("\n1. Testing version detection...")
	info, err := version.Detect(ctx)
	if err != nil {
		log.Printf("Version detection failed: %v", err)
	} else {
		fmt.Printf("Detected: %s\n", info.String())
	}

	// Test capability probing
	fmt.Println("\n2. Testing capability probing...")
	caps, err := version.ProbeCapabilities(ctx)
	if err != nil {
		log.Printf("Capability probing failed: %v", err)
	} else {
		fmt.Printf("Capabilities - Pools: %v, Datasets: %v, Snapshots: %v\n", 
			caps.Pools, caps.Datasets, caps.Snapshots)
	}

	// Test pool operations
	fmt.Println("\n3. Testing pool operations...")
	poolClient, err := zpool.New(ctx)
	if err != nil {
		log.Printf("Failed to create pool client: %v", err)
		return
	}
	defer poolClient.Close()

	pools, err := poolClient.List(ctx)
	if err != nil {
		log.Printf("Failed to list pools: %v", err)
	} else {
		fmt.Printf("Found %d pools:\n", len(pools))
		for _, pool := range pools {
			fmt.Printf("  - %s (GUID: %d, Health: %s, State: %s)\n", 
				pool.Name, pool.GUID, pool.Health, pool.State)
			
			// Try to get some properties for the first pool
			if len(pools) > 0 && pool.Name != "" {
				fmt.Printf("    Properties for %s:\n", pool.Name)
				props, err := poolClient.GetProperties(ctx, pool.Name, "size", "free", "allocated")
				if err != nil {
					fmt.Printf("      Failed to get properties: %v\n", err)
				} else {
					for name, prop := range props {
						fmt.Printf("      %s: %v (source: %s)\n", name, prop.Value, prop.Source)
					}
				}
			}
		}
	}

	// Test dataset operations
	fmt.Println("\n4. Testing dataset operations...")
	datasetClient, err := zfs.New(ctx)
	if err != nil {
		log.Printf("Failed to create dataset client: %v", err)
		return
	}
	defer datasetClient.Close()

	datasets, err := datasetClient.List(ctx, false) // non-recursive for speed
	if err != nil {
		log.Printf("Failed to list datasets: %v", err)
	} else {
		fmt.Printf("Found %d datasets (non-recursive):\n", len(datasets))
		for i, dataset := range datasets {
			fmt.Printf("  - %s (Type: %s, GUID: %d)\n", 
				dataset.Name, dataset.Type, dataset.GUID)
			
			// Only show properties for first few datasets to avoid spam
			if i < 3 && dataset.Name != "" {
				fmt.Printf("    Properties for %s:\n", dataset.Name)
				props, err := datasetClient.GetProperties(ctx, dataset.Name, "used", "avail", "mountpoint")
				if err != nil {
					fmt.Printf("      Failed to get properties: %v\n", err)
				} else {
					for name, prop := range props {
						fmt.Printf("      %s: %v (source: %s)\n", name, prop.Value, prop.Source)
					}
				}
			}
		}
	}

	// Test snapshots
	fmt.Println("\n5. Testing snapshot listing...")
	snapshots, err := datasetClient.ListSnapshots(ctx, "")
	if err != nil {
		log.Printf("Failed to list snapshots: %v", err)
	} else {
		fmt.Printf("Found %d snapshots:\n", len(snapshots))
		for i, snapshot := range snapshots {
			if i < 5 { // Show only first 5 to avoid spam
				fmt.Printf("  - %s (Parent: %s)\n", snapshot.Name, snapshot.Parent)
			}
		}
		if len(snapshots) > 5 {
			fmt.Printf("  ... and %d more\n", len(snapshots)-5)
		}
	}

	fmt.Println("\n=== Test completed successfully! ===")
}
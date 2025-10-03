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
	if len(os.Args) < 3 {
		fmt.Printf("Usage: %s <operation> <target> [options]\n", os.Args[0])
		fmt.Println("Operations:")
		fmt.Println("  create <dataset@snapshot>       - Create a snapshot")
		fmt.Println("  create-recursive <dataset@snapshot> - Create a recursive snapshot")
		fmt.Println("  destroy <snapshot>               - Destroy a snapshot")
		fmt.Println("  rollback <dataset> <snapshot>    - Rollback dataset to snapshot")
		fmt.Println("  rollback-force <dataset> <snapshot> - Force rollback dataset to snapshot")
		fmt.Println("  list [parent-dataset]            - List snapshots (optionally for specific parent)")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  create zroot/test@backup         - Create snapshot zroot/test@backup")
		fmt.Println("  create-recursive zroot@full      - Create recursive snapshot zroot@full")
		fmt.Println("  destroy zroot/test@backup        - Destroy snapshot zroot/test@backup")
		fmt.Println("  rollback zroot/test zroot/test@backup - Rollback zroot/test to @backup")
		fmt.Println("  list zroot/test                  - List snapshots for zroot/test")
		fmt.Println("")
		fmt.Println("WARNING: Rollback operations are destructive!")
		os.Exit(1)
	}

	operation := os.Args[1]
	target := os.Args[2]
	
	ctx := context.Background()

	client, err := zfs.New(ctx)
	if err != nil {
		log.Fatalf("Failed to create ZFS client: %v", err)
	}
	defer client.Close()

	switch operation {
	case "create":
		err = createSnapshot(ctx, client, target, false)
	case "create-recursive":
		err = createSnapshot(ctx, client, target, true)
	case "destroy":
		err = destroySnapshot(ctx, client, target)
	case "rollback":
		if len(os.Args) < 4 {
			log.Fatalf("Rollback requires dataset and snapshot names")
		}
		snapshot := os.Args[3]
		err = rollbackToSnapshot(ctx, client, target, snapshot, false)
	case "rollback-force":
		if len(os.Args) < 4 {
			log.Fatalf("Force rollback requires dataset and snapshot names")
		}
		snapshot := os.Args[3]
		err = rollbackToSnapshot(ctx, client, target, snapshot, true)
	case "list":
		parent := ""
		if target != "" {
			parent = target
		}
		err = listSnapshots(ctx, client, parent)
	default:
		log.Fatalf("Unknown operation: %s", operation)
	}

	if err != nil {
		log.Fatalf("Operation failed: %v", err)
	}

	fmt.Printf("Operation '%s' completed successfully\n", operation)
}

func createSnapshot(ctx context.Context, client *zfs.Client, snapshotName string, recursive bool) error {
	if !strings.Contains(snapshotName, "@") {
		return fmt.Errorf("snapshot name must be in format dataset@snapshot")
	}

	fmt.Printf("Creating %ssnapshot: %s\n", 
		map[bool]string{true: "recursive ", false: ""}[recursive], snapshotName)

	properties := map[string]string{
		// Common snapshot properties
	}

	return client.CreateSnapshot(ctx, snapshotName, recursive, properties)
}

func destroySnapshot(ctx context.Context, client *zfs.Client, snapshotName string) error {
	if !strings.Contains(snapshotName, "@") {
		return fmt.Errorf("snapshot name must be in format dataset@snapshot")
	}

	fmt.Printf("WARNING: You are about to DESTROY snapshot '%s'\n", snapshotName)
	fmt.Println("This operation is IRREVERSIBLE!")
	fmt.Print("Type 'DELETE' to confirm: ")
	
	var confirmation string
	fmt.Scanln(&confirmation)
	
	if confirmation != "DELETE" {
		return fmt.Errorf("operation cancelled - confirmation not provided")
	}

	fmt.Printf("Destroying snapshot: %s\n", snapshotName)
	return client.DestroySnapshot(ctx, snapshotName)
}

func rollbackToSnapshot(ctx context.Context, client *zfs.Client, datasetName, snapshotName string, force bool) error {
	fmt.Printf("WARNING: You are about to ROLLBACK dataset '%s' to snapshot '%s'\n", datasetName, snapshotName)
	fmt.Println("This will DESTROY all data created after the snapshot!")
	fmt.Println("This operation is IRREVERSIBLE!")
	if force {
		fmt.Println("FORCE MODE: Will destroy conflicting snapshots!")
	}
	fmt.Print("Type 'ROLLBACK' to confirm: ")
	
	var confirmation string
	fmt.Scanln(&confirmation)
	
	if confirmation != "ROLLBACK" {
		return fmt.Errorf("operation cancelled - confirmation not provided")
	}

	fmt.Printf("Rolling back dataset %s to snapshot %s (force: %v)\n", datasetName, snapshotName, force)
	return client.RollbackToSnapshot(ctx, datasetName, snapshotName, force)
}

func listSnapshots(ctx context.Context, client *zfs.Client, parent string) error {
	fmt.Printf("Listing snapshots")
	if parent != "" {
		fmt.Printf(" for parent dataset: %s", parent)
	}
	fmt.Println()

	snapshots, err := client.ListSnapshots(ctx, parent)
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		fmt.Println("No snapshots found")
		return nil
	}

	fmt.Printf("\nFound %d snapshots:\n", len(snapshots))
	fmt.Printf("====================\n")

	// Group snapshots by parent dataset
	parentGroups := make(map[string][]zfs.Snapshot)
	for _, snap := range snapshots {
		parentGroups[snap.Parent] = append(parentGroups[snap.Parent], snap)
	}

	for parentName, snapList := range parentGroups {
		fmt.Printf("\nParent: %s\n", parentName)
		fmt.Printf("%-30s %-20s %s\n", "Snapshot Name", "Type", "GUID")
		fmt.Printf("%-30s %-20s %s\n", strings.Repeat("-", 30), strings.Repeat("-", 20), strings.Repeat("-", 10))
		
		for _, snap := range snapList {
			snapName := snap.Name
			if atIndex := strings.LastIndex(snapName, "@"); atIndex != -1 {
				snapName = snapName[atIndex+1:] // Show just the snapshot part
			}
			fmt.Printf("%-30s %-20s %d\n", snapName, snap.Type, snap.GUID)
		}
	}

	// Show snapshot properties for first snapshot if any exist
	if len(snapshots) > 0 {
		firstSnap := snapshots[0]
		fmt.Printf("\nSample properties for snapshot %s:\n", firstSnap.Name)
		fmt.Printf("=====================================\n")
		
		props, err := client.GetProperties(ctx, firstSnap.Name, "used", "creation", "referenced")
		if err == nil && len(props) > 0 {
			for name, prop := range props {
				fmt.Printf("%-15s: %v (source: %s)\n", name, prop.Value, prop.Source)
			}
		}
	}

	return nil
}
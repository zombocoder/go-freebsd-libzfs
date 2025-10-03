package main

import (
	"context"
	"fmt"
	"log"

	"github.com/zombocoder/go-freebsd-libzfs/zfs"
)

func main() {
	ctx := context.Background()

	// Create ZFS client
	client, err := zfs.New(ctx)
	if err != nil {
		log.Fatalf("Failed to create ZFS client: %v", err)
	}
	defer client.Close()

	// Example usage of clone operations
	fmt.Println("=== ZFS Clone Operations Example ===")

	// Get runtime info
	impl, zfsVer, kernel, err := client.RuntimeInfo(ctx)
	if err != nil {
		log.Fatalf("Failed to get runtime info: %v", err)
	}
	fmt.Printf("Implementation: %s, ZFS Version: %s, Kernel: %s\n\n", impl, zfsVer, kernel)

	// Example 1: Create a clone from a snapshot
	snapshotName := "tank/test@snap1"
	cloneName := "tank/test-clone"

	fmt.Printf("Creating clone '%s' from snapshot '%s'\n", cloneName, snapshotName)
	err = client.CreateClone(ctx, snapshotName, cloneName, map[string]string{
		"mountpoint": "/mnt/test-clone",
		"readonly":   "off",
	})
	if err != nil {
		fmt.Printf("Note: Clone creation failed (expected if snapshot doesn't exist): %v\n", err)
	} else {
		fmt.Printf("✓ Clone created successfully\n")
	}

	// Example 2: Check if a dataset is a clone
	testDataset := "tank/test"
	isClone, err := client.IsClone(ctx, testDataset)
	if err != nil {
		fmt.Printf("Note: Could not check if '%s' is a clone: %v\n", testDataset, err)
	} else {
		fmt.Printf("Dataset '%s' is clone: %t\n", testDataset, isClone)
	}

	// Example 3: Get clone information
	fmt.Printf("\nGetting clone information for '%s'\n", testDataset)
	cloneInfo, err := client.GetCloneInfo(ctx, testDataset)
	if err != nil {
		fmt.Printf("Note: Could not get clone info: %v\n", err)
	} else {
		fmt.Printf("Clone Info:\n")
		fmt.Printf("  Name: %s\n", cloneInfo.Name)
		fmt.Printf("  Type: %s\n", cloneInfo.Type)
		fmt.Printf("  GUID: %d\n", cloneInfo.GUID)
		fmt.Printf("  Is Clone: %t\n", cloneInfo.IsClone)
		if cloneInfo.IsClone {
			fmt.Printf("  Origin: %s\n", cloneInfo.Origin)
		}
		fmt.Printf("  Clone Count: %d\n", cloneInfo.CloneCount)
		if len(cloneInfo.Dependents) > 0 {
			fmt.Printf("  Dependents: %v\n", cloneInfo.Dependents)
		}
	}

	// Example 4: List clones of a snapshot (if any)
	snapshotToCheck := "tank/test@snap1"
	fmt.Printf("\nListing clones of snapshot '%s'\n", snapshotToCheck)
	clones, err := client.ListClones(ctx, snapshotToCheck)
	if err != nil {
		fmt.Printf("Note: Could not list clones: %v\n", err)
	} else {
		if len(clones) == 0 {
			fmt.Printf("No clones found for snapshot '%s'\n", snapshotToCheck)
		} else {
			fmt.Printf("Found %d clone(s):\n", len(clones))
			for _, clone := range clones {
				fmt.Printf("  - %s\n", clone)
			}
		}
	}

	// Example 5: Promote a clone (if it exists)
	if isClone {
		fmt.Printf("\nPromoting clone '%s'\n", testDataset)
		err = client.PromoteClone(ctx, testDataset)
		if err != nil {
			fmt.Printf("Note: Clone promotion failed: %v\n", err)
		} else {
			fmt.Printf("✓ Clone promoted successfully\n")
		}
	}

	// Example 6: Demonstrate safe clone destruction
	cloneToDestroy := "tank/test-clone"
	fmt.Printf("\nDestroying clone '%s' (if it exists)\n", cloneToDestroy)
	err = client.DestroyClone(ctx, cloneToDestroy, false)
	if err != nil {
		fmt.Printf("Note: Clone destruction failed (expected if clone doesn't exist): %v\n", err)
	} else {
		fmt.Printf("✓ Clone destroyed successfully\n")
	}

	fmt.Println("\n=== Clone Operations Complete ===")
	fmt.Println("\nAvailable clone operations:")
	fmt.Println("  • CreateClone(snapshot, clone, properties)")
	fmt.Println("  • PromoteClone(clone)")
	fmt.Println("  • GetCloneInfo(dataset)")
	fmt.Println("  • ListClones(snapshot)")
	fmt.Println("  • DestroyClone(clone, force)")
	fmt.Println("  • IsClone(dataset)")
	fmt.Println("  • GetCloneOrigin(clone)")
}
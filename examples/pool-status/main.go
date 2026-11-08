//go:build freebsd

// Command pool-status prints pool health in the style of `zpool status`:
// the status condition, scrub/resilver progress, the vdev tree with error
// counters, spares, cache devices and data errors.
//
// Usage:
//
//	pool-status            # all pools
//	pool-status <pool>...  # selected pools
//
// Exits with status 1 if any reported pool is not healthy, so it can be used
// from monitoring scripts.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/zombocoder/go-freebsd-libzfs/zpool"
)

func main() {
	ctx := context.Background()

	client, err := zpool.New(ctx)
	if err != nil {
		log.Fatalf("Failed to create pool client: %v", err)
	}
	defer client.Close()

	names := os.Args[1:]
	if len(names) == 0 {
		pools, err := client.List(ctx)
		if err != nil {
			log.Fatalf("Failed to list pools: %v", err)
		}
		for _, p := range pools {
			names = append(names, p.Name)
		}
	}
	if len(names) == 0 {
		fmt.Println("no pools available")
		return
	}

	unhealthy := false
	for i, name := range names {
		if i > 0 {
			fmt.Println()
		}
		st, err := client.GetStatus(ctx, name)
		if err != nil {
			log.Printf("%s: %v", name, err)
			unhealthy = true
			continue
		}
		printStatus(st)
		if st.Pool.Health != zpool.HealthOnline || st.Problem != "OK" || st.Errors.DataErrors > 0 {
			unhealthy = true
		}
	}
	if unhealthy {
		os.Exit(1)
	}
}

func printStatus(st *zpool.Status) {
	fmt.Printf("  pool: %s\n", st.Pool.Name)
	fmt.Printf(" state: %s\n", st.Pool.Health)
	if st.Problem != "OK" {
		fmt.Printf("status: %s\n", st.Problem)
	}
	if st.MsgID != "" {
		fmt.Printf("   see: https://openzfs.github.io/openzfs-docs/msg/%s\n", st.MsgID)
	}
	fmt.Printf("  scan: %s\n", describeScan(st.Scan))

	fmt.Println("config:")
	fmt.Println()
	fmt.Printf("\t%-40s %-8s %5s %5s %5s\n", "NAME", "STATE", "READ", "WRITE", "CKSUM")
	printVdev(st.Config, 0)

	if len(st.Cache) > 0 {
		fmt.Println("\tcache")
		for _, v := range st.Cache {
			printVdev(v, 1)
		}
	}
	if len(st.Spares) > 0 {
		fmt.Println("\tspares")
		for _, v := range st.Spares {
			fmt.Printf("\t  %-38s %-8s\n", v.Name, v.State)
		}
	}
	fmt.Println()

	if st.Errors.DataErrors == 0 {
		fmt.Println("errors: No known data errors")
	} else {
		fmt.Printf("errors: %d data errors, use 'zpool status -v' for a list\n", st.Errors.DataErrors)
	}
}

func printVdev(v zpool.VDevTree, depth int) {
	name := strings.Repeat("  ", depth) + v.Name
	fmt.Printf("\t%-40s %-8s %5d %5d %5d\n", name, v.State,
		v.Stats.ReadErrors, v.Stats.WriteErrors, v.Stats.ChecksumErrors)
	for _, c := range v.Children {
		printVdev(c, depth+1)
	}
}

func describeScan(s *zpool.ScanStatus) string {
	if s == nil {
		return "none requested"
	}
	start := s.StartTime.Format(time.RFC1123)
	switch s.State {
	case zpool.ScanStateScanning, zpool.ScanStateSuspended:
		pct := 0.0
		if s.ToExamine > 0 {
			pct = 100 * float64(s.Issued) / float64(s.ToExamine)
		}
		verb := "in progress"
		if s.State == zpool.ScanStateSuspended {
			verb = "paused"
		}
		msg := fmt.Sprintf("%s %s since %s\n\t%s scanned, %s issued of %s, %.2f%% done",
			s.Function, verb, start, bytes(s.Examined), bytes(s.Issued), bytes(s.ToExamine), pct)
		if s.Rate > 0 {
			msg += fmt.Sprintf(" at %s/s", bytes(s.Rate))
		}
		return msg
	case zpool.ScanStateFinished:
		took := s.EndTime.Sub(s.StartTime).Round(time.Second)
		action := "repaired"
		if s.Function == zpool.ScanFunctionResilver {
			action = "resilvered"
		}
		return fmt.Sprintf("%s %s %s in %s with %d errors on %s",
			s.Function, action, bytes(s.Processed), took, s.Errors, s.EndTime.Format(time.RFC1123))
	case zpool.ScanStateCanceled:
		return fmt.Sprintf("%s canceled on %s", s.Function, s.EndTime.Format(time.RFC1123))
	default:
		return "none requested"
	}
}

// bytes formats a byte count with a binary unit suffix.
func bytes(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := uint64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", float64(n)/float64(div), "KMGTPE"[exp])
}

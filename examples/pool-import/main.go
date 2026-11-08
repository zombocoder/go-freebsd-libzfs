//go:build freebsd

// Command pool-import discovers and imports pools, like `zpool import`.
//
// Usage:
//
//	pool-import [-d dir]... [-D]                     # list importable pools
//	pool-import [-d dir]... [-D] [-f] [-R altroot] <pool> [newname]
//
// Flags:
//
//	-d dir   search dir for pool devices (repeatable; default /dev)
//	-D       list or import destroyed pools
//	-f       force import of a pool last used by another system
//	-R dir   import with an alternate root
//
// Importing requires root.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/zombocoder/go-freebsd-libzfs/zpool"
)

// dirList collects repeated -d flags.
type dirList []string

func (d *dirList) String() string     { return strings.Join(*d, ",") }
func (d *dirList) Set(v string) error { *d = append(*d, v); return nil }

func main() {
	var dirs dirList
	flag.Var(&dirs, "d", "directory to search for pool devices (repeatable)")
	destroyed := flag.Bool("D", false, "list or import destroyed pools")
	force := flag.Bool("f", false, "force import of a pool last used by another system")
	altRoot := flag.String("R", "", "alternate root for the imported pool")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] [pool [newname]]\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Without a pool name, lists pools available for import.")
		flag.PrintDefaults()
	}
	flag.Parse()

	ctx := context.Background()
	client, err := zpool.New(ctx)
	if err != nil {
		log.Fatalf("Failed to create pool client: %v", err)
	}
	defer client.Close()

	switch flag.NArg() {
	case 0:
		listImportable(ctx, client, dirs, *destroyed)
	case 1, 2:
		opts := zpool.ImportOptions{
			SearchPaths: dirs,
			Destroyed:   *destroyed,
			Force:       *force,
			AltRoot:     *altRoot,
			NewName:     flag.Arg(1),
		}
		if err := client.Import(ctx, flag.Arg(0), opts); err != nil {
			log.Fatal(err)
		}
		name := flag.Arg(0)
		if opts.NewName != "" {
			name = opts.NewName
		}
		fmt.Printf("pool %s imported\n", name)
	default:
		flag.Usage()
		os.Exit(2)
	}
}

func listImportable(ctx context.Context, client *zpool.Client, dirs []string, destroyed bool) {
	pools, err := client.Discover(ctx, zpool.DiscoverOptions{
		SearchPaths:      dirs,
		IncludeDestroyed: destroyed,
	})
	if err != nil {
		log.Fatal(err)
	}
	if len(pools) == 0 {
		fmt.Println("no pools available to import")
		return
	}
	for i, p := range pools {
		if i > 0 {
			fmt.Println()
		}
		fmt.Printf("   pool: %s\n", p.Name)
		fmt.Printf("     id: %d\n", p.GUID)
		fmt.Printf("  state: %s (%s)\n", p.Health, p.State)
		fmt.Println(" config:")
		for _, dev := range p.DevicePaths {
			fmt.Printf("\t%s\n", dev)
		}
	}
}

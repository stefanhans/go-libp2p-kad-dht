package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-discovery"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/opts"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
)

var (
	name   string
	prompt string

	err         error
	ctx         = context.Background()
	node        host.Host
	kademliaDHT *dht.IpfsDHT

	bootstrapPeerAddrs []multiaddr.Multiaddr

	rendezvousString string
	rendezvousPoint  cid.Cid
	v1b              = cid.V1Builder{Codec: cid.Raw, MhType: multihash.SHA2_256}

	routingDiscovery *discovery.RoutingDiscovery
)

// IPFS bootstrap nodes. Used to find other peers in the network.
var bootstrapPeers = []string{
	"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	"/ip4/104.236.179.241/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
	"/ip4/104.236.76.40/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
	"/ip4/128.199.219.111/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
	"/ip4/178.62.158.247/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
}

func main() {

	debug := flag.Bool("debug", false, "Switch on debugging")

	// log is the file to write logger output
	debugfilename := flag.String("debugfile", "", "file to write debugging output")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "missing or wrong parameter: <name>")
		os.Exit(1)
	}

	name = flag.Arg(0)
	prompt = "<" + name + "> "

	// "-debug" writes debugging to file
	if *debug {
		err = startDebugging((*debugfilename))
		if err != nil {
			panic(err)
		}
	}

	// Start logging into one file each session
	logfile, err = startLogging(name)
	if err != nil {
		log.Fatalf("error starting logging: %v", err)
	}
	defer logfile.Close()

	// Start logging into one common file for all sessions of today
	//cLog, commonLogfile, err = startCommonLogging(name, "C24DFD14-E8AF-44B7-8619-B552E58B4673")
	//if err != nil {
	//	log.Fatalf("error starting logging: %v", err)
	//}
	//defer commonLogfile.Close()

	// libp2p.New constructs a new libp2p Host. Other options can be added
	// here.
	node, err = libp2p.New(ctx, libp2p.DisableRelay())
	//libp2p.Identity(privRedId),
	//libp2p.ListenAddrs([]multiaddr.Multiaddr(config.ListenAddresses)...),

	if err != nil {
		panic(err)
	}

	//fmt.Printf("node.ID: %v\n", node.ID())
	//for i, addr := range node.Addrs() {
	//	fmt.Printf("%d: %v\n", i, addr)
	//}
	//bootstrapPeerAddrs, err = StringsToAddrs(bootstrapPeers)
	//if err != nil {
	//	panic(err)
	//}

	// Start a DHT, for use in peer discovery. We can't just make a new DHT
	// client because we want each peer to maintain its own local copy of the
	// DHT, so that the bootstrapping node of the DHT can go down without
	// inhibiting future peer discovery.
	kademliaDHT, err = dht.New(ctx, node, dhtopts.Validator(NullValidator{}))
	if err != nil {
		panic(err)
	}

	// Initialize chat command usage
	commandsInit()

	fmt.Fprint(os.Stdin, prompt)

	bio := bufio.NewReader(os.Stdin)

	go func() {
		for {
			line, hasPrefix, err := bio.ReadLine()
			if err != nil {
				panic(err)
			}
			writeChan <- fmt.Sprintf("%s", line)
			if hasPrefix {
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case str := <-writeChan:
				executeCommand(str)

				//fmt.Printf("%s", str)

			}
		}
	}()

	select {}
}
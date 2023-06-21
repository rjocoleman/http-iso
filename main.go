package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/kdomanski/iso9660"
)

type stringSlice []string

func (i *stringSlice) String() string {
	return fmt.Sprintf("%v", *i)
}

func (i *stringSlice) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var (
	isoImage     *iso9660.Image
	kernelFile   string
	initrdFiles  stringSlice
	kernelParams string
)

func ipxeHandler(w http.ResponseWriter, req *http.Request) {
	if kernelFile == "" || len(initrdFiles) == 0 {
		http.NotFound(w, req)
		return
	}

	fmt.Fprintf(w, "#!ipxe\nkernel http://%s%s %s\n", req.Host, kernelFile, kernelParams)
	for _, initrd := range initrdFiles {
		parts := strings.Split(initrd, ",")
		if len(parts) == 1 {
			fmt.Fprintf(w, "initrd http://%s%s\n", req.Host, parts[0])
		} else {
			fmt.Fprintf(w, "initrd http://%s%s %s\n", req.Host, parts[0], parts[1])
		}
	}
	fmt.Fprint(w, "boot")
}

func isoHandler(w http.ResponseWriter, req *http.Request) {
	reqPath := req.URL.Path
	currentFile, err := isoImage.RootDir()
	if err != nil {
		log.Fatalf("Failed to get root directory: %v", err)
	}
	parts := strings.Split(reqPath, "/")[1:]

	for _, part := range parts {
		if part == "" {
			continue
		}

		children, err := currentFile.GetChildren()
		if err != nil {
			http.NotFound(w, req)
			return
		}

		found := false
		for _, child := range children {
			if child.Name() == part {
				currentFile = child
				found = true
				break
			}
		}

		if !found {
			http.NotFound(w, req)
			return
		}
	}

	if currentFile.IsDir() {
		children, err := currentFile.GetChildren()
		if err != nil {
			http.Error(w, "Failed to get directory children", http.StatusInternalServerError)
			return
		}

		// Generate and write an HTML directory listing
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<html><body><ul>")
		for _, child := range children {
			fmt.Fprintf(w, "<li><a href=\"%s\">%s</a></li>", path.Join("/", reqPath, child.Name()), child.Name())
		}
		fmt.Fprint(w, "</ul></body></html>")
		return
	}

	io.Copy(w, currentFile.Reader())
}

func getLocalIPs() ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	var ips []string
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() || ipNet.IP.To4() == nil {
			continue
		}
		ips = append(ips, ipNet.IP.String())
	}

	return ips, nil
}

func main() {
	// Define command line arguments
	isoPath := flag.String("iso", "", "Path to the ISO file")
	kernelPath := flag.String("kernel", "", "Path to the kernel file relative to the ISO root")
	params := flag.String("params", "", "Parameters to pass to the kernel at boot")
	port := flag.String("port", "8080", "Port number to listen on")
	flag.Var(&initrdFiles, "initrd", "Comma-separated pairs of initrd files and their names in the iPXE script")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  --iso string       Path to the ISO file\n")
		fmt.Fprintf(os.Stderr, "  --kernel string    Path to the kernel file relative to the ISO root\n")
		fmt.Fprintf(os.Stderr, "  --initrd string[]  Comma-separated pairs of initrd files and their names in the iPXE script\n")
		fmt.Fprintf(os.Stderr, "  --params string    Parameters to pass to the kernel at boot\n")
		fmt.Fprintf(os.Stderr, "  --port string      Port number to listen on (default: 8080)\n")
	}
	flag.Parse()

	// Show usage if no arguments were provided
	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(1)
	}

	// Check if ISO path was provided
	if *isoPath == "" {
		log.Fatal("Please provide the path to an ISO file.")
	}

	// Update global variables with user-provided values
	kernelFile = *kernelPath
	kernelParams = *params

	// Open the ISO image
	file, err := os.Open(*isoPath)
	if err != nil {
		log.Fatalf("Failed to open ISO file: %v", err)
	}
	defer file.Close()

	isoImage, err = iso9660.OpenImage(file)
	if err != nil {
		log.Fatalf("Failed to read ISO image: %v", err)
	}

	// Get local IPs
	ips, err := getLocalIPs()
	if err != nil {
		log.Fatalf("Failed to get local IPs: %v", err)
	}

	// Print server addresses
	for _, ip := range ips {
		fmt.Printf("Serving on http://%s:%s\n", ip, *port)
		fmt.Printf("To boot from iPXE: chain --autofree http://%s:%s/boot.ipxe\n", ip, *port)
	}

	// Start the HTTP server
	http.HandleFunc("/boot.ipxe", ipxeHandler)
	http.HandleFunc("/", isoHandler)
	err = http.ListenAndServe(":"+*port, nil)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

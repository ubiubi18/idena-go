package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	files "github.com/ipfs/go-ipfs-files"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	ifaceopts "github.com/ipfs/interface-go-ipfs-core/options"
	ifacepath "github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo/fsrepo"
)

type pinRow struct {
	CID       string `json:"cid"`
	Type      string `json:"type"`
	FileBytes int64  `json:"fileBytes,omitempty"`
	BlockSize int    `json:"blockSize,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Preview   string `json:"preview,omitempty"`
	Export    string `json:"export,omitempty"`
	Error     string `json:"error,omitempty"`
}

type blockRow struct {
	CID  string `json:"cid"`
	Size int    `json:"size,omitempty"`
}

type report struct {
	RepoPath          string         `json:"repoPath"`
	PinCount          int            `json:"pinCount"`
	PinTypes          map[string]int `json:"pinTypes"`
	Pins              []pinRow       `json:"pins"`
	BlockCount        int            `json:"blockCount"`
	BlockBytes        int64          `json:"blockBytes"`
	BlockSizeErrors   int            `json:"blockSizeErrors"`
	BlockSamples      []blockRow     `json:"blockSamples"`
	TruncatedPins     bool           `json:"truncatedPins"`
	TruncatedBlocks   bool           `json:"truncatedBlocks"`
	InspectionSeconds float64        `json:"inspectionSeconds"`
}

func defaultRepoPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Library", "Application Support", "IdenaAI", "node", "datadir", "ipfs")
}

func unixfsFileInfo(ctx context.Context, api coreiface.CoreAPI, cid string, previewBytes int64) (int64, string, string, error) {
	nodeFile, err := api.Unixfs().Get(ctx, ifacepath.New("/ipfs/"+cid))
	if err != nil {
		return 0, "", "", err
	}
	file := files.ToFile(nodeFile)
	defer file.Close()
	size, err := file.Size()
	if err != nil {
		return 0, "", "", err
	}
	preview, _ := io.ReadAll(io.LimitReader(file, previewBytes))
	kind, textPreview := detectContent(preview)
	return size, kind, textPreview, nil
}

func isMostlyPrintable(text string) bool {
	if text == "" {
		return true
	}
	printable := 0
	total := 0
	for _, r := range text {
		total++
		if r >= 32 || r == '\n' || r == '\r' || r == '\t' {
			printable++
		}
	}
	return total > 0 && float64(printable)/float64(total) >= 0.92
}

func detectContent(data []byte) (string, string) {
	switch {
	case len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n":
		return "image/png", ""
	case len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff:
		return "image/jpeg", ""
	case len(data) >= 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a"):
		return "image/gif", ""
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return "image/webp", ""
	case !utf8.Valid(data):
		return "binary", fmt.Sprintf("%x", data)
	}

	text := string(data)
	if !isMostlyPrintable(text) {
		return "binary", fmt.Sprintf("%x", data)
	}
	trimmed := strings.TrimSpace(text)
	switch {
	case strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "["):
		return "json/text", trimmed
	case strings.HasPrefix(strings.ToLower(trimmed), "<svg"):
		return "image/svg+xml", trimmed
	case strings.HasPrefix(trimmed, "data:"):
		header := trimmed
		if comma := strings.Index(header, ","); comma >= 0 {
			header = header[:comma]
		}
		return "data-url", header
	default:
		return "text/plain", trimmed
	}
}

func extensionForKind(kind string) string {
	switch kind {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "text/plain":
		return ".txt"
	case "json/text":
		return ".json"
	case "data-url":
		return ".txt"
	default:
		return ".bin"
	}
}

func sanitizeKind(kind string) string {
	if kind == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")
	return replacer.Replace(kind)
}

func exportUnixfsFile(ctx context.Context, api coreiface.CoreAPI, cid string, kind string, exportDir string) (string, error) {
	if exportDir == "" {
		return "", nil
	}
	nodeFile, err := api.Unixfs().Get(ctx, ifacepath.New("/ipfs/"+cid))
	if err != nil {
		return "", err
	}
	file := files.ToFile(nodeFile)
	defer file.Close()

	targetDir := filepath.Join(exportDir, sanitizeKind(kind))
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}
	targetPath := filepath.Join(targetDir, cid+extensionForKind(kind))
	out, err := os.Create(targetPath)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		return "", err
	}
	return targetPath, nil
}

func splitCSV(value string) map[string]bool {
	result := map[string]bool{}
	for _, part := range strings.Split(value, ",") {
		text := strings.TrimSpace(part)
		if text != "" {
			result[text] = true
		}
	}
	return result
}

func main() {
	repoPath := flag.String("repo", defaultRepoPath(), "IPFS repo path")
	limit := flag.Int("limit", 100, "maximum pins and block samples to print")
	previewBytes := flag.Int64("preview-bytes", 160, "bytes to read from each shown UnixFS pin for content detection")
	exportDir := flag.String("export-dir", "", "export matching pinned UnixFS roots to this directory")
	exportKinds := flag.String("export-kinds", "", "comma-separated kind filter for export, for example image/jpeg,text/plain")
	exportCids := flag.String("export-cids", "", "comma-separated CID filter for export")
	exportLimit := flag.Int("export-limit", 0, "maximum files to export; 0 means no export limit")
	includeIndirect := flag.Bool("include-indirect", false, "include indirect pins in export")
	timeout := flag.Duration("timeout", 2*time.Minute, "inspection timeout")
	jsonOutput := flag.Bool("json", false, "print JSON")
	flag.Parse()

	if *repoPath == "" {
		fmt.Fprintln(os.Stderr, "repo path is empty")
		os.Exit(1)
	}
	if !fsrepo.IsInitialized(*repoPath) {
		fmt.Fprintf(os.Stderr, "IPFS repo is not initialized: %s\n", *repoPath)
		os.Exit(1)
	}

	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	plugins, err := loader.NewPluginLoader("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "load IPFS plugins: %v\n", err)
		os.Exit(1)
	}
	if err := plugins.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "initialize IPFS plugins: %v\n", err)
		os.Exit(1)
	}
	if err := plugins.Inject(); err != nil {
		fmt.Fprintf(os.Stderr, "inject IPFS plugins: %v\n", err)
		os.Exit(1)
	}

	repo, err := fsrepo.Open(*repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open repo: %v\n", err)
		os.Exit(1)
	}
	defer repo.Close()

	node, err := core.NewNode(ctx, &core.BuildCfg{Repo: repo, Online: false})
	if err != nil {
		fmt.Fprintf(os.Stderr, "open offline IPFS node: %v\n", err)
		os.Exit(1)
	}
	defer node.Close()

	api, err := coreapi.NewCoreAPI(node)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create core API: %v\n", err)
		os.Exit(1)
	}

	out := report{
		RepoPath: *repoPath,
		PinTypes: map[string]int{},
	}
	kindFilter := splitCSV(*exportKinds)
	cidFilter := splitCSV(*exportCids)
	exported := 0

	pinCh, err := api.Pin().Ls(ctx, ifaceopts.Pin.Ls.All())
	if err != nil {
		fmt.Fprintf(os.Stderr, "list pins: %v\n", err)
		os.Exit(1)
	}
	for pin := range pinCh {
		out.PinCount++
		if pin.Err() != nil {
			if len(out.Pins) < *limit {
				out.Pins = append(out.Pins, pinRow{Error: pin.Err().Error()})
			}
			continue
		}
		cid := pin.Path().Cid().String()
		pinType := pin.Type()
		out.PinTypes[pinType]++

		row := pinRow{CID: cid, Type: pinType}
		if size, err := node.Blocks.Blockstore().GetSize(ctx, pin.Path().Cid()); err == nil {
			row.BlockSize = size
		}
		if fileSize, kind, preview, err := unixfsFileInfo(ctx, api, cid, *previewBytes); err == nil {
			row.FileBytes = fileSize
			row.Kind = kind
			row.Preview = preview
			shouldExport := *exportDir != "" &&
				(*includeIndirect || pinType != "indirect" || len(cidFilter) > 0) &&
				(len(cidFilter) == 0 || cidFilter[cid]) &&
				(len(kindFilter) == 0 || kindFilter[kind]) &&
				(*exportLimit <= 0 || exported < *exportLimit)
			if shouldExport {
				if exportPath, err := exportUnixfsFile(ctx, api, cid, kind, *exportDir); err == nil {
					row.Export = exportPath
					exported++
				} else {
					row.Error = err.Error()
				}
			}
		} else {
			row.Error = err.Error()
		}
		if len(out.Pins) < *limit {
			out.Pins = append(out.Pins, row)
		} else {
			out.TruncatedPins = true
		}
	}

	keyCh, err := node.Blocks.Blockstore().AllKeysChan(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "list blockstore keys: %v\n", err)
		os.Exit(1)
	}
	for cid := range keyCh {
		out.BlockCount++
		size, err := node.Blocks.Blockstore().GetSize(ctx, cid)
		if err != nil {
			out.BlockSizeErrors++
		} else {
			out.BlockBytes += int64(size)
		}
		if len(out.BlockSamples) < *limit {
			out.BlockSamples = append(out.BlockSamples, blockRow{CID: cid.String(), Size: size})
		} else {
			out.TruncatedBlocks = true
		}
	}

	sort.Slice(out.Pins, func(i, j int) bool {
		if out.Pins[i].Type != out.Pins[j].Type {
			return out.Pins[i].Type < out.Pins[j].Type
		}
		return out.Pins[i].CID < out.Pins[j].CID
	})
	sort.Slice(out.BlockSamples, func(i, j int) bool {
		return out.BlockSamples[i].CID < out.BlockSamples[j].CID
	})
	out.InspectionSeconds = time.Since(started).Seconds()

	if *jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(out); err != nil {
			fmt.Fprintf(os.Stderr, "write JSON: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Printf("Repo: %s\n", out.RepoPath)
	fmt.Printf("Pins: %d", out.PinCount)
	if len(out.PinTypes) > 0 {
		fmt.Print(" (")
		i := 0
		for pinType, count := range out.PinTypes {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%s=%d", pinType, count)
			i++
		}
		fmt.Print(")")
	}
	fmt.Println()
	fmt.Printf("Blocks: %d\n", out.BlockCount)
	fmt.Printf("Block bytes: %d\n", out.BlockBytes)
	fmt.Printf("Inspection seconds: %.2f\n", out.InspectionSeconds)

	if len(out.Pins) > 0 {
		fmt.Println("\nPins:")
		fmt.Printf("%-48s %-10s %12s %12s %-14s %s\n", "CID", "TYPE", "FILE_BYTES", "BLOCK_SIZE", "KIND", "PREVIEW/ERROR")
		for _, pin := range out.Pins {
			detail := pin.Preview
			if detail == "" {
				detail = pin.Error
			}
			detail = strings.ReplaceAll(detail, "\n", "\\n")
			if len(detail) > 80 {
				detail = detail[:80] + "..."
			}
			fmt.Printf("%-48s %-10s %12d %12d %-14s %s\n", pin.CID, pin.Type, pin.FileBytes, pin.BlockSize, pin.Kind, detail)
		}
		if out.TruncatedPins {
			fmt.Printf("... more pins omitted by --limit=%d\n", *limit)
		}
	}

	if len(out.BlockSamples) > 0 {
		fmt.Println("\nBlock samples:")
		fmt.Printf("%-64s %12s\n", "CID", "SIZE")
		for _, block := range out.BlockSamples {
			fmt.Printf("%-64s %12d\n", block.CID, block.Size)
		}
		if out.TruncatedBlocks {
			fmt.Printf("... more block CIDs omitted by --limit=%d\n", *limit)
		}
	}
}

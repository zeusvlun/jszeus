package main

import (
    "bytes"
    "fmt"
    "html"
    "io"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "sync"
    "golang.org/x/net/html"
    "github.com/fatih/color"
    "gopkg.in/pterm.v2"
)

func main() {
    targetURL := "https://example.com"
    outputDir := "js_files"

    // Fetch and parse HTML
    doc, err := fetchAndParseHTML(targetURL)
    if err != nil {
        log.Fatal(err)
    }

    // Find all <script> tags and extract src attributes
    var scripts []string
    findScripts(doc, &scripts)

    // Download JS files with progress bar
    var wg sync.WaitGroup
    sem := make(chan struct{}, 5) // Semaphore for concurrency control
    pterm.DefaultProgress.Start(len(scripts))
    for _, scriptURL := range scripts {
        wg.Add(1)
        go func(url string) {
            defer wg.Done()
            sem <- struct{}{} // Acquire semaphore
            downloadFile(url, outputDir)
            <-sem // Release semaphore
            pterm.DefaultProgress.Increment()
        }(scriptURL)
    }
    wg.Wait()
    pterm.DefaultProgress.Stop()
}

func fetchAndParseHTML(url string) (*html.Node, error) {
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    doc, err := html.Parse(resp.Body)
    if err != nil {
        return nil, err
    }
    return doc, nil
}

func findScripts(node *html.Node, scripts *[]string) {
    if node.Type == html.ElementNode && node.Data == "script" {
        src := getAttribute(node, "src")
        if src != "" {
            *scripts = append(*scripts, src)
        }
    }
    for child := node.FirstChild; child != nil; child = child.NextSibling {
        findScripts(child, scripts)
    }
}

func getAttribute(node *html.Node, attr string) string {
    for _, a := range node.Attr {
        if a.Key == attr {
            return a.Val
        }
    }
    return ""
}

func downloadFile(url string, outputDir string) {
    resp, err := http.Get(url)
    if err != nil {
        color.Red.Printf("Error downloading %s: %v\n", url, err)
        return
    }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        color.Yellow.Printf("Status %d for %s\n", resp.StatusCode, url)
        return
    }
    filename := filepath.Base(url)
    filePath := filepath.Join(outputDir, filename)
    os.MkdirAll(outputDir, os.ModePerm)
    file, err := os.Create(filePath)
    if err != nil {
        color.Red.Printf("Error creating %s: %v\n", filePath, err)
        return
    }
    defer file.Close()
    _, err = io.Copy(file, resp.Body)
    if err != nil {
        color.Red.Printf("Error writing %s: %v\n", filePath, err)
        return
    }
    color.Green.Printf("Downloaded %s to %s\n", url, filePath)
}

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

type meta struct {
	Status  int
	Headers map[string][]string
}

type page struct {
	URL      string
	Meta     meta
	Elements map[string]int
}

type parseChan struct {
	url chan string
	res chan page
	err chan error
}

var (
	//THREADS - the number of threads
	THREADS = flag.Int("tr", 1, "number of threads")
	//URLFILE - the pass to the list of URLs
	URLFILE = flag.String("URL", "input.txt", "the pass to the list of URLs")
	//WaitGroup
	wg sync.WaitGroup
	//Preparing channel
	parseURL = make(chan string)
	parseRes = make(chan page)
	parseErr = make(chan error)
	parseCh  = parseChan{
		url: parseURL,
		res: parseRes,
		err: parseErr,
	}
)

func readURLs() ([]string, error) {
	fmt.Println("Opening file with URLs...")
	urlFile, err := os.OpenFile(*URLFILE, os.O_RDONLY, 0666)
	if err != nil {
		return nil, err
	}
	defer urlFile.Close()
	fmt.Println("The file has been successfully opened")

	//counting the number of URLs in the file
	reader := bufio.NewReader(urlFile)
	urlNumber := 0
	for {
		str, err := reader.ReadString('\n')
		if (err == io.EOF && str == "") || str == "\n" {
			break
		} else {
			if err != nil && err != io.EOF {
				return nil, err
			}
		}
		urlNumber++
	}
	//setting the offset for the next read to the begining of the file
	if _, err := urlFile.Seek(0, 0); err != nil {
		return nil, err
	}
	urlList := make([]string, 0, urlNumber)
	fmt.Println("Reading URLs...")
	for {
		str, err := reader.ReadString('\n')
		if (err == io.EOF && str == "") || str == "\n" {
			break
		} else {
			if err != nil && err != io.EOF {
				return nil, err
			}
		}
		urlList = append(urlList, str)
	}
	fmt.Println("URLs have been successfully read")
	return urlList, nil
}

func urlParser() {
	for url := range parseCh.url {
		resp, err := http.Get(url)
		if err != nil {
			panic(err)
			// ...
		}
		defer resp.Body.Close()
		doc, err := html.Parse(resp.Body)
		if err != nil {
			panic(err)
			// ...
		}

		var pg page
		pg.Meta.Headers = make(map[string][]string)
		pg.Elements = make(map[string]int)

		pg.URL = url
		pg.Meta.Status = resp.StatusCode
		for k, val := range resp.Header {
			pg.Meta.Headers[k] = val
		}

		var f func(*html.Node)
		f = func(n *html.Node) {
			if n.Type == html.ElementNode {
				pg.Elements[n.Data] = pg.Elements[n.Data] + 1
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}
		f(doc)
		parseCh.res <- pg
	}
	wg.Done()
}

func collector() {

}

func main() {
	flag.Parse()
	urlList, err := readURLs()
	if err != nil {
		panic(err)
	}
	wg.Add(1)
	//Creating THREADS number of goroutines for parsing
	go func() {
		for i := 0; i < *THREADS; i++ {
			wg.Add(1)
			go urlParser()
		}
		wg.Done()
	}()
	//
	wg.Add(1)
	go func(urlList []string) {
		for _, str := range urlList {
			parseCh.url <- strings.TrimSuffix(str, "\n")
		}
		close(parseCh.url)
		wg.Done()
	}(urlList)
	wg.Wait()
}

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	var wg sync.WaitGroup
	urls := loadUrls()
	repos := []repoInfo{}
	for _, url := range urls {
		wg.Add(1)
		a := url
		go func() {
			defer wg.Done()
			repo := process(a)
			repos = append(repos, repo)
			fmt.Print(".")
		}()
	}
	wg.Wait()
	printSortedAlpha(repos)
	printSortedLastcommit(repos)
}

func printSortedAlpha(repos []repoInfo) {
	sort.Sort(reposByURL(repos))
	fmt.Print("\n\n")
	for _, r := range repos {
		fmt.Println(r.Markdown())
	}
}

func printSortedLastcommit(repos []repoInfo) {
	sort.Sort(reposByLastcommit(repos))
	fmt.Print("\n\n")
	for _, r := range repos {
		fmt.Println(r.MarkdownActivity())
	}
}

func process(url string) repoInfo {
	parser := repoParser{}
	doc := parser.getDoc(url)
	repo := repoInfo{
		url:         strings.ToLower(url),
		description: parser.getDescription(doc),
		lastcommit:  parser.getLastcommit(doc),
	}
	return repo
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

/*
	Repo Parser logic
*/
type repoParser struct{}

func (r repoParser) getDescription(doc *goquery.Document) string {
	var content string
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		if name == "description" {
			content, _ = s.Attr("content")
		}
	})
	return content
}

func (r repoParser) getLastcommit(doc *goquery.Document) string {
	if r.hasIncludedLastcommit(doc) {
		return r.getLastcommitIncluded(doc)
	}
	return r.getLastcommitAjax(doc)
}

func (r repoParser) hasIncludedLastcommit(doc *goquery.Document) bool {
	found := true
	doc.Find(".commit-loader").Each(func(i int, s *goquery.Selection) {
		found = false
	})
	return found
}

func (r repoParser) getLastcommitIncluded(doc *goquery.Document) string {
	var datetime string
	doc.Find(".commit-tease relative-time").Each(func(i int, s *goquery.Selection) {
		datetime, _ = s.Attr("datetime")
	})
	return datetime
}

func (r repoParser) getLastcommitAjax(doc *goquery.Document) string {
	// extract the ajax url
	// e.g.: <include-fragment class="commit-tease commit-loader" src="/f2prateek/coi/tree-commit/866dee22e2b11dd9780770c00bae53886d9b4863">
	s := doc.Find(".commit-loader")
	path, _ := s.Attr("src")
	url := "https://github.com" + path
	ajaxDoc := r.urlDoc(url)
	return r.getLastcommit(ajaxDoc)
}

func (r repoParser) getDoc(url string) *goquery.Document {
	return r.urlDoc(url)
	// return localDoc()
}

func (r repoParser) urlDoc(url string) *goquery.Document {
	doc, err := goquery.NewDocument(url)
	check(err)
	return doc
}

func (r repoParser) localDoc() *goquery.Document {
	filename := "crawler/fixture.html"
	file, err := os.Open(filename)
	check(err)
	doc, err := goquery.NewDocumentFromReader(file)
	check(err)
	return doc
}

/*
repoInfo logic
*/
type repoInfo struct {
	url         string
	description string
	lastcommit  string
}

func (ri repoInfo) Markdown() string {
	lastcommit := ri.lastcommit[0:10]
	shorturl := strings.Replace(ri.url, "https://github.com/", "", -1)
	return fmt.Sprintf("- [%s](%s) - %s <br/> ( %s )", shorturl, ri.url, ri.description, lastcommit)
}

func (ri repoInfo) MarkdownActivity() string {
	lastcommit := ri.lastcommit[0:10]
	shorturl := strings.Replace(ri.url, "https://github.com/", "", -1)
	link := fmt.Sprintf("[%s](%s)", shorturl, ri.url)
	return fmt.Sprintf("- %s - %s  <br/> %s ", lastcommit, link, ri.description)
}

type reposByLastcommit []repoInfo

func (ris reposByLastcommit) Len() int           { return len(ris) }
func (ris reposByLastcommit) Less(i, j int) bool { return ris[i].lastcommit > ris[j].lastcommit }
func (ris reposByLastcommit) Swap(i, j int)      { ris[i], ris[j] = ris[j], ris[i] }

type reposByURL []repoInfo

func (ris reposByURL) Len() int           { return len(ris) }
func (ris reposByURL) Less(i, j int) bool { return ris[i].url < ris[j].url }
func (ris reposByURL) Swap(i, j int)      { ris[i], ris[j] = ris[j], ris[i] }

/*
data loading (simple lines reader)
*/

func loadUrls() []string {
	return file2lines("data/urls.txt")
}

func file2lines(filePath string) []string {
	f, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if t := scanner.Text(); validURL(t) {
			lines = append(lines, t)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	return lines
}

func validURL(l string) bool {
	return !strings.Contains(l, " ") && len(l) != 0
}

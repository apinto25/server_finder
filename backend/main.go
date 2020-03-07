package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	_ "github.com/lib/pq"

	"github.com/PuerkitoBio/goquery"
	"github.com/buaazp/fasthttprouter"
	"github.com/gocolly/colly"
	"github.com/likexian/whois-go"
	"github.com/valyala/fasthttp"
)

type (
	Server struct {
		Address  string `json:"address"`
		SslGrade string `json:"ssl_grade"`
		Country  string `json:"country"`
		Owner    string `json:"owner"`
	}

	WebPage struct {
		Servers          []Server `json:"servers"`
		ServersChanged   bool     `json:"servers_changed"`
		SslGrade         string   `json:"ssl_grade"`
		PreviousSslGrade string   `json:"previous_ssl_grade"`
		Logo             string   `json:"logo"`
		Title            string   `json:"title"`
		IsDown           bool     `json:"is_down"`
	}

	VisitedURLs struct {
		Items []string `json:items`
	}
)

func WebSearch(ctx *fasthttp.RequestCtx) {

	urlToSearch := string(ctx.QueryArgs().Peek("webURL"))

	if urlToSearch == "" {
		ctx.Error("Web url is null", 500)
		return
	}

	addWebsiteDB(urlToSearch)
	paco := WebScraper(urlToSearch)

	jsonBody, err2 := json.Marshal(paco)

	if err2 != nil {
		ctx.Error(" json marshal fail", 500)
		return
	}

	ctx.SetContentType("application/json; charset=utf-8")
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.SetStatusCode(200)
	ctx.Response.SetBody(jsonBody)
}

func GetWebsites(ctx *fasthttp.RequestCtx) {

	db, err := sql.Open("postgres", "postgresql://maxroach@localhost:26257/websites?sslmode=disable")
	if err != nil {
		log.Fatal("error connecting to the database: ", err)
	}

	rows, err := db.Query("SELECT websiteURL FROM visitedWebsites")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var urls []string // Create an empty nil slice
	// urls = append(urls, "name") // Appends "name" to the slice, creating a new slice if required

	for rows.Next() {
		var web string
		if err := rows.Scan(&web); err != nil {
			log.Fatal(err)
		}
		urls = append(urls, web) // Appends "name" to the slice, creating a new slice if required
	}
	// fmt.Println(urls)

	paco := &VisitedURLs{
		Items: urls}

	jsonBody, err2 := json.Marshal(paco)

	if err2 != nil {
		ctx.Error(" json marshal fail", 500)
		return
	}

	ctx.SetContentType("application/json; charset=utf-8")
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.SetStatusCode(200)
	ctx.Response.SetBody(jsonBody)
}

func WebScraper(urlToSearch string) *WebPage {

	urlData := doRequest("https://api.ssllabs.com/api/v3/analyze?host=" + urlToSearch)

	var urlInfo map[string]interface{}        // Declared an empty interface
	json.Unmarshal([]byte(urlData), &urlInfo) // Unmarshal or Decode the JSON to the interface.

	//host := urlInfo["host"].(string)

	var serverItems []Server
	var grades []string

	endpoints, _ := urlInfo["endpoints"].([]interface{})

	for index, object := range endpoints {

		myMap, _ := object.(map[string]interface{})
		address, _ := myMap["ipAddress"].(string)
		grade, _ := myMap["grade"].(string)
		country, owner := whoisIP(myMap["ipAddress"].(string))

		fmt.Println(index, address, grade, country, owner)

		serverIndex := Server{
			Address:  address,
			SslGrade: grade,
			Country:  country,
			Owner:    owner}
		fmt.Println("server", serverIndex)

		serverItems = append(serverItems, serverIndex)
		grades = append(grades, grade)
		//grades = append(grades, grade)
	}

	fmt.Println("serverItems", serverItems)
	fmt.Println("grades:", grades)

	isDown := false
	status := urlInfo["status"].(string)
	if status == "ERROR" {
		isDown = true
	}

	title := getTitle(urlInfo["host"].(string))
	logo := getLogo(urlInfo["host"].(string))

	//endpoints := urlInfo["endpoints"].([]interface{})
	//fmt.Println("endpoints", endpoints)

	sort.Strings(grades)
	totalGrade := "not found"
	fmt.Println(len(grades), grades)
	if len(grades) > 0 && grades[len(grades)-1] != "" {
		totalGrade = grades[len(grades)-1]
	}

	fmt.Println("sslTotalGrade", totalGrade)

	paco := &WebPage{
		Servers: serverItems,
		//	ServersChanged:   true,
		SslGrade: totalGrade,
		//	PreviousSslGrade: "A+",
		Logo:   logo,
		Title:  title,
		IsDown: isDown}

	return paco
}

func addWebsiteDB(urlToSearch string) {

	db, err := sql.Open("postgres", "postgresql://maxroach@localhost:26257/websites?sslmode=disable")
	if err != nil {
		log.Fatal("error connecting to the database: ", err)
	}

	if _, err := db.Exec(
		"INSERT INTO visitedWebsites (websiteURL) VALUES ($1) ON CONFLICT DO NOTHING", urlToSearch); err != nil {
		log.Fatal(err)
	}
}

func getTitle(url string) string {

	title := "title not found"
	c := colly.NewCollector()
	c.OnHTML("head", func(e *colly.HTMLElement) {
		title = e.ChildText("title")
		fmt.Printf("Title found: %q\n", title)
	})
	c.Visit(url)

	return title
}

func getLogo(url string) string {

	logo := "logo not found"

	c := colly.NewCollector()
	c.OnHTML("head", func(e *colly.HTMLElement) {
		// Extract tags from the document
		tags := e.DOM.Find("link")
		tags.Each(func(_ int, s *goquery.Selection) {
			// Search for rel  tags
			property, _ := s.Attr("rel")
			if strings.EqualFold(property, "shortcut icon") {
				result, _ := s.Attr("href")
				logo = result
				fmt.Printf("Link found: %q", logo)
			}
		})
	})
	c.Visit(url)

	return logo
}

func whoisIP(ipAddress string) (string, string) {

	owner := "owner not found"
	country := "country not found"
	line := ""

	o, _ := regexp.Compile("orgname")
	c, _ := regexp.Compile("country")

	result, err := whois.Whois(ipAddress)
	if err == nil {
		scanner := bufio.NewScanner(strings.NewReader(result))
		for scanner.Scan() {
			if o.MatchString(strings.ToLower(scanner.Text())) {
				line = scanner.Text()
				sliced := strings.Split(line, ":")
				owner = strings.TrimSpace(sliced[1])
			}
			if c.MatchString(strings.ToLower(scanner.Text())) {
				line = scanner.Text()
				sliced := strings.Split(line, ":")
				country = strings.TrimSpace(sliced[1])
			}
		}
	}
	return country, owner
}

func doRequest(url string) string {

	// url = "https://api.ssllabs.com/api/v3/analyze?host=" + url

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	req.SetRequestURI(url)

	fasthttp.Do(req, resp)

	bodyBytes := resp.Body()
	jsonResponse := string(bodyBytes)

	return jsonResponse
}

func main() {
	router := fasthttprouter.New()
	router.GET("/WebSearch", WebSearch)
	router.GET("/visited", GetWebsites)
	log.Fatal(fasthttp.ListenAndServe(":8090", router.Handler))
}

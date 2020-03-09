package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

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
		Items []string `json:"items"`
	}
)

// WebSearch returns a json response converted from a WebPage struct. It
// receives an url to be scrapepd and returns a json response with the WebPage
// struct containing all its information. It first scrap the information using
// using the WebScrapper method, then completes the missing information using
// the calculateDiferences. When complete, converts the WebPage struct to a json
// and returns it as a response to 'localhost:8090/WebPage?webURL='
func WebSearch(ctx *fasthttp.RequestCtx) {

	urlToSearch := string(ctx.QueryArgs().Peek("webURL"))

	if urlToSearch == "" {
		ctx.Error("Web url is null", 500)
		return
	}

	result := WebScraper(urlToSearch)
	result = calculateDiferences(result, urlToSearch)

	jsonBody, err2 := json.Marshal(result)

	if err2 != nil {
		ctx.Error(" json marshal fail", 500)
		return
	}

	ctx.SetContentType("application/json; charset=utf-8")
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.SetStatusCode(200)
	ctx.Response.SetBody(jsonBody)
}

// GetWebsites returns a json listing all the urls that have been searched using
// the WebSearch method. The urls are saved in the database, so it looks for the
// registers in the column websiteURL and saves them one by one in a list, when
// the last register is in the list, converts the list to a json and returns it
// as a response to 'localhost:8090/visited'
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

	var urls []string

	for rows.Next() {
		var web string
		if err := rows.Scan(&web); err != nil {
			log.Fatal(err)
		}
		urls = append(urls, web)
	}

	visitedUrls := &VisitedURLs{
		Items: urls}

	jsonBody, err2 := json.Marshal(visitedUrls)

	if err2 != nil {
		ctx.Error(" json marshal fail", 500)
		return
	}

	ctx.SetContentType("application/json; charset=utf-8")
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.SetStatusCode(200)
	ctx.Response.SetBody(jsonBody)
}

// WebScraper receives a url to be search by the sslabs api, then it creates and
// returns a WebPage map with its information about the servers of a the url to
// be searched, the title and logo url of the url's html, the total ssl grade of
// the url (given its servers) and if the webpage is down.
func WebScraper(urlToSearch string) *WebPage {

	urlData := doRequest("https://api.ssllabs.com/api/v3/analyze?host=" + urlToSearch)

	var urlInfo map[string]interface{}        // Declared an empty interface
	json.Unmarshal([]byte(urlData), &urlInfo) // Unmarshal or Decode the JSON to the interface.

	//host := urlInfo["host"].(string)

	var serverItems []Server
	var grades []string

	endpoints, _ := urlInfo["endpoints"].([]interface{})

	for _, object := range endpoints {

		myMap, _ := object.(map[string]interface{})
		address, _ := myMap["ipAddress"].(string)
		grade, _ := myMap["grade"].(string)
		country, owner := whoisIP(myMap["ipAddress"].(string))

		serverIndex := Server{
			Address:  address,
			SslGrade: grade,
			Country:  country,
			Owner:    owner}

		serverItems = append(serverItems, serverIndex)
		grades = append(grades, grade)
	}

	isDown := false
	status := urlInfo["status"].(string)
	if status == "ERROR" {
		isDown = true
	}

	title := getTitle(urlInfo["host"].(string))
	logo := getLogo(urlInfo["host"].(string))

	sort.Strings(grades)
	totalGrade := "not found"
	if len(grades) > 0 && grades[len(grades)-1] != "" {
		totalGrade = grades[len(grades)-1]
	}

	result := &WebPage{
		Servers: serverItems,
		//	ServersChanged:   false,
		SslGrade: totalGrade,
		//	PreviousSslGrade: "",
		Logo:   logo,
		Title:  title,
		IsDown: isDown}

	return result
}

// calculatesDifferences receives the urlInfo (WebPage struct) and the webUrl.
// It checks in the database the previous ssl grade and the previous information
// about the servers to determine if the servers and the server grade have
// changed when compared to their previous state an hour or more ago. If the url
// has never been searched it saves it as the first register and the previous
// state of ssl grade and the servers will be the actual ones.
// It returns the updated urlInfo (whit the scrapped information from
// webScrapper and the information about its previous state).
func calculateDiferences(urlInfo *WebPage, webUrl string) *WebPage {

	db, err := sql.Open("postgres", "postgresql://maxroach@localhost:26257/websites?sslmode=disable")
	if err != nil {
		log.Fatal("error connecting to the database: ", err)
	}

	rows, err := db.Query(
		`SELECT * FROM visitedWebsites
		WHERE websiteurl = $1`, webUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var id, timestamp int
	var web, sslGrade, prevSslGrade, previousendpoint, actualendpoint string

	strServers, err := json.Marshal(urlInfo.Servers)
	if err != nil {
		panic(err)
	}
	actualServers := string(strServers)

	timestampNow := int(time.Now().Unix())

	for rows.Next() {
		if err := rows.Scan(&id, &web, &timestamp, &prevSslGrade, &sslGrade, &previousendpoint, &actualendpoint); err != nil {
			log.Fatal(err)
		}
	}

	urlInfo.ServersChanged = false

	if id != 0 {
		if (timestampNow - timestamp) > 3600 {
			urlInfo.PreviousSslGrade = sslGrade

			urlInfo.ServersChanged = (actualServers != actualendpoint)

			if _, err := db.Exec(
				`UPDATE visitedWebsites 
				SET (timestamp, previoussslgrade, sslgrade, previousendpoint, actualendpoint) = ($1, $2, $3, $4, $5)
				WHERE websiteurl = $6`, timestampNow, urlInfo.PreviousSslGrade, urlInfo.SslGrade, actualendpoint, actualServers, web); err != nil {
				log.Fatal(err)
			}

		} else {
			urlInfo.PreviousSslGrade = prevSslGrade
			urlInfo.ServersChanged = (actualServers != previousendpoint)
		}
	} else {
		if _, err := db.Exec(
			`INSERT INTO visitedWebsites (websiteurl, timestamp, previoussslgrade, sslgrade,  previousendpoint, actualendpoint)
			VALUES ($1, $2, $3, $4, $5, $6) 
			ON CONFLICT DO NOTHING`, webUrl, timestampNow, urlInfo.SslGrade, urlInfo.SslGrade, actualServers, actualServers); err != nil {
			log.Fatal(err)
		}
	}
	return urlInfo
}

// getTitle returns the title of an html webpage given its url. The title is
// obtained by finding the head tag of the html and then the title tag. The
// given url must start with  http:// or https://, if the title is not found
// getTitle returns the string "title not found".
func getTitle(url string) string {

	title := "title not found"
	c := colly.NewCollector()
	c.OnHTML("head", func(e *colly.HTMLElement) {
		title = e.ChildText("title")
	})
	c.Visit(url)

	return title
}

// getLogo returns the url of the icon logo of an html webpage given its url.
// The title is obtained by finding the head tag of the html and then the link
// tag which has shortcut-icon in the rel attribute. The given url must start
// with  http:// or https://, if the logo is not found getTitle returns the
// string "logo not found".
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
			}
		})
	})
	c.Visit(url)

	return logo
}

// whoisIP receives an ip address and returns its owner and country if found. It
// uses the library whois.Whois which returns all information obtained from the
// ip address. From the obtained information the method look for the lines where
// the parameters orgname (owner of the ip) and country are written. When found,
// sliced it by the colon (separator of the parameter name and the information)
// and trim the spaces to return the found country and owner.
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

// doRequest makes a request to "link de la api", and returns a string with the
// information obtained from the json response.
func doRequest(targetUrl string) string {

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(targetUrl)

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

/*
	The following project is a proof of concept:
	Project Title: go2postman
	Goal or Aim:
	* Take cURL commands in a single file and generate a postman file.
	* Take multiple burp repeater "saved item" files saved in a folder and generate a postman file.
	ToDo:
	- 
	written by Caddyshack2175
		<caddyshack2175@github.com>
*/

package main

/* All imports needed in the main function */
import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"flag"
	"log"
	
	"github.com/google/uuid"
)

/* 
	###################################### START MAIN FUNCTION ########################################################### 
*/
func main() {
	var (
		curlinPtr, burpdirPtr, postmanOutPtr, startbanner string
	)
	startbanner = `	 -=[+] ... Go-2-Postman Postman Generator ... [+]=- `

	// Present operation flags or operation syntax
	flag.StringVar(&curlinPtr, "curl-in", "", "This is to load in a single text file with cURL commands, one per line.")
	flag.StringVar(&curlinPtr, "c", "", "This is to load in a single text file with cURL commands, one per line. (short syntax for -curl-in)")
	// Domain - setup
	flag.StringVar(&burpdirPtr, "burp-dir", "", `This is to load a directory multiple burp repeater "saved item" files saved in a folder and generate a postman file.`)
	flag.StringVar(&burpdirPtr, "b", "", `This is to load a directory multiple burp repeater "saved item" files saved in a folder and generate a postman file. (short syntax for -burp-dir)`)
	// Suffix - setup
	flag.StringVar(&postmanOutPtr, "postman-out", "postman_out.json", `This option is for the generated a postman output file name.`)
	flag.StringVar(&postmanOutPtr, "o", "postman_out.json", `This option is for the generated a postman output file name. (short syntax for -postman-out)`)
	// Parse all the flags
	flag.Usage = func() {
		flagSet := flag.CommandLine
		shorthand := []string{"c", "b", "o"}
		fmt.Printf("\n    	The following syntax is for shorthand operational flags:\n\n")
		for _, name := range shorthand {
			flag := flagSet.Lookup(name)
			fmt.Printf("\t-%s\t | %s\n", flag.Name, flag.Usage)
		}
		longhand := []string{"curl-in", "burp-dir", "postman-out"}
		fmt.Printf("\n    	The following syntax is for longhand operational flags:\n\n")
		for _, name := range longhand {
			flag := flagSet.Lookup(name)
			fmt.Printf("\t-%s\t | %s\n", flag.Name, flag.Usage)
		}
		fmt.Printf("\n    	The following shows examples of tool usage:\n\n")
		fmt.Printf("    	./go2postman -c list-of-curl-commands.txt -o postman-out-collection.json\n")
		fmt.Printf("    	./go2postman -curl-in list-of-curl-commands.txt -postman-out postman-out-collection.json\n")
		fmt.Printf("    	./go2postman -b BURP_XML_FILES/ -postman-out postman-out-collection.json\n")
		fmt.Printf("\n    	** Please note; it is only possible to import a list of commands OR a directory of burp XML files, NOT both! **\n")
		fmt.Printf("\n\n")
	}
	flag.Parse()

	// If no input file or directory, print the banner message
	if (curlinPtr == "") && (burpdirPtr == "") {
		fmt.Printf("\n    	%s\n", startbanner)
		flag.Usage()
		return
	} else if (len(curlinPtr) > 0 ) && (len(burpdirPtr) > 0) {
		fmt.Printf("\n    	%s\n", startbanner)
		flag.Usage()
		return
	}
	
	var collection PostmanCollection
	if (burpdirPtr == "") {
		collection.Info.Name = "cURL API Collection"
		collection.Info.Description = "The POSTMAN file was generated from cURL commands"
	} else {
		collection.Info.Name = "Burp XML API Collection"
		collection.Info.Description = "The POSTMAN file was generated from Burp XML files"
	}

	collection.Info.Schema = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
	collection.Info.PostmanID = uuid.New().String()
	collection.Info.Updated = time.Now()
	
	var outputFile string
	
	if (burpdirPtr != "") {
		log.Printf("	%s\n", startbanner)
		inputDir := burpdirPtr
		outputFile = postmanOutPtr
		
		// Process directory recursively
		err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("[!] Error accessing path %s: %v\n", path, err)
				return err
			}
			
			if info.IsDir() {
				return nil
			}
			
			ext := strings.ToLower(filepath.Ext(path))
			
			switch {
			case ext == ".xml":
				// Check if it's a Burp XML file
				file, err := os.Open(path)
				if err != nil {
					fmt.Printf("[!] Error opening file %s: %v\n", path, err)
					return nil
				}
				
				// Read the first few bytes to check if it looks like a Burp XML file
				buffer := make([]byte, 256)
				_, err = file.Read(buffer)
				file.Close()
				
				if err != nil && err != io.EOF {
					fmt.Printf("[!] Error reading file %s: %v\n", path, err)
					return nil
				}
				
				// Check if it contains Burp XML signature
				content := string(buffer)
				if strings.Contains(content, "<!DOCTYPE items") || strings.Contains(content, "<items burpVersion") {
					fmt.Printf("[+] ... Processing Burp XML file: %s\n", path)
					items, err := ProcessBurpXML(path)
					if err != nil {
						fmt.Printf("[!] Error processing Burp XML file %s: %v\n", path, err)
						return nil
					}
					collection.Item = append(collection.Item, items...)
				}
				
			case ext == ".txt", ext == ".curl":
				// Check if it's a cURL commands file
				file, err := os.Open(path)
				if err != nil {
					fmt.Printf("[!] Error opening file %s: %v\n", path, err)
					return nil
				}
				
				// Read the first line to check if it's a cURL file
				scanner := bufio.NewScanner(file)
				var firstLine string
				if scanner.Scan() {
					firstLine = scanner.Text()
				}
				file.Close()
				
				if err := scanner.Err(); err != nil {
					fmt.Printf("[!] Error reading file %s: %v\n", path, err)
					return nil
				}
				
				if strings.HasPrefix(firstLine, "curl ") {
					fmt.Printf("[+] ... Processing cURL commands file: %s\n", path)
					items, err := ProcessCurlFile(path)
					if err != nil {
						fmt.Printf("[!] Error processing cURL file %s: %v\n", path, err)
						return nil
					}
					collection.Item = append(collection.Item, items...)
				}
			}
			
			return nil
		})
		
		if err != nil {
			fmt.Printf("[!] Error walking directory: %v\n", err)
			return
		}
		
	} else {
		log.Printf("	%s\n", startbanner)
		// Single file mode
		inputFile := curlinPtr
		outputFile = postmanOutPtr
		
		ext := strings.ToLower(filepath.Ext(inputFile))
		
		switch {
		case ext == ".xml":
			fmt.Printf("[+] ... Processing Burp XML file: %s\n", inputFile)
			items, err := ProcessBurpXML(inputFile)
			if err != nil {
				fmt.Printf("[!] Error processing Burp XML file: %v\n", err)
				return
			}
			collection.Item = append(collection.Item, items...)
			
		case ext == ".txt", ext == ".curl", ext == "":
			fmt.Printf("[+] ... Processing cURL commands file: %s\n", inputFile)
			items, err := ProcessCurlFile(inputFile)
			if err != nil {
				fmt.Printf("[!] Error processing cURL file: %v\n", err)
				return
			}
			collection.Item = append(collection.Item, items...)
			
		default:
			fmt.Printf("[!] Unsupported file type: %s\n", ext)
			return
		}
	}
	
	// Check if we found any items
	if len(collection.Item) == 0 {
		fmt.Println("[*] No items were found to convert!")
		return
	}
	
	// Write the collection to the output file
	output, err := json.MarshalIndent(collection, "", "  ")
	if err != nil {
		fmt.Printf("[!] Error marshaling JSON: %v\n", err)
		return
	}
	
	err = os.WriteFile(outputFile, output, 0644)
	if err != nil {
		fmt.Printf("[!] Error writing file: %v\n", err)
		return
	}
	
	fmt.Printf("[+] ... Successfully converted %d requests to Postman collection: %s\n", len(collection.Item), outputFile)
}
/* 
	###################################### END MAIN FUNCTION ########################################################### 
*/
/* 
	################################### FILE PROCESSING FUNCTIONS ######################################################
*/

// ParseCurlCommand parses a cURL command and returns a PostmanItem
func ParseCurlCommand(curlCmd string, index int) (PostmanItem, error) {
	item := PostmanItem{
		Name: fmt.Sprintf("Request %d", index),
	}
	
	// Initialize request structure
	item.Request.Header = []PostmanHeader{}
	item.Request.Body = PostmanBody{
		Mode: "raw",
	}
	
	// Extract method
	methodRegex := regexp.MustCompile(`-X\s+['"]?([A-Z]+)['"]?`)
	methodMatches := methodRegex.FindStringSubmatch(curlCmd)
	if len(methodMatches) > 1 {
		item.Request.Method = methodMatches[1]
	} else {
		item.Request.Method = "GET" // Default method
	}
	
	// Extract URL (look for URL in quotes at the end of the command)
	urlRegex := regexp.MustCompile(`"(https?://[^"]+)"`)
	urlMatches := urlRegex.FindStringSubmatch(curlCmd)
	var urlStr string
	if len(urlMatches) > 1 {
		urlStr = urlMatches[1]
	} else {
		// Try without quotes
		urlRegex = regexp.MustCompile(`\s(https?://\S+)`)
		urlMatches = urlRegex.FindStringSubmatch(curlCmd)
		if len(urlMatches) > 1 {
			urlStr = urlMatches[1]
		}
	}
	
	if urlStr != "" {
		// Parse URL components
		urlParts := strings.Split(urlStr, "://")
		protocol := urlParts[0]
		hostPathQuery := urlParts[1]
		
		// Split host and path+query
		parts := strings.SplitN(hostPathQuery, "/", 2)
		host := parts[0]
		path := ""
		query := []PostmanQueryParam{}
		
		if len(parts) > 1 {
			pathPart := parts[1]
			
			// Handle query parameters if any
			queryParts := strings.SplitN(pathPart, "?", 2)
			path = queryParts[0]
			
			if len(queryParts) > 1 {
				queryStr := queryParts[1]
				queryParams := strings.Split(queryStr, "&")
				
				for _, param := range queryParams {
					kv := strings.SplitN(param, "=", 2)
					queryParam := PostmanQueryParam{
						Key: kv[0],
					}
					if len(kv) > 1 {
						queryParam.Value = kv[1]
					}
					query = append(query, queryParam)
				}
			}
		}
		
		// Extract path components
		pathComponents := []string{}
		if path != "" {
			pathComponents = strings.Split(path, "/")
			// Remove empty components
			cleanPath := []string{}
			for _, p := range pathComponents {
				if p != "" {
					cleanPath = append(cleanPath, p)
				}
			}
			pathComponents = cleanPath
		}
		
		item.Request.URL = PostmanURL{
			Raw:      urlStr,
			Protocol: protocol,
			Host:     strings.Split(host, "."),
			Path:     pathComponents,
			Query:    query,
		}
		
		// Try to extract a better name from the URL
		resourceName := "root"
		if len(pathComponents) > 0 {
			resourceName = pathComponents[len(pathComponents)-1]
		}
		item.Name = fmt.Sprintf("%s %s", item.Request.Method, resourceName)
	}
	
	// Parse headers
	headerRegex := regexp.MustCompile(`-H\s+['"]([^'"]+)['"]`)
	headerMatches := headerRegex.FindAllStringSubmatch(curlCmd, -1)
	
	for _, match := range headerMatches {
		if len(match) > 1 {
			headerLine := match[1]
			parts := strings.SplitN(headerLine, ":", 2)
			
			if len(parts) == 2 {
				header := PostmanHeader{
					Key:   strings.TrimSpace(parts[0]),
					Value: strings.TrimSpace(parts[1]),
					Type:  "text",
				}
				item.Request.Header = append(item.Request.Header, header)
				
				// Check for Authorization header
				if strings.ToLower(header.Key) == "authorization" {
					authValue := header.Value
					if strings.HasPrefix(authValue, "Bearer ") {
						item.Request.Auth = &PostmanAuth{
							Type: "bearer",
							Bearer: []PostmanAuthDetail{
								{
									Key:   "token",
									Value: strings.TrimPrefix(authValue, "Bearer "),
									Type:  "string",
								},
							},
						}
					} else if strings.HasPrefix(authValue, "Basic ") {
						item.Request.Auth = &PostmanAuth{
							Type: "basic",
							Basic: []PostmanAuthDetail{
								{
									Key:   "password",
									Value: strings.TrimPrefix(authValue, "Basic "),
									Type:  "string",
								},
							},
						}
					}
				}
			}
		}
	}
	
	// Parse cookies
	cookieRegex := regexp.MustCompile(`-b\s+['"]([^'"]+)['"]`)
	cookieMatches := cookieRegex.FindStringSubmatch(curlCmd)
	if len(cookieMatches) > 1 && cookieMatches[1] != "" {
		cookieStr := cookieMatches[1]
		header := PostmanHeader{
			Key:   "Cookie",
			Value: cookieStr,
			Type:  "text",
		}
		item.Request.Header = append(item.Request.Header, header)
	}
	
	// Parse data/body
	dataRegexes := []string{
		`-d\s+['"]([^'"]+)['"]`,
		`--data\s+['"]([^'"]+)['"]`,
		`--data-raw\s+['"]([^'"]+)['"]`,
	}
	
	for _, regex := range dataRegexes {
		re := regexp.MustCompile(regex)
		matches := re.FindStringSubmatch(curlCmd)
		if len(matches) > 1 {
			bodyData := matches[1]
			item.Request.Body = PostmanBody{
				Mode: "raw",
				Raw:  bodyData,
				Options: map[string]interface{}{
					"raw": map[string]interface{}{
						"language": "json",
					},
				},
			}
			break
		}
	}
	
	return item, nil
}

// ParseHttpRequest parses an HTTP request string and returns a PostmanItem
func ParseHttpRequest(reqStr string, index int, name string) (PostmanItem, error) {
	item := PostmanItem{
		Name: name,
	}
	
	// Initialize request structure
	item.Request.Header = []PostmanHeader{}
	
	// Split the request into lines
	lines := strings.Split(reqStr, "\n")
	if len(lines) == 0 {
		return item, fmt.Errorf("empty request")
	}
	
	// Parse the first line (method, path, HTTP version)
	firstLine := strings.TrimSpace(lines[0])
	firstLineParts := strings.Split(firstLine, " ")
	if len(firstLineParts) < 2 {
		return item, fmt.Errorf("invalid request line: %s", firstLine)
	}
	
	method := firstLineParts[0]
	path := firstLineParts[1]
	item.Request.Method = method
	
	// Parse headers
	var bodyStartIndex int
	var host string
	
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			bodyStartIndex = i + 1
			break
		}
		
		headerParts := strings.SplitN(line, ":", 2)
		if len(headerParts) == 2 {
			key := strings.TrimSpace(headerParts[0])
			value := strings.TrimSpace(headerParts[1])
			
			header := PostmanHeader{
				Key:   key,
				Value: value,
				Type:  "text",
			}
			item.Request.Header = append(item.Request.Header, header)
			
			if strings.ToLower(key) == "host" {
				host = value
			}
			
			// Check for Authorization header
			if strings.ToLower(key) == "authorization" {
				if strings.HasPrefix(value, "Bearer ") {
					item.Request.Auth = &PostmanAuth{
						Type: "bearer",
						Bearer: []PostmanAuthDetail{
							{
								Key:   "token",
								Value: strings.TrimPrefix(value, "Bearer "),
								Type:  "string",
							},
						},
					}
				} else if strings.HasPrefix(value, "Basic ") {
					item.Request.Auth = &PostmanAuth{
						Type: "basic",
						Basic: []PostmanAuthDetail{
							{
								Key:   "password",
								Value: strings.TrimPrefix(value, "Basic "),
								Type:  "string",
							},
						},
					}
				}
			}
		}
	}
	
	// Extract request body
	var body string
	if bodyStartIndex > 0 && bodyStartIndex < len(lines) {
		body = strings.Join(lines[bodyStartIndex:], "\n")
		if body != "" {
			// Determine content type
			contentType := "text/plain"
			for _, header := range item.Request.Header {
				if strings.ToLower(header.Key) == "content-type" {
					contentType = header.Value
					break
				}
			}
			
			item.Request.Body = PostmanBody{
				Mode: "raw",
				Raw:  body,
			}
			
			// Set language based on content type
			if strings.Contains(contentType, "json") {
				item.Request.Body.Options = map[string]interface{}{
					"raw": map[string]interface{}{
						"language": "json",
					},
				}
			} else if strings.Contains(contentType, "xml") {
				item.Request.Body.Options = map[string]interface{}{
					"raw": map[string]interface{}{
						"language": "xml",
					},
				}
			} else if strings.Contains(contentType, "javascript") {
				item.Request.Body.Options = map[string]interface{}{
					"raw": map[string]interface{}{
						"language": "javascript",
					},
				}
			} else if strings.Contains(contentType, "html") {
				item.Request.Body.Options = map[string]interface{}{
					"raw": map[string]interface{}{
						"language": "html",
					},
				}
			} else {
				item.Request.Body.Options = map[string]interface{}{
					"raw": map[string]interface{}{
						"language": "text",
					},
				}
			}
		}
	}
	
	// Construct URL
	var protocol string
	
	// Check if the request is secure (HTTPS)
	if strings.Contains(path, "https://") {
		protocol = "https"
		path = strings.TrimPrefix(path, "https://")
		// Extract host from URL if present
		parts := strings.SplitN(path, "/", 2)
		if len(parts) > 0 {
			host = parts[0]
			if len(parts) > 1 {
				path = "/" + parts[1]
			} else {
				path = "/"
			}
		}
	} else if strings.Contains(path, "http://") {
		protocol = "http"
		path = strings.TrimPrefix(path, "http://")
		// Extract host from URL if present
		parts := strings.SplitN(path, "/", 2)
		if len(parts) > 0 {
			host = parts[0]
			if len(parts) > 1 {
				path = "/" + parts[1]
			} else {
				path = "/"
			}
		}
	} else {
		// Default to HTTPS if not specified
		protocol = "https"
	}
	
	// Process path and query parameters
	pathAndQuery := strings.SplitN(path, "?", 2)
	path = pathAndQuery[0]
	
	var queryParams []PostmanQueryParam
	if len(pathAndQuery) > 1 {
		queryStr := pathAndQuery[1]
		queryParts := strings.Split(queryStr, "&")
		for _, part := range queryParts {
			kv := strings.SplitN(part, "=", 2)
			param := PostmanQueryParam{
				Key: kv[0],
			}
			if len(kv) > 1 {
				param.Value = kv[1]
			}
			queryParams = append(queryParams, param)
		}
	}
	
	// Process path components
	pathComponents := []string{}
	if path != "" && path != "/" {
		if strings.HasPrefix(path, "/") {
			path = path[1:]
		}
		pathComponents = strings.Split(path, "/")
		// Filter out empty components
		filteredComponents := []string{}
		for _, comp := range pathComponents {
			if comp != "" {
				filteredComponents = append(filteredComponents, comp)
			}
		}
		pathComponents = filteredComponents
	}
	
	// Construct full URL
	fullURL := fmt.Sprintf("%s://%s", protocol, host)
	if len(pathComponents) > 0 {
		fullURL += "/" + strings.Join(pathComponents, "/")
	}
	
	// Add query parameters if any
	if len(queryParams) > 0 {
		queryStrings := []string{}
		for _, param := range queryParams {
			if param.Value != "" {
				queryStrings = append(queryStrings, fmt.Sprintf("%s=%s", param.Key, param.Value))
			} else {
				queryStrings = append(queryStrings, param.Key)
			}
		}
		fullURL += "?" + strings.Join(queryStrings, "&")
	}
	
	item.Request.URL = PostmanURL{
		Raw:      fullURL,
		Protocol: protocol,
		Host:     strings.Split(host, "."),
		Path:     pathComponents,
		Query:    queryParams,
	}
	
	// Set a better name if we can
	if len(pathComponents) > 0 {
		lastComponent := pathComponents[len(pathComponents)-1]
		item.Name = fmt.Sprintf("%s %s", method, lastComponent)
	} else {
		item.Name = fmt.Sprintf("%s %s", method, host)
	}
	
	return item, nil
}

// ProcessBurpXML processes a Burp XML file and returns PostmanItems
func ProcessBurpXML(filePath string) ([]PostmanItem, error) {
	xmlFile, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening Burp XML file: %v", err)
	}
	defer xmlFile.Close()
	
	var burpItems BurpItems
	decoder := xml.NewDecoder(xmlFile)
	if err := decoder.Decode(&burpItems); err != nil {
		return nil, fmt.Errorf("error decoding Burp XML: %v", err)
	}
	
	var items []PostmanItem
	for i, item := range burpItems.Items {
		var reqData string
		
		// Check if the request is base64 encoded
		if item.Request.Base64 == "true" {
			// Decode base64 request
			data, err := base64.StdEncoding.DecodeString(item.Request.Content)
			if err != nil {
				fmt.Printf("Warning: Could not decode base64 request for item %d: %v\n", i+1, err)
				continue
			}
			reqData = string(data)
		} else {
			reqData = item.Request.Content
		}
		
		// Extract the resource name from the path or URL
		resourceName := "request"
		if item.Path != "" {
			pathParts := strings.Split(item.Path, "/")
			if len(pathParts) > 0 && pathParts[len(pathParts)-1] != "" {
				resourceName = pathParts[len(pathParts)-1]
			}
		} else if item.URL != "" {
			urlObj, err := ParseURL(item.URL)
			if err == nil && len(urlObj.Path) > 0 {
				resourceName = urlObj.Path[len(urlObj.Path)-1]
			}
		}
		
		name := fmt.Sprintf("%s %s", item.Method, resourceName)
		postmanItem, err := ParseHttpRequest(reqData, i+1, name)
		if err != nil {
			fmt.Printf("Warning: Could not parse HTTP request for item %d: %v\n", i+1, err)
			continue
		}
		
		items = append(items, postmanItem)
	}
	
	return items, nil
}

// ProcessCurlFile processes a file containing cURL commands and returns PostmanItems
func ProcessCurlFile(filePath string) ([]PostmanItem, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening cURL file: %v", err)
	}
	defer file.Close()
	
	var items []PostmanItem
	scanner := bufio.NewScanner(file)
	index := 1
	
	for scanner.Scan() {
		curlCmd := scanner.Text()
		if strings.HasPrefix(curlCmd, "curl ") {
			item, err := ParseCurlCommand(curlCmd, index)
			if err != nil {
				fmt.Printf("Warning: Could not parse cURL command at line %d: %v\n", index, err)
				continue
			}
			items = append(items, item)
			index++
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading cURL file: %v", err)
	}
	
	return items, nil
}

// ParseURL parses a URL string and returns a PostmanURL
func ParseURL(urlStr string) (PostmanURL, error) {
	var result PostmanURL
	
	// Extract protocol
	urlParts := strings.SplitN(urlStr, "://", 2)
	if len(urlParts) < 2 {
		return result, fmt.Errorf("invalid URL format: %s", urlStr)
	}
	
	protocol := urlParts[0]
	hostPathQuery := urlParts[1]
	
	// Split host and path+query
	parts := strings.SplitN(hostPathQuery, "/", 2)
	host := parts[0]
	path := ""
	
	if len(parts) > 1 {
		path = parts[1]
	}
	
	// Handle query parameters
	pathAndQuery := strings.SplitN(path, "?", 2)
	path = pathAndQuery[0]
	
	var queryParams []PostmanQueryParam
	if len(pathAndQuery) > 1 {
		queryStr := pathAndQuery[1]
		queryParts := strings.Split(queryStr, "&")
		for _, part := range queryParts {
			kv := strings.SplitN(part, "=", 2)
			param := PostmanQueryParam{
				Key: kv[0],
			}
			if len(kv) > 1 {
				param.Value = kv[1]
			}
			queryParams = append(queryParams, param)
		}
	}
	
	// Process path components
	var pathComponents []string
	if path != "" {
		pathComponents = strings.Split(path, "/")
		// Filter empty components
		filteredComponents := []string{}
		for _, comp := range pathComponents {
			if comp != "" {
				filteredComponents = append(filteredComponents, comp)
			}
		}
		pathComponents = filteredComponents
	}
	
	result = PostmanURL{
		Raw:      urlStr,
		Protocol: protocol,
		Host:     strings.Split(host, "."),
		Path:     pathComponents,
		Query:    queryParams,
	}
	
	return result, nil
}
/* 
	####################################### DATA STRUCTURES ############################################################ 
*/
// PostmanCollection represents the structure of a Postman collection
type PostmanCollection struct {
	Info struct {
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Schema      string    `json:"schema"`
		PostmanID   string    `json:"_postman_id"`
		Updated     time.Time `json:"updatedAt"`
	} `json:"info"`
	Item []PostmanItem `json:"item"`
}

// PostmanItem represents a request in the Postman collection
type PostmanItem struct {
	Name    string         `json:"name"`
	Request PostmanRequest `json:"request"`
}

// PostmanRequest represents the request details
type PostmanRequest struct {
	Method string            `json:"method"`
	Header []PostmanHeader   `json:"header"`
	Body   PostmanBody       `json:"body,omitempty"`
	URL    PostmanURL        `json:"url"`
	Auth   *PostmanAuth      `json:"auth,omitempty"`
}

// PostmanHeader represents a header in the request
type PostmanHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

// PostmanQueryParam represents a query parameter
type PostmanQueryParam struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// PostmanBody represents the request body
type PostmanBody struct {
	Mode    string                 `json:"mode"`
	Raw     string                 `json:"raw,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// PostmanURL represents the URL details
type PostmanURL struct {
	Raw      string            `json:"raw"`
	Protocol string            `json:"protocol"`
	Host     []string          `json:"host"`
	Path     []string          `json:"path"`
	Query    []PostmanQueryParam `json:"query,omitempty"`
}

// PostmanAuth represents authentication details
type PostmanAuth struct {
	Type   string              `json:"type"`
	Bearer []PostmanAuthDetail `json:"bearer,omitempty"`
	Basic  []PostmanAuthDetail `json:"basic,omitempty"`
}

// PostmanAuthDetail represents auth details
type PostmanAuthDetail struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

// BurpRequestData represents the request data in Burp XML
type BurpRequestData struct {
	Content  string `xml:",chardata"`
	Base64   string `xml:"base64,attr"`
}

// BurpResponseData represents the response data in Burp XML
type BurpResponseData struct {
	Content  string `xml:",chardata"`
	Base64   string `xml:"base64,attr"`
}

// BurpItem represents an item in Burp XML
type BurpItem struct {
	Time           string          `xml:"time"`
	URL            string          `xml:"url"`
	Host           string          `xml:"host"`
	Port           string          `xml:"port"`
	Protocol       string          `xml:"protocol"`
	Method         string          `xml:"method"`
	Path           string          `xml:"path"`
	Extension      string          `xml:"extension"`
	Request        BurpRequestData `xml:"request"`
	Status         string          `xml:"status"`
	ResponseLength string          `xml:"responselength"`
	MimeType       string          `xml:"mimetype"`
	Response       BurpResponseData `xml:"response"`
	Comment        string          `xml:"comment"`
}

// BurpItems represents the top-level XML structure
type BurpItems struct {
	XMLName     xml.Name   `xml:"items"`
	BurpVersion string     `xml:"burpVersion,attr"`
	ExportTime  string     `xml:"exportTime,attr"`
	Items       []BurpItem `xml:"item"`
}

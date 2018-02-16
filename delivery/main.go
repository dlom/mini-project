package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"

	"github.com/garyburd/redigo/redis"
)

type postback struct {
	Endpoint struct {
		Method string
		URL    string
	}
	Data []map[string]string
}

func main() {
	redisHost := os.Getenv("REDIS_URL")
	if redisHost == "" {
		redisHost = "redis://localhost:6379"
	}
	queueName := os.Getenv("REDIS_QUEUE")
	if queueName == "" {
		queueName = "queue:postback"
	}
	defaultReplacementValue := os.Getenv("DEFAULT_REPLACEMENT_VALUE")

	fmt.Println("Attempting to connect to redis...")
	c, err := redis.DialURL(redisHost)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, "Failed to connect to redis!")
		fmt.Fprintln(os.Stderr, "Retrying... (probably)")
		os.Exit(1)
	}
	fmt.Println("Connected!")
	defer c.Close()

	for {
		err := handlePostback(c, queueName, defaultReplacementValue)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

// handlePostback blocks until an item is available in the redis queue.
// After an item is available in the queue, it is popped off and immediately
// processed.
func handlePostback(c redis.Conn, queueName string, defaultReplacementValue string) error {
	p, err := getPostback(c, queueName)
	if err != nil {
		return err
	}
	return processPostback(p, defaultReplacementValue)
}

// getPostback blocks until an item is available in the redis queue.
// After an item is available in the queue, that item is poppsed off, and
// getPostback returns the data, unmarshalled into the proper data structure.
func getPostback(c redis.Conn, queueName string) (postback, error) {
	var p postback
	bytes, err := redis.ByteSlices(c.Do("BLPOP", queueName, 0))
	if err != nil {
		return p, err
	}
	err = json.Unmarshal(bytes[1], &p)
	return p, err
}

// getPostback takes a raw postback data structure and performs the necessary
// processing to prepare the postback for http requests.  Afterwards, it
// kicks off those http requests.
func processPostback(p postback, defaultReplacementValue string) error {
	valueSet, err := convertPostbackToValues(p, defaultReplacementValue)
	if err != nil {
		return err
	}
	parsedURL, err := url.Parse(p.Endpoint.URL)
	if err != nil {
		return err
	}
	parsedURL.RawQuery = ""
	URL := parsedURL.String()

	for _, values := range valueSet {
		performRequest(p.Endpoint.Method, URL, values)
	}
	return nil
}

// convertPostbackToValues takes the postback data structure and uses the Data
// field and the tokens present in the URL to create an array of url.Values
// that can be converted back into a real querystring, or passed into a call to
// http.postForm().
func convertPostbackToValues(p postback, defaultValue string) ([]url.Values, error) {
	results := make([]url.Values, len(p.Data))

	// Split out querystring from original URL
	parsedURL, err := url.Parse(p.Endpoint.URL)
	if err != nil {
		return results, err
	}
	queryValues, err := url.ParseQuery(parsedURL.RawQuery)
	if err != nil {
		return results, err
	}

	// Extract all key tokens from querystring
	regex := regexp.MustCompile("{(.*?)}")
	keyMap := make(map[string]string)
	for key, value := range queryValues {
		// Assume that all queryValues are unique
		keyMap[key] = regex.FindStringSubmatch(value[0])[1]
	}

	// Loop through data structs and marry the key tokens with the values
	// from the data structs
	for index, data := range p.Data {
		results[index] = make(url.Values)
		for key, replace := range keyMap {
			// Test if the `replace` key exists exists in the data map
			if value, ok := data[replace]; ok {
				results[index].Set(key, value)
			} else {
				results[index].Set(key, defaultValue)
			}
		}
	}

	return results, nil
}

// performRequest performs an http request based on the passed parameters
// and returns the response.
func performRequest(method string, URL string, values url.Values) (*http.Response, error) {
	var response *http.Response
	var err error
	fmt.Printf("%s %s\n", method, URL)
	switch method {
	case "GET":
		response, err = http.Get(URL + "?" + values.Encode())
	case "POST":
		response, err = http.PostForm(URL, values)
	default:
		// Fail silently
		return nil, nil
	}
	return response, err
}

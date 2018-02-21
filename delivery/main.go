package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/garyburd/redigo/redis"
	"go.uber.org/zap"
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

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Attempting to connect to redis...")
	c, err := redis.DialURL(redisHost)
	if err != nil {
		logger.Error("Failed to connect to redis", zap.Error(err))
		logger.Sync()
		os.Exit(1)
	}
	logger.Info("Connected!")
	defer c.Close()

	for {
		err := handlePostback(c, queueName, defaultReplacementValue, logger)
		if err != nil {
			logger.Error("Failed to handle postback", zap.Error(err))
		}
	}
}

// handlePostback blocks until an item is available in the redis queue.
// After an item is available in the queue, it is popped off and immediately
// processed.
func handlePostback(c redis.Conn, queueName string, defaultReplacementValue string, logger *zap.Logger) error {
	p, err := getPostback(c, queueName)
	if err != nil {
		return err
	}
	return processPostback(p, defaultReplacementValue, logger)
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
func processPostback(p postback, defaultReplacementValue string, logger *zap.Logger) error {
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
		go func(values url.Values) {
			start := time.Now()
			response, err := performRequest(p.Endpoint.Method, URL, values)
			elapsed := time.Since(start)

			if err != nil {
				logger.Error("Failed to send request", zap.Error(err))
				return
			}

			defer response.Body.Close()
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				logger.Error("Failed to read response body", zap.Error(err))
				return
			}

			logger.Info("Request sent",
				zap.Time("deliveryTime", start),
				zap.Int("responseCode", response.StatusCode),
				zap.Duration("responseTime", elapsed),
				zap.ByteString("responseBody", body))
		}(values)
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
	replaceKeys := make(map[string]string)
	plainKeys := make(map[string]string)
	for key, value := range queryValues {
		// Assume that all queryValues are unique
		regexResult := regex.FindStringSubmatch(value[0])
		// Test if this is a key that needs replacing
		if len(regexResult) == 2 {
			replaceKeys[key] = regexResult[1]
		} else {
			plainKeys[key] = value[0]
		}
	}

	// Loop through data structs and marry the key tokens with the values
	// from the data structs
	for index, data := range p.Data {
		results[index] = make(url.Values)
		for key, replace := range replaceKeys {
			// Test if the `replace` key exists exists in the data map
			if value, ok := data[replace]; ok {
				results[index].Set(key, value)
			} else {
				results[index].Set(key, defaultValue)
			}
		}
		// Add back on the key/value pairs that didn't need any replacement
		for key, value := range plainKeys {
			results[index].Set(key, value)
		}
	}

	return results, nil
}

// performRequest performs an http request based on the passed parameters
// and returns the response.
func performRequest(method string, URL string, values url.Values) (*http.Response, error) {
	var response *http.Response
	var err error
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

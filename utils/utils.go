package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/sirupsen/logrus"
	"github.developer.allianz.io/global-blockchain-centre-of-competence/ics-lib-go/rpc"
	pb "github.developer.allianz.io/global-blockchain-centre-of-competence/ics-service-foreign-claim-api/api/pb"
	"google.golang.org/grpc/metadata"
)

const (
	// Required URL paths of FC middleware and document middleware
	CreateClaimPath    = "/v1/fcs/claim"
	MutatePropertyPath = "/v1/fcs/claim/child/property"
	MutatePersonPath   = "/v1/fcs/claim/child/person"
	DocUploadPath      = "/v1/document/upload"
	CreateCommentPath  = "/v1/fcs/claim/child/comment"
)

// StatusCodeError represents an http response error.
type StatusCodeError struct {
	Code            int
	Status          string
	ResponseMessage string
	CISLError       CISLError
}

func (t StatusCodeError) Error() string {
	return t.ResponseMessage
}

func (t StatusCodeError) HTTPStatusCode() int {
	return t.Code
}

type CISLError struct {
	ClassID    string `json:"classId"`
	Count      int    `json:"count"`
	Violations []struct {
		ClassID      string `json:"classId"`
		ErrorCode    string `json:"errorCode"`
		Message      string `json:"message"`
		MessageType  string `json:"messageType"`
		PropertyPath string `json:"propertyPath"`
		Severity     string `json:"severity"`
	} `json:"violations"`
}

// checkStatusCode checks the HTTP response status code
func checkStatusCode(resp *http.Response) error {

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logrus.WithError(err).Error("Error retrieving response")
			return err
		}

		var errorResponse CISLError
		err = json.Unmarshal(body, &errorResponse)
		if err != nil || errorResponse.ClassID == "" {
			return StatusCodeError{Code: resp.StatusCode, Status: resp.Status, ResponseMessage: string(body)}
		}

		if len(errorResponse.Violations) > 0 {
			combinedErrorsMessage := ""
			combinedPropertyPath := ""
			for i, violation := range errorResponse.Violations {
				combinedPropertyPath += fmt.Sprintf("%d: %s\n", i+1, violation.PropertyPath)
				combinedErrorsMessage += fmt.Sprintf("%d: %s\n", i+1, violation.Message)

			}
			logrus.WithFields(nil).Errorf("Request failed")
			errorMessage := fmt.Sprintf("violationPropertyPaths={%v}\n\n\n\nviolationMessages={%v}", combinedPropertyPath, combinedErrorsMessage)
			errorMessage = strings.ReplaceAll(errorMessage, "\n", " ")
			return StatusCodeError{Code: resp.StatusCode, Status: resp.Status, ResponseMessage: errorMessage}
		}

	}
	return nil

}

// performRequest performs an HTTP request with the given request parameters and response parser.
func performRequest(ctx context.Context, client *http.Client, req *http.Request, parser responseParser) error {
	req = req.WithContext(ctx)
	start := time.Now()
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	err = checkStatusCode(response)
	if err != nil {
		return err
	}
	defer func() {
		entry := logrus.WithFields(logrus.Fields{
			"method":       req.Method,
			"duration":     time.Since(start),
			"host":         req.Host,
			"path":         req.URL.Path,
			"responseCode": response.StatusCode,
		})
		entry.Info("Request handled")
	}()
	parser(response)
	return nil
}

// PostJSONResource sends a POST request to the specified endpoint with the provided JSON payload.
func PostJSONResource(ctx context.Context, client *http.Client, endpoint string, values url.Values, token string, json []byte, result interface{}) error {
	if values != nil {
		endpoint = endpoint + "?" + values.Encode()
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(json))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	return performRequest(ctx, client, req, newJSONParser(result))
}

// PostFormData sends a POST request to the specified endpoint with the provided JSON payload.
func PostFormData(ctx context.Context, client *http.Client, endpoint string, token string, formData url.Values, json []byte, result interface{}) error {
	formEncoded := formData.Encode()
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(formEncoded))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	return performRequest(ctx, client, req, newJSONParser(result))
}

// FetchResource fetches a resource using HTTP GET request to the specified endpoint.
func FetchResource(ctx context.Context, client *http.Client, endpoint string, values url.Values, token string, result interface{}) error {
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.URL.RawQuery = values.Encode()
	req = req.WithContext(ctx)
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	err = checkStatusCode(response)
	if err != nil {
		return err
	}
	return performRequest(ctx, client, req, newJSONParser(result))
}

func PutJSONResource(ctx context.Context, client *http.Client, endpoint string, params url.Values, json []byte, token string, result interface{}) error {
	req, err := http.NewRequest("PUT", endpoint, bytes.NewBuffer(json))
	if err != nil {
		return err
	}
	if params != nil {
		req.URL.RawQuery = params.Encode()
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	return performRequest(ctx, client, req, newJSONParser(result))
}

func DeleteJSONResource(ctx context.Context, client *http.Client, endpoint string, params url.Values, json []byte, token string, result interface{}) error {
	req, err := http.NewRequest("DELETE", endpoint, bytes.NewBuffer(json))
	if err != nil {
		return err
	}
	if params != nil {
		req.URL.RawQuery = params.Encode()
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	return performRequest(ctx, client, req, newJSONParser(result))
}

type responseParser func(*http.Response) error

// newJSONParser returns a responseParser function that parses the HTTP response body as JSON and
// unmarshals it into the provided destination struct.
func newJSONParser(dst interface{}) responseParser {
	return func(resp *http.Response) error {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		err = json.Unmarshal(body, &dst)
		if err != nil {
			return err
		}
		return nil
	}
}

func IsExpired(creationTime time.Time, expirationTimeInSeconds int64) bool {
	expirationDuration := time.Duration(expirationTimeInSeconds) * time.Second
	expirationTime := creationTime.Add(expirationDuration)
	return time.Now().After(expirationTime)
}

// RFC3339DateOnly is "2006-01-02"
const RFC3339DateOnly = "2006-01-02"

func TimestampDateOfString(x string) (time.Time, error) {
	ts, err := time.Parse(RFC3339DateOnly, x)
	if err != nil {
		return time.Time{}, err
	}
	return ts, nil
}

func ReverseMapping(dataMappings map[string]map[string]string) map[string]map[string]string {
	reverseMap := make(map[string]map[string]string)

	for key, value := range dataMappings {
		for value, reverseKey := range value {
			if reverseMap[reverseKey] == nil {
				reverseMap[reverseKey] = make(map[string]string)
			}
			reverseMap[reverseKey][key] = value
		}
	}

	return reverseMap
}

// FCSRequestBuilder identifies, builds and triggers post requests to FC middleware or Doc middleware
func FCSRequestBuilder(ctx context.Context, client *rpc.Client, object interface{}, opts ...interface{}) (interface{}, error) {

	switch value := object.(type) {
	case *pb.Property:
		var mutatePropertyResp pb.MutateClaimPropertyResponse

		//construct a property
		propertyReq, err := json.Marshal(&struct {
			*pb.Property
		}{
			Property: value,
		})
		if err != nil {
			return "", err
		}

		if err = client.Call(ctx, "PATCH", MutatePropertyPath, string(propertyReq), &mutatePropertyResp); err != nil {
			return mutatePropertyResp, err
		}

		return mutatePropertyResp, nil

	case *pb.Person:

		var mutatePersonResp pb.MutateClaimPersonResponse

		//construct a person
		mutatePersonJsonReq, err := json.Marshal(&struct {
			*pb.Person
		}{
			Person: value,
		})
		if err != nil {
			return "", err
		}
		// call to fcs mutate person
		if err = client.Call(ctx, "PATCH", MutatePersonPath, string(mutatePersonJsonReq), &mutatePersonResp); err != nil {
			return mutatePersonResp, err
		}
		return mutatePersonResp, nil
	case *pb.Claim:

		claimResponse := pb.CreateClaimResponse{}

		//Mutate Claim
		if value.GetIcsRefNo() != "" {

			mutateClaimJsonReq, err := json.Marshal(&pb.MutateClaimRequest{
				Claim: value,
			})
			if err != nil {
				return nil, err
			}
			err = client.Call(ctx, "PATCH", CreateClaimPath+"/"+value.IcsRefNo, string(mutateClaimJsonReq), &claimResponse)
			if err != nil {
				return claimResponse, err
			}
			return claimResponse, nil

		}

		createClaimJsonReq, err := json.Marshal(&struct {
			*pb.Claim
		}{
			Claim: value,
		})

		if err != nil {
			return "", err
		}
		err = client.Call(ctx, "POST", CreateClaimPath, string(createClaimJsonReq), &claimResponse)
		if err != nil {
			return claimResponse, err
		}
		return claimResponse, nil
	case *pb.Comment:
		var createClaimCommentResponse pb.CreateClaimCommentResponse
		createCommentJsonReq, err :=
			json.Marshal(&pb.CreateClaimCommentRequest{
				Comment: value,
			})
		if err != nil {
			return "", err
		}
		if err = client.Call(ctx, "POST", CreateCommentPath, string(createCommentJsonReq), &createClaimCommentResponse); err != nil {
			return "", err
		}
		return createClaimCommentResponse.GetComment().GetCommentId(), nil

	}
	return nil, nil
}

// ConvertToJSON converts data to JSON and then writes to HTTP response writer
func ConvertToJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		fmt.Fprintf(w, "%s", err.Error())
	}
}

// ERROR writes the error to HTTP response writer
func ERROR(w http.ResponseWriter, statusCode int, err error) {
	if err != nil {
		ConvertToJSON(w, statusCode, struct {
			Error string `json:"error"`
		}{
			Error: err.Error(),
		})
		entry := logrus.WithFields(logrus.Fields{
			"error message": err.Error(),
		})
		entry.Error("CISL adapter response error")
		return
	}
	ConvertToJSON(w, statusCode, nil)
}

// createAuthContext create a context with the JWT token in the request header
func CreateAuthContext(authToken string) context.Context {
	ctx := context.Background()
	header := metadata.New(map[string]string{
		"cookie": authToken,
	})
	ctx = metadata.NewIncomingContext(ctx, header)
	return ctx
}

func ReverseString(input string) string {
	runes := []rune(input)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func InsertRune(input string, runeOption rune) string {
	var searchString strings.Builder
	inserted := false

	for _, char := range input {
		if unicode.IsDigit(char) && !inserted {
			searchString.WriteRune(runeOption)
			inserted = true
		}
		searchString.WriteRune(char)
	}
	return searchString.String()
}

package commercetools_test

import (
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"testing"

	"github.com/labd/commercetools-go-sdk/commercetools"
	"github.com/labd/commercetools-go-sdk/testutil"
	"github.com/stretchr/testify/assert"
)

type OutputData struct{}

func errorHandler(statusCode int, returnValue string, encoding string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(returnValue))
	}
}

func TestClientGetBadRequestJson(t *testing.T) {
	body := `,`
	client, server := testutil.MockClient(
		t, "", nil, errorHandler(http.StatusBadRequest, body, "application/json"))
	defer server.Close()

	output := OutputData{}

	err := client.Get("/", nil, &output)
	assert.Equal(t, "invalid character ',' looking for beginning of value", err.Error())
}

func TestClientNotFound(t *testing.T) {
	body := ``
	client, server := testutil.MockClient(
		t, "", nil, errorHandler(http.StatusNotFound, body, "application/json"))
	defer server.Close()

	output := OutputData{}

	err := client.Get("/", nil, &output)
	assert.Equal(t, "Not Found (404): ResourceNotFound", err.Error())

	ctErr, ok := err.(commercetools.ErrorResponse)
	assert.Equal(t, true, ok)
	assert.Equal(t, 404, ctErr.StatusCode)
}

func TestAuthError(t *testing.T) {
	body := `{
		"statusCode": 403,
		"message": "Insufficient scope",
		"errors": [
			{
				"code": "insufficient_scope",
				"message": "Insufficient scope"
			}
		],
		"error": "insufficient_scope",
		"error_description": "Insufficient scope"
	}`
	client, server := testutil.MockClient(
		t, "", nil, errorHandler(http.StatusForbidden, body, "application/json"))

	defer server.Close()

	output := OutputData{}

	err := client.Get("/", nil, &output)

	assert.Equal(t, "Insufficient scope", err.Error())

	ctErr, ok := err.(commercetools.ErrorResponse)
	assert.Equal(t, true, ok)

	ctChildErr, ok := ctErr.Errors[0].(commercetools.InsufficientScopeError)
	assert.Equal(t, true, ok)
	assert.Equal(t, "Insufficient scope", ctChildErr.Message)
}

func TestInvalidJsonError(t *testing.T) {
	body := `{
		"statusCode": 400,
		"message": "Request body does not contain valid JSON.",
		"errors": [
			{
				"code": "InvalidJsonInput",
				"message": "Request body does not contain valid JSON.",
				"detailedErrorMessage": "No content to map due to end-of-input"
			}
		]
	}`
	client, server := testutil.MockClient(
		t, "", nil, errorHandler(http.StatusBadRequest, body, "application/json"))

	defer server.Close()

	output := OutputData{}

	err := client.Get("/", nil, &output)

	assert.Equal(t, "Request body does not contain valid JSON.", err.Error())

	ctErr, ok := err.(commercetools.ErrorResponse)
	assert.Equal(t, true, ok)

	ctChildErr, ok := ctErr.Errors[0].(commercetools.InvalidJSONInputError)
	assert.Equal(t, "Request body does not contain valid JSON.", ctChildErr.Error())
	// assert.Equal(t, commercetools.ErrInvalidJSONInput, ctErr.Errors[0].Code())
}

func TestQueryInput(t *testing.T) {
	tr := true
	fa := false
	testCases := []struct {
		desc     string
		input    *commercetools.QueryInput
		query    url.Values
		rawQuery string
	}{
		{
			desc: "Where",
			input: &commercetools.QueryInput{
				Where: "not (name = 'Peter' and age < 42)",
			},
			query: url.Values{
				"where": []string{"not (name = 'Peter' and age < 42)"},
			},
			rawQuery: "where=not+%28name+%3D+%27Peter%27+and+age+%3C+42%29",
		},
		{
			desc: "Sort",
			input: &commercetools.QueryInput{
				Sort: []string{"name desc", "dog.age asc"},
			},
			query: url.Values{
				"sort": []string{"name desc", "dog.age asc"},
			},
			rawQuery: "sort=name+desc&sort=dog.age+asc",
		},
		{
			desc: "Expand",
			input: &commercetools.QueryInput{
				Expand: "taxCategory",
			},
			query: url.Values{
				"expand": []string{"taxCategory"},
			},
			rawQuery: "expand=taxCategory",
		},
		{
			desc: "Limit",
			input: &commercetools.QueryInput{
				Limit: 20,
			},
			query: url.Values{
				"limit": []string{"20"},
			},
			rawQuery: "limit=20",
		},
		{
			desc: "Offset",
			input: &commercetools.QueryInput{
				Offset: 20,
			},
			query: url.Values{
				"offset": []string{"20"},
			},
			rawQuery: "offset=20",
		},
		{
			desc: "WithTotal False",
			input: &commercetools.QueryInput{
				WithTotal: &fa,
			},
			query: url.Values{
				"withTotal": []string{"false"},
			},
			rawQuery: "withTotal=false",
		},
		{
			desc: "WithTotal True",
			input: &commercetools.QueryInput{
				WithTotal: &tr,
			},
			query: url.Values{
				"withTotal": []string{"true"},
			},
			rawQuery: "withTotal=true",
		},
		{
			desc: "WithTotal nil",
			input: &commercetools.QueryInput{
				WithTotal: nil,
			},
			query:    url.Values{},
			rawQuery: "",
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			output := testutil.RequestData{}

			client, server := testutil.MockClient(t, "{}", &output, nil)
			defer server.Close()

			_, err := client.TaxCategoryQuery(tC.input)

			assert.Nil(t, err)
			assert.Equal(t, tC.query, output.URL.Query())
			assert.Equal(t, tC.rawQuery, output.URL.RawQuery)
		})
	}
}

func TestUserAgents(t *testing.T) {
	testCases := []struct {
		cfg               *commercetools.Config
		expectedUserAgent string
	}{
		{
			cfg: &commercetools.Config{
				LibraryName:    "terraform-provider-commercetools",
				LibraryVersion: "0.1",
				ContactURL:     "https://example.com",
				ContactEmail:   "test@example.org",
			},
			expectedUserAgent: fmt.Sprintf("commercetools-go-sdk/1.0.0 Go/%s (%s; %s) terraform-provider-commercetools/0.1 (+https://example.com; +test@example.org)", runtime.Version(), runtime.GOOS, runtime.GOARCH),
		},
		{
			cfg: &commercetools.Config{
				ContactURL:   "https://example.com",
				ContactEmail: "test@example.org",
			},
			expectedUserAgent: fmt.Sprintf("commercetools-go-sdk/1.0.0 Go/%s (%s; %s) (+https://example.com; +test@example.org)", runtime.Version(), runtime.GOOS, runtime.GOARCH),
		},
		{
			cfg: &commercetools.Config{
				LibraryName:    "terraform-provider-commercetools",
				LibraryVersion: "0.1",
				ContactEmail:   "test@example.org",
			},
			expectedUserAgent: fmt.Sprintf("commercetools-go-sdk/1.0.0 Go/%s (%s; %s) terraform-provider-commercetools/0.1 (+test@example.org)", runtime.Version(), runtime.GOOS, runtime.GOARCH),
		},
		{
			cfg: &commercetools.Config{
				LibraryName:  "terraform-provider-commercetools",
				ContactURL:   "https://example.com",
				ContactEmail: "test@example.org",
			},
			expectedUserAgent: fmt.Sprintf("commercetools-go-sdk/1.0.0 Go/%s (%s; %s) terraform-provider-commercetools (+https://example.com; +test@example.org)", runtime.Version(), runtime.GOOS, runtime.GOARCH),
		},
	}

	for _, tC := range testCases {
		t.Run("Test user agent", func(t *testing.T) {
			userAgent := commercetools.GetUserAgent(tC.cfg)
			assert.Equal(t, tC.expectedUserAgent, userAgent)
		})
	}
}

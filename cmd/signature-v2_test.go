package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"testing"
	"time"
)

// Tests for 'func TestResourceListSorting(t *testing.T)'.
func TestResourceListSorting(t *testing.T) {
	sortedResourceList := make([]string, len(resourceList))
	copy(sortedResourceList, resourceList)
	sort.Strings(sortedResourceList)
	for i := 0; i < len(resourceList); i++ {
		if resourceList[i] != sortedResourceList[i] {
			t.Errorf("Expected resourceList[%d] = \"%s\", resourceList is not correctly sorted.", i, sortedResourceList[i])
			break
		}
	}
}

// Tests presigned v2 signature.
func TestDoesPresignedV2SignatureMatch(t *testing.T) {
	root, err := newTestConfig(globalMinioDefaultRegion)
	if err != nil {
		t.Fatal("Unable to initialize test config.")
	}
	defer removeAll(root)

	now := time.Now().UTC()

	testCases := []struct {
		queryParams map[string]string
		headers     map[string]string
		expected    APIErrorCode
	}{
		// (0) Should error without a set URL query.
		{
			expected: ErrInvalidQueryParams,
		},
		// (1) Should error on an invalid access key.
		{
			queryParams: map[string]string{
				"Expires":        "60",
				"Signature":      "badsignature",
				"AWSAccessKeyId": "Z7IXGOO6BZ0REAN1Q26I",
			},
			expected: ErrInvalidAccessKeyID,
		},
		// (2) Should error with malformed expires.
		{
			queryParams: map[string]string{
				"Expires":        "60s",
				"Signature":      "badsignature",
				"AWSAccessKeyId": serverConfig.GetCredential().AccessKey,
			},
			expected: ErrMalformedExpires,
		},
		// (3) Should give an expired request if it has expired.
		{
			queryParams: map[string]string{
				"Expires":        "60",
				"Signature":      "badsignature",
				"AWSAccessKeyId": serverConfig.GetCredential().AccessKey,
			},
			expected: ErrExpiredPresignRequest,
		},
		// (4) Should error when the signature does not match.
		{
			queryParams: map[string]string{
				"Expires":        fmt.Sprintf("%d", now.Unix()+60),
				"Signature":      "badsignature",
				"AWSAccessKeyId": serverConfig.GetCredential().AccessKey,
			},
			expected: ErrSignatureDoesNotMatch,
		},
		// (5) Should error when the signature does not match.
		{
			queryParams: map[string]string{
				"Expires":        fmt.Sprintf("%d", now.Unix()+60),
				"Signature":      "zOM2YrY/yAQe15VWmT78OlBrK6g=",
				"AWSAccessKeyId": serverConfig.GetCredential().AccessKey,
			},
			expected: ErrSignatureDoesNotMatch,
		},
	}

	// Run each test case individually.
	for i, testCase := range testCases {
		// Turn the map[string]string into map[string][]string, because Go.
		query := url.Values{}
		for key, value := range testCase.queryParams {
			query.Set(key, value)
		}

		// Create a request to use.
		req, e := http.NewRequest(http.MethodGet, "http://host/a/b?"+query.Encode(), nil)
		if e != nil {
			t.Errorf("(%d) failed to create http.Request, got %v", i, e)
		}
		// Should be set since we are simulating a http server.
		req.RequestURI = req.URL.RequestURI()

		// Do the same for the headers.
		for key, value := range testCase.headers {
			req.Header.Set(key, value)
		}

		// Check if it matches!
		err := doesPresignV2SignatureMatch(req)
		if err != testCase.expected {
			t.Errorf("(%d) expected to get %s, instead got %s", i, niceError(testCase.expected), niceError(err))
		}
	}
}

// TestValidateV2AuthHeader - Tests validate the logic of V2 Authorization header validator.
func TestValidateV2AuthHeader(t *testing.T) {
	// Initialize server config.
	if _, err := initConfig(); err != nil {
		t.Fatal(err)
	}

	// Save config.
	if err := serverConfig.Save(); err != nil {
		t.Fatal(err)
	}
	accessID := serverConfig.GetCredential().AccessKey

	testCases := []struct {
		authString    string
		expectedError APIErrorCode
	}{
		// Test case - 1.
		// Case with empty V2AuthString.
		{

			authString:    "",
			expectedError: ErrAuthHeaderEmpty,
		},
		// Test case - 2.
		// Test case with `signV2Algorithm` ("AWS") not being the prefix.
		{

			authString:    "NoV2Prefix",
			expectedError: ErrSignatureVersionNotSupported,
		},
		// Test case - 3.
		// Test case with missing parts in the Auth string.
		// below is the correct format of V2 Authorization header.
		// Authorization = "AWS" + " " + AWSAccessKeyId + ":" + Signature
		{

			authString:    signV2Algorithm,
			expectedError: ErrMissingFields,
		},
		// Test case - 4.
		// Test case with signature part missing.
		{

			authString:    fmt.Sprintf("%s %s", signV2Algorithm, accessID),
			expectedError: ErrMissingFields,
		},
		// Test case - 5.
		// Test case with wrong accessID.
		{

			authString:    fmt.Sprintf("%s %s:%s", signV2Algorithm, "InvalidAccessID", "signature"),
			expectedError: ErrInvalidAccessKeyID,
		},
		// Test case - 6.
		// Case with right accessID and format.
		{

			authString:    fmt.Sprintf("%s %s:%s", signV2Algorithm, accessID, "signature"),
			expectedError: ErrNone,
		},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("Case %d AuthStr \"%s\".", i+1, testCase.authString), func(t *testing.T) {

			actualErrCode := validateV2AuthHeader(testCase.authString)

			if testCase.expectedError != actualErrCode {
				t.Errorf("Expected the error code to be %v, got %v.", testCase.expectedError, actualErrCode)
			}
		})
	}

}

func TestDoesPolicySignatureV2Match(t *testing.T) {
	if _, err := initConfig(); err != nil {
		t.Fatal(err)
	}

	if err := serverConfig.Save(); err != nil {
		t.Fatal(err)
	}
	creds := serverConfig.GetCredential()
	policy := "policy"
	testCases := []struct {
		accessKey string
		policy    string
		signature string
		errCode   APIErrorCode
	}{
		{"invalidAccessKey", policy, calculateSignatureV2(policy, creds.SecretKey), ErrInvalidAccessKeyID},
		{creds.AccessKey, policy, calculateSignatureV2("random", creds.SecretKey), ErrSignatureDoesNotMatch},
		{creds.AccessKey, policy, calculateSignatureV2(policy, creds.SecretKey), ErrNone},
	}
	for i, test := range testCases {
		formValues := make(map[string]string)
		formValues["Awsaccesskeyid"] = test.accessKey
		formValues["Signature"] = test.signature
		formValues["Policy"] = test.policy
		errCode := doesPolicySignatureV2Match(formValues)
		if errCode != test.errCode {
			t.Fatalf("(%d) expected to get %s, instead got %s", i+1, niceError(test.errCode), niceError(errCode))
		}
	}
}
